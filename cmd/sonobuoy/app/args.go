/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/image"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	ops "github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	v1 "k8s.io/api/core/v1"
)

const (
	namespaceFlag       = "namespace"
	sonobuoyImageFlag   = "sonobuoy-image"
	imagePullPolicyFlag = "image-pull-policy"
	pluginFlag          = "plugin"
	timeoutFlag         = "timeout"
	waitOutputFlag      = "wait-output"
	customRegistryFlag  = "custom-registry"
	kubeconfig          = "kubeconfig"
	kubecontext         = "context"
)

// AddNamespaceFlag initialises a namespace flag.
func AddNamespaceFlag(str *string, flags *pflag.FlagSet) {
	flags.StringVarP(
		str, namespaceFlag, "n", config.DefaultNamespace,
		"The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously.",
	)
}

// AddDNSNamespaceFlag initialises the dns-namespace flag.
// The value of this flag is used during preflight checks to determine which namespace to use
// when looking for the DNS pods.
func AddDNSNamespaceFlag(str *string, flags *pflag.FlagSet) {
	flags.StringVar(
		str, "dns-namespace", config.DefaultDNSNamespace,
		"The namespace to check for DNS pods during preflight checks.",
	)
}

// AddDNSPodLabelsFlag initialises the dns-pod-labels flag.
// The value of this flag is used during preflight checks to determine which labels to use
// when looking for the DNS pods.
func AddDNSPodLabelsFlag(str *[]string, flags *pflag.FlagSet) {
	flags.StringSliceVar(
		str, "dns-pod-labels", config.DefaultDNSPodLabels,
		"The label selectors to use for locating DNS pods during preflight checks. Can be specified multiple times or as a comma-separated list.",
	)
}

// AddModeFlag initialises a mode flag.
// The mode is a preset configuration of sonobuoy configuration and e2e configuration variables.
// Mode can be partially or fully overridden by specifying config, e2e-focus, and e2e-skip.
// The variables specified by those flags will overlay the defaults provided by the given mode.
func AddModeFlag(mode *ops.Mode, flags *pflag.FlagSet) {
	*mode = ops.NonDisruptiveConformance // default
	flags.VarP(
		mode, "mode", "m",
		fmt.Sprintf("What mode to run sonobuoy in. Valid modes are %s.", strings.Join(ops.GetModes(), ", ")),
	)
}

// AddSonobuoyImage initialises an image url flag.
func AddSonobuoyImage(image *string, flags *pflag.FlagSet) {
	flags.StringVar(
		image, sonobuoyImageFlag, config.DefaultImage,
		"Container image override for the sonobuoy worker and container.",
	)
}

// AddKubeConformanceImage initialises an image url flag.
func AddKubeConformanceImage(image *string, flags *pflag.FlagSet) {
	flags.StringVar(
		image, "kube-conformance-image", "",
		"Container image override for the kube conformance image. Overrides --kube-conformance-image-version.",
	)
}

// AddSystemdLogsImage initialises the systemd-logs-image flag.
func AddSystemdLogsImage(image *string, flags *pflag.FlagSet) {
	flags.StringVar(
		image, "systemd-logs-image", config.DefaultSystemdLogsImage,
		"Container image override for the systemd-logs plugin image.",
	)
}

// AddKubeConformanceImageVersion initialises an image version flag.
func AddKubeConformanceImageVersion(imageVersion *image.ConformanceImageVersion, flags *pflag.FlagSet) {
	help := "Use default Conformance image, but override the version. "
	help += "Default is 'auto', which will be set to your cluster's version if detected, erroring otherwise."
	help += "You can also choose 'latest' which will find the latest dev image upstream."

	*imageVersion = image.ConformanceImageVersionAuto
	flags.Var(imageVersion, "kube-conformance-image-version", help)
}

// AddKubeconfigFlag adds a kubeconfig and context flags to the provided command.
func AddKubeconfigFlag(cfg *Kubeconfig, flags *pflag.FlagSet) {
	// The default is the empty string (look in the environment)
	flags.Var(cfg, "kubeconfig", "Path to explicit kubeconfig file.")
	flags.StringVar(&cfg.Context, kubecontext, "", "Context in the kubeconfig to use.")
}

// AddE2ERegistryConfigFlag adds a e2eRegistryConfigFlag flag to the provided command.
func AddE2ERegistryConfigFlag(cfg *string, flags *pflag.FlagSet) {
	flags.StringVar(
		cfg, e2eRegistryConfigFlag, "",
		"Specify a yaml file acting as KUBE_TEST_REPO_LIST, overriding registries for test images. Required when pushing images for the e2e plugin.",
	)
}

// AddCustomRepoFlag adds a custom registry flag to the provided command.
func AddCustomRegistryFlag(cfg *string, flags *pflag.FlagSet) {
	flags.StringVar(
		cfg, customRegistryFlag, "",
		"Specify a registry to override the Sonobuoy and Plugin image registries.",
	)
}

// AddSonobuoyConfigFlag adds a SonobuoyConfig flag to the provided command.
func AddSonobuoyConfigFlag(cfg *SonobuoyConfig, flags *pflag.FlagSet) {
	flags.Var(
		cfg, "config",
		"Path to a sonobuoy configuration JSON file. Overrides --mode.",
	)
}

const (
	e2eFocusFlag          = "e2e-focus"
	e2eSkipFlag           = "e2e-skip"
	e2eParallelFlag       = "e2e-parallel"
	e2eRegistryConfigFlag = "e2e-repo-config"
)

// AddE2EConfigFlags adds three arguments: --e2e-focus, --e2e-skip and
// --e2e-parallel. These are not taken as pointers, as they are only used by
// GetE2EConfig. Instead, they are returned as a Flagset which should be passed
// to GetE2EConfig. The returned flagset will be added to the passed in flag set.
//
// e2e-parallel is added as a hidden flag that should only be used by "power"
// users. Using e2e-parallel incorrectly has the potential to destroy clusters!
func AddE2EConfigFlags(flags *pflag.FlagSet) *pflag.FlagSet {
	e2eFlags := pflag.NewFlagSet("e2e", pflag.ExitOnError)
	modeName := ops.NonDisruptiveConformance
	defaultMode := modeName.Get()
	e2eFlags.String(
		e2eFocusFlag, defaultMode.E2EConfig.Focus,
		"Specify the E2E_FOCUS flag to the conformance tests. Overrides --mode.",
	)
	e2eFlags.String(
		e2eSkipFlag, defaultMode.E2EConfig.Skip,
		"Specify the E2E_SKIP flag to the conformance tests. Overrides --mode.",
	)
	e2eFlags.String(
		e2eParallelFlag, defaultMode.E2EConfig.Parallel,
		"Specify the E2E_PARALLEL flag to the conformance tests. Overrides --mode.",
	)
	e2eFlags.String(
		e2eRegistryConfigFlag, "",
		"Specify a yaml file acting as KUBE_TEST_REPO_LIST, overriding registries for test images.",
	)
	e2eFlags.MarkHidden(e2eParallelFlag)
	flags.AddFlagSet(e2eFlags)
	return e2eFlags
}

// GetE2EConfig gets the E2EConfig from the mode, then overrides them with e2e-focus, e2e-skip and e2e-parallel if they
// are provided. We can't rely on the zero value of the flags, as "" is a valid focus, skip or parallel value.
func GetE2EConfig(mode ops.Mode, flags *pflag.FlagSet) (*ops.E2EConfig, error) {
	cfg := mode.Get().E2EConfig
	if flags.Changed(e2eFocusFlag) {
		focus, err := flags.GetString(e2eFocusFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve focus flag")
		}
		if _, err := regexp.Compile(focus); err != nil {
			return nil, errors.Wrap(err, "focus flag fails regexp validation")
		}
		cfg.Focus = focus
	}

	if flags.Changed(e2eSkipFlag) {
		skip, err := flags.GetString(e2eSkipFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve skip flag")
		}
		if _, err := regexp.Compile(skip); err != nil {
			return nil, errors.Wrap(err, "skip flag fails regexp validation")
		}
		cfg.Skip = skip
	}

	if flags.Changed(e2eParallelFlag) {
		parallel, err := flags.GetString(e2eParallelFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve parallel flag")
		}
		cfg.Parallel = parallel
	}

	if flags.Changed(e2eRegistryConfigFlag) {
		repoFile, err := flags.GetString(e2eRegistryConfigFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve registry list flag")
		}
		contents, err := ioutil.ReadFile(repoFile)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't read registry list")
		}

		// Unmarshal just to validate yaml and short circuit failures from malformed files.
		err = yaml.Unmarshal(contents, map[string]string{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse yaml registry list:")
		}

		cfg.CustomRegistries = string(contents)
	}

	return &cfg, nil
}

// AddRBACModeFlags adds an E2E Argument with the provided default.
func AddRBACModeFlags(mode *RBACMode, flags *pflag.FlagSet, defaultMode RBACMode) {
	*mode = defaultMode // default
	flags.Var(
		mode, "rbac",
		// Doesn't use the map in app.rbacModeMap to preserve order so we can add an explanation for detect.
		"Whether to enable rbac on Sonobuoy. Valid modes are Enable, Disable, and Detect (query the server to see whether to enable RBAC).",
	)
}

// AddSkipPreflightFlag adds a boolean flag to skip preflight checks.
func AddSkipPreflightFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "skip-preflight", false,
		"If true, skip all checks before kicking off the sonobuoy run.",
	)
}

// AddDeleteAllFlag adds a boolean flag for deleting everything (including E2E tests).
func AddDeleteAllFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "all", false,
		"In addition to deleting Sonobuoy namespaces, also clean up dangling e2e namespaces (those with 'e2e-framework' and 'e2e-run' labels).",
	)
}

// AddDeleteWaitFlag adds a boolean flag for waiting for the delete process to complete.
func AddDeleteWaitFlag(flag *int, flags *pflag.FlagSet) {
	flags.IntVar(
		flag, "wait", 0,
		"Wait for resources to be deleted before completing. 0 indicates do not wait. By providing --wait the default is to wait for 1 hour.",
	)
	flags.Lookup("wait").NoOptDefVal = "60"
}

// AddRunWaitFlag adds an int flag for waiting for the entire run to finish.
func AddRunWaitFlag(flag *int, flags *pflag.FlagSet) {
	flags.IntVar(
		flag, "wait", 0,
		"How long (in minutes) to wait for sonobuoy run to be completed or fail, where 0 indicates do not wait. If specified, the default wait time is 1 day.",
	)
	flags.Lookup("wait").NoOptDefVal = "1440"
}

// AddTimeoutFlag adds an int flag for waiting for the entire run to finish.
func AddTimeoutFlag(flag *int, flags *pflag.FlagSet) {
	flags.IntVar(
		flag, timeoutFlag, config.DefaultAggregationServerTimeoutSeconds,
		"How long (in seconds) Sonobuoy will wait for plugins to complete before exiting. 0 indicates no timeout.",
	)
}

// AddShowDefaultPodSpecFlag adds an bool flag for determining whether or not to include the default pod spec
// used by Sonobuoy in the output
func AddShowDefaultPodSpecFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "show-default-podspec", false,
		"If true, include the default pod spec used for plugins in the output",
	)
}

// AddWaitOutputFlag adds a flag for spinner when wait flag is set for Sonobuoy operations.
func AddWaitOutputFlag(mode *WaitOutputMode, flags *pflag.FlagSet, defaultMode WaitOutputMode) {
	*mode = defaultMode
	flags.Var(
		mode, waitOutputFlag,
		"Whether to enable spinner on Sonobuoy. Valid modes are Silent and Spinner")
}

// AddImagePullPolicyFlag adds a boolean flag for deleting everything (including E2E tests).
func AddImagePullPolicyFlag(policy *ImagePullPolicy, flags *pflag.FlagSet) {
	*policy = ImagePullPolicy(v1.PullAlways) //default
	flags.Var(
		policy, imagePullPolicyFlag,
		fmt.Sprintf("The ImagePullPolicy Sonobuoy should use for the aggregators and workers. Valid options are %s.", strings.Join(ValidPullPolicies(), ", ")),
	)
}

// AddSSHKeyPathFlag initialises an SSH key path flag. The SSH key is uploaded
// as a secret and used in the containers to enable running of E2E tests which
// require SSH keys to be present.
func AddSSHKeyPathFlag(path *string, flags *pflag.FlagSet) {
	flags.StringVar(
		path, "ssh-key", "",
		"Path to the private key enabling SSH to cluster nodes.",
	)
}

// AddSSHUserFlag initialises an SSH user flag. Used by the container when
// enabling E2E tests which require SSH.
func AddSSHUserFlag(user *string, flags *pflag.FlagSet) {
	flags.StringVar(
		user, "ssh-user", "",
		"SSH user for ssh-key.",
	)
}

// AddPluginSetFlag adds the flag for gen/run which keeps track of which plugins
// to run and loads them from local files if necessary.
func AddPluginSetFlag(p *pluginList, flags *pflag.FlagSet) {
	flags.VarP(p, "plugin", "p", "Which plugins to run. Can either point to a URL, local file/directory, or be one of the known plugins (e2e or systemd-logs). Can be specified multiple times to run multiple plugins.")
}

// AddPluginEnvFlag adds the flag for gen/run which keeps track of which plugins
// to run and loads them from local files if necessary.
func AddPluginEnvFlag(p *PluginEnvVars, flags *pflag.FlagSet) {
	flags.Var(p, "plugin-env", "Set env vars on plugins. Values can be given multiple times and are in the form plugin.env=value")
}

// AddPluginListFlag adds the flag to keep track of which built-in plugins to use.
func AddPluginListFlag(p *[]string, flags *pflag.FlagSet) {
	flags.StringSliceVarP(p, "plugin", "p", []string{"e2e", "systemd-logs"}, "Describe which plugin's images to interact with (Valid plugins are 'e2e', 'systemd-logs').")
}

// AddKubernetesVersionFlag adds a string flag for the user to specify the cluster version to assume instead of reaching
// out to the cluster to auto-detect it.
func AddKubernetesVersionFlag(version *string, flags *pflag.FlagSet) {
	flags.StringVar(
		version, "kubernetes-version", "",
		"Version to assume for Kubernetes. If empty, the cluster will be queried for its version",
	)
}

// AddShortFlag adds a boolean flag to just print the Sonobuoy version and
// nothing else. Useful in scripts.
func AddShortFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "short", false,
		"If true, prints just the sonobuoy version",
	)
}

// AddDryRunFlag adds a boolean flag to perform a dry-run of image operations.
func AddDryRunFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "dry-run", false,
		"If true, only print the image operations that would be performed.",
	)
}

// AddNodeSelectorFlag adds the flag for gen/run which keeps track of node selectors
// to add to the aggregator. Allows running of the aggregator on Windows nodes.
func AddNodeSelectorsFlag(p *NodeSelectors, flags *pflag.FlagSet) {
	flags.Var(p, "aggregator-node-selector", "Node selectors to add to the aggregator. Values can be given multiple times and are in the form key:value")
}

// AddExtractFlag adds a boolean flag to extract results instead of just downloading the tarball.
func AddExtractFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVarP(
		flag, "extract", "x", false,
		"If true, extracts the results instead of just downloading the results",
	)
}
