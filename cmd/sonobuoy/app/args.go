/*
Copyright 2018 Heptio Inc.

Copyright 2022 the Sonobuoy Project contributors

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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/image"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
)

const (
	namespaceFlag              = "namespace"
	sonobuoyImageFlag          = "sonobuoy-image"
	imagePullPolicyFlag        = "image-pull-policy"
	forceImagePullPolicyFlag   = "force-image-pull-policy"
	pluginFlag                 = "plugin"
	timeoutFlag                = "timeout"
	waitOutputFlag             = "wait-output"
	customRegistryFlag         = "custom-registry"
	kubeconfig                 = "kubeconfig"
	kubecontext                = "context"
	e2eFocusFlag               = "e2e-focus"
	e2eSkipFlag                = "e2e-skip"
	e2eParallelFlag            = "e2e-parallel"
	e2eRegistryConfigFlag      = "e2e-repo-config"
	e2eRegistryFlag            = "e2e-repo"
	pluginImageFlag            = "plugin-image"
	filenameFlag               = "filename"
	retrievePathFlag           = "retrieve-path"
	securityContextModeFlag    = "security-context-mode"
	aggregatorPermissionsFlag  = "aggregator-permissions"
	serviceAccountNameFlag     = "service-account-name"
	existingServiceAccountFlag = "existing-service-account"
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

// AddSonobuoyImage initialises an image url flag.
func AddSonobuoyImage(image *string, flags *pflag.FlagSet) {
	flags.StringVar(
		image, sonobuoyImageFlag, *image,
		"Container image override for the sonobuoy worker and aggregator.",
	)
}

func AddPluginImage(pluginTransforms *map[string][]func(*manifest.Manifest) error, fs *pflag.FlagSet) {
	fs.Var(
		&pluginImageFlagType{
			transforms: *pluginTransforms,
		},
		pluginImageFlag,
		"Override a plugins image from what is in its definition (e.g. myPlugin:testimage)",
	)
}

// AddKubeConformanceImage initialises the kube-conformance-image flag. Really just a legacy wrapper for --plugin-image=e2e:imageName
func AddKubeConformanceImage(pluginTransforms *map[string][]func(*manifest.Manifest) error, fs *pflag.FlagSet) {
	fs.Var(
		&hardcodedPluginImageFlagType{
			plugin: e2ePlugin,
			underlyingFlag: &pluginImageFlagType{
				transforms: *pluginTransforms,
			},
		},
		"kube-conformance-image",
		"Container image override for the e2e plugin. Shorthand for --plugin-image=e2e:<string>",
	)
}

// AddSystemdLogsImage initialises the systemd-logs-image flag. Really just a legacy wrapper for --plugin-image=systemd-logs:imageName
func AddSystemdLogsImage(pluginTransforms *map[string][]func(*manifest.Manifest) error, fs *pflag.FlagSet) {
	fs.Var(
		&hardcodedPluginImageFlagType{
			plugin: systemdLogsPlugin,
			underlyingFlag: &pluginImageFlagType{
				transforms: *pluginTransforms,
			},
		},
		"systemd-logs-image",
		"Container image override for the systemd-logs plugin. Shorthand for --plugin-image=systemd-logs:<string>",
	)
}

// AddKubeConformanceImageVersion initialises an image version flag.
func AddKubeConformanceImageVersion(imageVersion *image.ConformanceImageVersion, pluginTransforms *map[string][]func(*manifest.Manifest) error, flags *pflag.FlagSet) {
	help := "Use default e2e plugin image, but override the version. "
	help += "Default is 'auto', which will be set to your cluster's version if detected, erroring otherwise."
	help += "You can also choose 'latest' which will find the latest dev image upstream."

	flags.Var(
		&kubernetesVersionLogicFlag{
			raw: imageVersion,
			underlyingFlag: &pluginImageFlagType{
				transforms: *pluginTransforms,
			},
		}, "kube-conformance-image-version", help)
	if err := flags.MarkDeprecated("kube-conformance-image-version", "Use --kubernetes-version instead."); err != nil {
		panic("Failed to setup flags properly")
	}
}

// AddKubeconfigFlag adds a kubeconfig and context flags to the provided command.
func AddKubeconfigFlag(cfg *Kubeconfig, flags *pflag.FlagSet) {
	// The default is the empty string (look in the environment)
	flags.Var(cfg, "kubeconfig", "Path to explicit kubeconfig file.")
	flags.StringVar(&cfg.Context, kubecontext, "", "Context in the kubeconfig to use.")
}

func AddSecurityContextMode(mode *string, flags *pflag.FlagSet) {
	flags.StringVar(
		mode, securityContextModeFlag, "nonroot",
		"Type of security context to use for the aggregator pod. Allowable values are [none, nonroot]",
	)
}

func AddAggregatorPermissionsFlag(mode *string, flags *pflag.FlagSet) {
	flags.StringVar(
		mode, aggregatorPermissionsFlag, "clusterAdmin",
		"Type of aggregator permission to use in the cluster. Allowable values are [namespaceAdmin, clusterRead, clusterAdmin]",
	)
}

// AddExistingServiceAccountFlag adds a boolean flag which disables service account creation.
func AddExistingServiceAccountFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, existingServiceAccountFlag, false,
		"If true, use an existing service account, else attempt to create one.",
	)
}

func AddServiceAccountNameFlag(name *string, flags *pflag.FlagSet) {
	flags.StringVar(
		name, serviceAccountNameFlag, "sonobuoy-serviceaccount",
		"Name of the service account to be used by sonobuoy.",
	)
}

// AddE2ERegistryConfigFlag adds a e2eRegistryConfigFlag flag to the provided command.
func AddE2ERegistryConfigFlag(cfg *string, flags *pflag.FlagSet) {
	flags.StringVar(
		cfg, e2eRegistryConfigFlag, "",
		"Specify a yaml file acting as KUBE_TEST_REPO_LIST, overriding registries for test images.",
	)
}

// AddE2ERegistryFlag adds a e2eRegistryFlag flag to the provided command.
func AddE2ERegistryFlag(cfg *string, flags *pflag.FlagSet) {
	flags.StringVar(
		cfg, e2eRegistryFlag, "",
		"Specify a registry for KUBE_TEST_REPO, overriding registries for test images.",
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
		"Path to a sonobuoy configuration JSON file.",
	)
}

// AddLegacyE2EFlags is a way to add flags which target the e2e plugin specifically
// by leveraging the existing flags. They typically wrap other fields (like the env var
// overrides) and modify those.
func AddLegacyE2EFlags(env *PluginEnvVars, pluginTransforms *map[string][]func(*manifest.Manifest) error, fs *pflag.FlagSet) {
	m := &Mode{
		env:  env,
		name: "",
	}
	if err := m.Set(E2eModeNonDisruptiveConformance); err != nil {
		panic("Failed to initial mode flag")
	}
	fs.VarP(
		m, "mode", "m",
		fmt.Sprintf("What mode to run the e2e plugin in. Valid modes are %s.", validE2eModes()),
	)
	fs.Var(
		&envVarModierFlag{
			plugin: e2ePlugin, field: "E2E_FOCUS",
			PluginEnvVars:  *env,
			validationFunc: regexpValidation},
		e2eFocusFlag,
		"Specify the E2E_FOCUS value for the e2e plugin, specifying which tests to run. Shorthand for --plugin-env=e2e.E2E_FOCUS=<string>",
	)
	fs.Var(
		&envVarModierFlag{
			plugin: e2ePlugin, field: "E2E_SKIP",
			PluginEnvVars:  *env,
			validationFunc: regexpValidation},
		e2eSkipFlag,
		"Specify the E2E_SKIP value for the e2e plugin, specifying which tests to skip. Shorthand for --plugin-env=e2e.E2E_SKIP=<string>",
	)
	pf := fs.VarPF(
		&envVarModierFlag{
			plugin:        e2ePlugin,
			field:         "E2E_PARALLEL",
			PluginEnvVars: *env,
		}, e2eParallelFlag, "",
		"Specify the E2E_PARALLEL value for the e2e plugin. Shorthand for --plugin-env=e2e.E2E_PARALLEL=<string>",
	)
	if err := pf.Value.Set("false"); err != nil {
		panic("Failed to initial parallel flag")
	}
	pf.Hidden = true

	// Used to replace the --rerun-failed functionality from the old e2e command.
	fs.Var(
		&rerunFailedFlag{env: env}, "rerun-failed",
		"Read the given tarball and set the E2E_FOCUS to target all the failed tests",
	)

	// Used by the container when enabling E2E tests which require SSH.
	fs.Var(
		&envVarModierFlag{plugin: e2ePlugin, field: "KUBE_SSH_USER", PluginEnvVars: *env}, "ssh-user",
		"SSH user for ssh-key. Required if running e2e plugin with certain tests that require SSH access to nodes.",
	)

	fs.Var(
		&e2eRepoFlag{
			plugin:     e2ePlugin,
			transforms: *pluginTransforms,
		}, e2eRegistryConfigFlag,
		"Specify a yaml file acting as KUBE_TEST_REPO_LIST, overriding registries for test images.",
	)

	fs.Var(
		&envVarModierFlag{plugin: e2ePlugin, field: "KUBE_TEST_REPO", PluginEnvVars: *env}, e2eRegistryFlag,
		"Specify a registry to use as the default for pulling Kubernetes test images. Same as providing --e2e-repo-config but specifying the same repo repeatedly.",
	)
}

// AddRBACModeFlags adds an E2E Argument with the provided default.
func AddRBACModeFlags(mode *RBACMode, flags *pflag.FlagSet, defaultMode RBACMode) {
	*mode = defaultMode // default
	flags.Var(
		mode, "rbac",
		// Doesn't use the map in app.rbacModeMap to preserve order so we can add an explanation for detect.
		"Whether to enable RBAC on Sonobuoy. Valid modes are Enable, Disable, and Detect (query the server to see whether to enable RBAC).",
	)
}

// AddSkipPreflightFlag adds a boolean flag to skip preflight checks.
func AddSkipPreflightFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "skip-preflight", false,
		"If true, skip all checks before starting the sonobuoy run.",
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
		"Wait for resources to be deleted before completing. 0 indicates do not wait. By providing --wait the default is to wait up to 1 hour.",
	)
	flags.Lookup("wait").NoOptDefVal = "60"
}

// AddRunWaitFlag adds an int flag for waiting for the entire run to finish.
func AddRunWaitFlag(flag *int, flags *pflag.FlagSet) {
	flags.IntVar(
		flag, "wait", 0,
		"How long (in minutes) for the CLI to wait for sonobuoy run to be completed or fail, where 0 indicates do not wait. If specified, the default wait time is 1 day.",
	)
	flags.Lookup("wait").NoOptDefVal = "1440"
}

// AddTimeoutFlag adds an int flag for waiting for the entire run to finish.
func AddTimeoutFlag(flag *int, flags *pflag.FlagSet) {
	flags.IntVar(
		flag, timeoutFlag, config.DefaultAggregationServerTimeoutSeconds,
		"How long (in seconds) Sonobuoy aggregator will wait for plugins to complete before exiting. 0 indicates no timeout.",
	)
}

// AddShowDefaultPodSpecFlag adds an bool flag for determining whether or not to include the default pod spec
// used by Sonobuoy in the output
func AddShowDefaultPodSpecFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "show-default-podspec", false,
		"If true, include the default pod spec used for plugins in the output.",
	)
}

// AddWaitOutputFlag adds a flag for spinner when wait flag is set for Sonobuoy operations.
func AddWaitOutputFlag(mode *WaitOutputMode, flags *pflag.FlagSet, defaultMode WaitOutputMode) {
	*mode = defaultMode
	flags.Var(
		mode, waitOutputFlag,
		"Specify the type of output Sonobuoy should produce when --wait is used. Valid modes are silent, spinner, or progress")
}

// AddImagePullPolicyFlag adds a string flag for the image pull policy for the aggregator/worker. If ForceImagePullPolicy
// is set, it will also apply to the plugins.
func AddImagePullPolicyFlag(policy *string, flags *pflag.FlagSet) {
	flags.StringVar(
		policy, imagePullPolicyFlag, config.DefaultSonobuoyPullPolicy,
		fmt.Sprintf("Set the ImagePullPolicy for the Sonobuoy image and all plugins (if --%v is set). Valid options are %s.", forceImagePullPolicyFlag, strings.Join(ValidPullPolicies(), ", ")),
	)
}

// AddForceImagePullPolicyFlag adds a boolean flag for applying the ImagePullPolicy to all plugins even if set in their
// plugin specification.
func AddForceImagePullPolicyFlag(apply *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		apply, forceImagePullPolicyFlag, false, "Force plugins' imagePullPolicy to match the value for the Sonobuoy pod",
	)
}

// AddSSHKeyPathFlag initialises an SSH key path flag. The SSH key is uploaded
// as a secret and used in the containers to enable running of E2E tests which
// require SSH keys to be present.
func AddSSHKeyPathFlag(path *string, pluginTransforms *map[string][]func(*manifest.Manifest) error, flags *pflag.FlagSet) {
	flags.Var(
		&sshPathFlag{
			filename:   path,
			transforms: *pluginTransforms,
		}, "ssh-key",
		"Path to the private key enabling SSH to cluster nodes. May be required by some tests from the e2e plugin.",
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

// AddKubernetesVersionFlag initialises an image version flag.
func AddKubernetesVersionFlag(imageVersion *image.ConformanceImageVersion, pluginTransforms *map[string][]func(*manifest.Manifest) error, flags *pflag.FlagSet) {
	help := "Use default E2E image, but override the version. "
	help += "Default is 'auto', which will be set to your cluster's version if detected, erroring otherwise. "
	help += "'ignore' will try version resolution but ignore errors. "
	help += "'latest' will find the latest dev image/version upstream."

	flags.Var(
		&kubernetesVersionLogicFlag{
			raw: imageVersion,
			underlyingFlag: &pluginImageFlagType{
				transforms: *pluginTransforms,
			},
		}, "kubernetes-version", help)
}

// AddShortFlag adds a boolean flag to just print the Sonobuoy version and
// nothing else. Useful in scripts.
func AddShortFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "short", false,
		"If true, prints just the Sonobuoy version.",
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
		"If true, extracts the results instead of just downloading the results tarball.",
	)
}

// AddFilename initialises a namespace flag.
func AddFilenameFlag(str *string, flags *pflag.FlagSet) {
	flags.StringVarP(
		str, filenameFlag, "f", "",
		"Specify the name of the downloaded file. If empty, defaults to the name of the tarball in the pod.",
	)
}

// AddRetrievePathFlag initialises a namespace flag.
func AddRetrievePathFlag(str *string, flags *pflag.FlagSet) {
	flags.StringVar(
		str, retrievePathFlag, config.AggregatorResultsPath,
		"Specify the path to retrieve from the aggregator pod.",
	)
}

// Used if we're just setting the given string as the value; focus and skip need
// regexp validation first.
type envVarModierFlag struct {
	plugin, field string
	PluginEnvVars
	validationFunc func(string) error
}

func (i *envVarModierFlag) String() string {
	if i != nil && (i.PluginEnvVars)[i.plugin] != nil {
		return (i.PluginEnvVars)[i.plugin][i.field]
	}
	return ""
}
func (i *envVarModierFlag) Type() string { return "envModifier" }
func (i *envVarModierFlag) Set(str string) error {
	if i.validationFunc != nil {
		if err := i.validationFunc(str); err != nil {
			return err
		}
	}
	return (i.PluginEnvVars).Set(fmt.Sprintf("%v.%v=%v", i.plugin, i.field, str))
}

func regexpValidation(s string) error {
	if _, err := regexp.Compile(s); err != nil {
		return errors.Wrapf(err, "flag value %q fails regexp validation", s)
	}
	return nil
}

// Mode represents the sonobuoy configuration for a given mode.
type Mode struct {
	name string
	env  *PluginEnvVars
}

func (m *Mode) String() string { return m.name }
func (m *Mode) Type() string   { return "Mode" }
func (m *Mode) Set(str string) error {
	lcase := strings.ToLower(str)
	mode, ok := validModes[lcase]
	if !ok {
		return fmt.Errorf("unknown mode %v", lcase)
	}

	if err := m.env.Set(fmt.Sprintf("e2e.E2E_FOCUS=%v", mode.focus)); err != nil {
		return fmt.Errorf("failed to set flag with value %v", str)
	}
	if err := m.env.Set(fmt.Sprintf("e2e.E2E_SKIP=%v", mode.skip)); err != nil {
		return fmt.Errorf("failed to set flag with value %v", str)
	}
	if err := m.env.Set(fmt.Sprintf("e2e.E2E_PARALLEL=%v", mode.parallel)); err != nil {
		return fmt.Errorf("failed to set flag with value %v", str)
	}

	m.name = lcase
	return nil
}

type e2eRepoFlag struct {
	plugin   string
	filename string

	transforms map[string][]func(*manifest.Manifest) error

	// Value to put in the configmap as the filename. Defaults to filename.
	filenameOverride string
}

func (f *e2eRepoFlag) String() string { return f.filename }
func (f *e2eRepoFlag) Type() string   { return "yaml-filepath" }
func (f *e2eRepoFlag) Set(str string) error {
	f.filename = str
	name := filepath.Base(str)
	if len(f.filenameOverride) > 0 {
		name = f.filenameOverride
	}
	fData, err := os.ReadFile(str)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %q", str)
	}

	f.transforms[f.plugin] = append(f.transforms[f.plugin], func(m *manifest.Manifest) error {
		m.ConfigMap = map[string]string{
			name: string(fData),
		}
		m.Spec.Env = append(m.Spec.Env, corev1.EnvVar{
			Name:  "KUBE_TEST_REPO_LIST",
			Value: fmt.Sprintf("/tmp/sonobuoy/config/%v", name),
		})
		return nil
	})
	return nil
}

// The ssh-key flag needs to store the path to the ssh key but also
// wire up the e2e plugin for using it.
type sshPathFlag struct {
	filename   *string
	transforms map[string][]func(*manifest.Manifest) error
}

func (f *sshPathFlag) String() string { return *f.filename }
func (f *sshPathFlag) Type() string   { return "yamlFile" }
func (f *sshPathFlag) Set(str string) error {
	*f.filename = str

	f.transforms[e2ePlugin] = append(f.transforms[e2ePlugin], func(m *manifest.Manifest) error {
		// Add volume mount, volume, and 3 env vars (for different possible platforms) for SSH capabilities.
		defMode := int32(256)
		m.ExtraVolumes = append(m.ExtraVolumes, manifest.Volume{
			Volume: corev1.Volume{
				Name: "sshkey-vol",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  "ssh-key",
						DefaultMode: &defMode,
					},
				},
			},
		})
		m.Spec.Env = append(m.Spec.Env,
			corev1.EnvVar{Name: "LOCAL_SSH_KEY", Value: "id_rsa"},
			corev1.EnvVar{Name: "AWS_SSH_KEY", Value: "/root/.ssh/id_rsa"},
			corev1.EnvVar{Name: "KUBE_SSH_KEY", Value: "id_rsa"},
		)
		m.Spec.VolumeMounts = append(m.Spec.VolumeMounts,
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      "sshkey-vol",
				MountPath: "/root/.ssh",
			},
		)
		return nil
	})
	return nil
}

type pluginImageFlagType struct {
	overrides  map[string]string
	transforms map[string][]func(*manifest.Manifest) error
}

func (f *pluginImageFlagType) String() string { return fmt.Sprint(f.overrides) }
func (f *pluginImageFlagType) Type() string   { return "plugin:image" }
func (f *pluginImageFlagType) Set(str string) error {
	parts := strings.SplitN(str, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("failed to parse plugin image flag, expected format plugin:image and got %v", str)
	}
	if f.overrides == nil {
		f.overrides = map[string]string{}
	}
	f.overrides[parts[0]] = parts[1]

	f.transforms[parts[0]] = append(f.transforms[parts[0]], func(m *manifest.Manifest) error {
		m.Spec.Image = parts[1]
		return nil
	})
	return nil
}

type hardcodedPluginImageFlagType struct {
	underlyingFlag *pluginImageFlagType
	plugin         string
}

func (f *hardcodedPluginImageFlagType) String() string {
	return f.underlyingFlag.String()
}
func (f *hardcodedPluginImageFlagType) Type() string { return "image" }
func (f *hardcodedPluginImageFlagType) Set(str string) error {
	return f.underlyingFlag.Set(fmt.Sprintf("%v:%v", f.plugin, str))
}

type kubernetesVersionLogicFlag struct {
	underlyingFlag *pluginImageFlagType
	raw            *image.ConformanceImageVersion
}

func (f *kubernetesVersionLogicFlag) String() string {
	return f.raw.String()
}
func (f *kubernetesVersionLogicFlag) Type() string { return "string" }
func (f *kubernetesVersionLogicFlag) Set(str string) error {
	if err := f.raw.Set(str); err != nil {
		return err
	}

	var img, ver string
	switch *f.raw {
	case "", image.ConformanceImageVersionIgnore, image.ConformanceImageVersionAuto:
		img = config.UpstreamKubeConformanceImageURL
		ver = "$SONOBUOY_K8S_VERSION"
	case image.ConformanceImageVersionLatest:
		version, err := image.GetLatestDevVersion(image.DevVersionURL)
		if err != nil {
			return errors.Wrap(err, "couldn't identify latest dev image")
		}

		img = image.DevVersionImageURL
		ver = version
	default:
		img = config.UpstreamKubeConformanceImageURL
		ver = f.raw.String()
	}
	return f.underlyingFlag.Set(fmt.Sprintf("%v:%v:%v", e2ePlugin, img, ver))
}

type rerunFailedFlag struct {
	s   string
	env *PluginEnvVars
}

func (f *rerunFailedFlag) String() string { return f.s }
func (f *rerunFailedFlag) Type() string   { return "tar.gz file" }
func (f *rerunFailedFlag) Set(str string) error {
	failedList, err := failedTestsFromTar(str, "e2e")
	if err != nil {
		return err
	}

	if len(failedList) == 0 {
		return fmt.Errorf("no tests failed for plugin %q in tarball %q", "e2e", str)
	}

	val := stringsToRegexp(failedList)
	if err := f.env.Set(fmt.Sprintf("e2e.E2E_FOCUS=%v", val)); err != nil {
		return fmt.Errorf("failed to set flag with value %v", str)
	}

	// Removes E2E_SKIP value.
	if err := f.env.Set("e2e.E2E_SKIP"); err != nil {
		return fmt.Errorf("failed to set flag with value %v", str)
	}
	f.s = val
	return nil
}
