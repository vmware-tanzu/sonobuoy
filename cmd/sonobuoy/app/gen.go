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
	"os"

	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	imagepkg "github.com/vmware-tanzu/sonobuoy/pkg/image"

	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

type genFlags struct {
	sonobuoyConfig              SonobuoyConfig
	mode                        client.Mode
	rbacMode                    RBACMode
	kubecfg                     Kubeconfig
	namespace                   string
	dnsNamespace                string
	dnsPodLabels                []string
	sonobuoyImage               string
	kubeConformanceImage        string
	systemdLogsImage            string
	sshKeyPath                  string
	sshUser                     string
	kubeConformanceImageVersion imagepkg.ConformanceImageVersion
	imagePullPolicy             ImagePullPolicy
	timeoutSeconds              int
	showDefaultPodSpec          bool

	// plugins will keep a list of the plugins we want. Custom type for
	// flag support.
	plugins pluginList

	// pluginEnvs is a set of overrides for plugin env vars. Provided out of band
	// from the list of plugins because the e2e/systemd plugins are dynamically
	// generated at the current time and so we can't manipulate those objects.
	pluginEnvs PluginEnvVars

	// nodeSelectors, if set, will be applied to the aggregator allowing it to be
	// schedule on specific nodes.
	nodeSelectors NodeSelectors

	// These two fields are here since to properly squash settings down into nested
	// configs we need to tell whether or not values are default values or the user
	// provided them on the command line/config file.
	e2eflags *pflag.FlagSet
	genflags *pflag.FlagSet
}

func GenFlagSet(cfg *genFlags, rbac RBACMode) *pflag.FlagSet {
	genset := pflag.NewFlagSet("generate", pflag.ExitOnError)
	AddModeFlag(&cfg.mode, genset)
	AddSonobuoyConfigFlag(&cfg.sonobuoyConfig, genset)
	AddKubeconfigFlag(&cfg.kubecfg, genset)
	cfg.e2eflags = AddE2EConfigFlags(genset)
	AddRBACModeFlags(&cfg.rbacMode, genset, rbac)
	AddImagePullPolicyFlag(&cfg.imagePullPolicy, genset)
	AddTimeoutFlag(&cfg.timeoutSeconds, genset)
	AddShowDefaultPodSpecFlag(&cfg.showDefaultPodSpec, genset)

	AddNamespaceFlag(&cfg.namespace, genset)
	AddDNSNamespaceFlag(&cfg.dnsNamespace, genset)
	AddDNSPodLabelsFlag(&cfg.dnsPodLabels, genset)
	AddSonobuoyImage(&cfg.sonobuoyImage, genset)
	AddKubeConformanceImage(&cfg.kubeConformanceImage, genset)
	AddSystemdLogsImage(&cfg.systemdLogsImage, genset)
	AddKubeConformanceImageVersion(&cfg.kubeConformanceImageVersion, genset)
	AddSSHKeyPathFlag(&cfg.sshKeyPath, genset)
	AddSSHUserFlag(&cfg.sshUser, genset)

	AddPluginSetFlag(&cfg.plugins, genset)
	AddPluginEnvFlag(&cfg.pluginEnvs, genset)

	AddNodeSelectorsFlag(&cfg.nodeSelectors, genset)
	cfg.genflags = genset
	return genset
}

func (g *genFlags) Config() (*client.GenConfig, error) {
	e2ecfg, err := GetE2EConfig(g.mode, g.e2eflags)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve E2E config")
	}

	// TODO: Refactor this logic to be less convuled: https://github.com/vmware-tanzu/sonobuoy/issues/481

	// In some configurations, the kube client isn't actually needed for correct executation
	// Therefore, delay reporting the error until we're sure we need the client
	kubeclient, kubeError := getClient(&g.kubecfg)

	// the Enabled and Disabled modes of rbacmode don't need the client, so kubeclient can be nil.
	// if kubeclient is needed, ErrRBACNoClient will be returned and that error can be reported back up.
	rbacEnabled, err := g.rbacMode.Enabled(kubeclient)
	if err != nil {
		if errors.Cause(err) == ErrRBACNoClient {
			return nil, errors.Wrap(err, kubeError.Error())
		}
		return nil, err
	}

	var image string

	// --kube-conformance-image overrides --kube-conformance-image-version
	if g.kubeConformanceImage != "" {
		image = g.kubeConformanceImage
	} else {
		// kubeclient can be null. Prevent a null-pointer exception by gating on
		// that to retrieve the discovery client
		var discoveryClient discovery.ServerVersionInterface
		if kubeclient != nil {
			discoveryClient = kubeclient.DiscoveryClient
		}

		// Only the `auto`  value requires the discovery client to be non-nil
		// if discoveryClient is needed, ErrImageVersionNoClient will be returned and that error can be reported back up
		imageRegistry, imageVersion, err := g.kubeConformanceImageVersion.Get(discoveryClient, imagepkg.DevVersionURL)
		if err != nil {
			if errors.Cause(err) == imagepkg.ErrImageVersionNoClient {
				return nil, errors.Wrap(err, kubeError.Error())
			}
			return nil, err
		}

		image = fmt.Sprintf("%v:%v", imageRegistry, imageVersion)
	}

	return &client.GenConfig{
		E2EConfig:            e2ecfg,
		Config:               g.resolveConfig(),
		EnableRBAC:           rbacEnabled,
		KubeConformanceImage: image,
		SystemdLogsImage:     g.systemdLogsImage,
		ImagePullPolicy:      g.imagePullPolicy.String(),
		SSHKeyPath:           g.sshKeyPath,
		SSHUser:              g.sshUser,
		DynamicPlugins:       g.plugins.DynamicPlugins,
		StaticPlugins:        g.plugins.StaticPlugins,
		PluginEnvOverrides:   g.pluginEnvs,
		ShowDefaultPodSpec:   g.showDefaultPodSpec,
		NodeSelectors:        g.nodeSelectors,
	}, nil
}

func NewCmdGen() *cobra.Command {
	var genflags genFlags
	var GenCommand = &cobra.Command{
		Use:   "gen",
		Short: "Generates a sonobuoy manifest for submission via kubectl",
		Run:   genManifest(&genflags),
		Args:  cobra.ExactArgs(0),
	}
	GenCommand.Flags().AddFlagSet(GenFlagSet(&genflags, EnabledRBACMode))
	return GenCommand
}

func genManifest(genflags *genFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cfg, err := genflags.Config()
		if err != nil {
			errlog.LogError(err)
			os.Exit(1)
		}

		// Generate does not require any client configuration
		sbc := &client.SonobuoyClient{}

		bytes, err := sbc.GenerateManifest(cfg)
		if err == nil {
			fmt.Printf("%s\n", bytes)
			return
		}
		errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
		os.Exit(1)
	}
}

// getClient returns a client if one can be found, and the error attempting to retrieve that client if not.
func getClient(kubeconfig *Kubeconfig) (*kubernetes.Clientset, error) {
	// Usually we don't need a client. But in this case, we _might_ if we're using detect.
	// So pass in nil if we get an error, then display the errors from trying to get a client
	// if it turns out we needed it.
	cfg, err := kubeconfig.Get()
	var client *kubernetes.Clientset

	var kubeError error
	if err == nil {
		client, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			kubeError = err
		}
	} else {
		kubeError = err
	}

	return client, kubeError
}

// getConfig generates a config which has the the following rules:
//  - command line options override config values
//  - plugins specified manually via flags specifically override plugins implied by mode flag
//  - config values override default values
// NOTE: Since it mutates plugin values, it should be called before using them.
func (g *genFlags) resolveConfig() *config.Config {
	if g == nil {
		return config.New()
	}

	conf := config.New()
	suppliedConfig := g.sonobuoyConfig.Get()
	if suppliedConfig != nil {
		mergeConfigs(suppliedConfig, conf)
		conf = suppliedConfig
	}

	// Resolve plugins.
	//  - If using the plugin flags, no actions needed.
	//  - Otherwise use the supplied config and mode to figure out the plugins to run.
	//    This only works for e2e/systemd-logs which are internal plugins so then "Set" them
	//    as if they were provided on the cmdline.
	// Gate the logic with a nil check because tests may not specify flags and intend the legacy logic.
	if g.genflags == nil || !g.genflags.Changed("plugin") {
		// Use legacy logic; conf.SelectedPlugins or mode if not set
		if conf.PluginSelections == nil {
			modeConfig := g.mode.Get()
			if modeConfig != nil {
				conf.PluginSelections = modeConfig.Selectors
			}
		}

		// Set these values as if the user had requested the defaults.
		if g.genflags != nil {
			for _, v := range conf.PluginSelections {
				g.genflags.Lookup("plugin").Value.Set(v.Name)
			}
		}
	}

	// Have to embed the flagset itself so we can inspect if these fields
	// have been set explicitly or not on the command line. Otherwise
	// we fail to properly prioritize command line/config/default values.
	if g.genflags == nil {
		return conf
	}

	if g.genflags.Changed(namespaceFlag) {
		conf.Namespace = g.namespace
	}

	if g.genflags.Changed(sonobuoyImageFlag) {
		conf.WorkerImage = g.sonobuoyImage
	}

	if g.genflags.Changed(imagePullPolicyFlag) {
		conf.ImagePullPolicy = g.imagePullPolicy.String()
	}

	if g.genflags.Changed(timeoutFlag) {
		conf.Aggregation.TimeoutSeconds = g.timeoutSeconds
	}

	return conf
}

func mergeConfigs(dst, src *config.Config) {
	// Workaround for the fact that an explicitly stated empty slice is still
	// considered a zero value by mergo. This means that the value given
	// by the user is not respected. Even a custom transformation can't
	// get around this. See https://github.com/imdario/mergo/issues/118
	emptyResources := false
	if len(dst.Resources) == 0 && dst.Resources != nil {
		emptyResources = true
	}

	// Workaround to differentiate a false value and a nil value
	// Only override dst.Limits.PodLogs.SonobuoyNamespace when it's nil
	// See https://github.com/imdario/mergo/issues/89
	var sonobuoyNamespace *bool
	if dst.Limits.PodLogs.SonobuoyNamespace != nil {
		sonobuoyNamespace = new(bool)
		*sonobuoyNamespace = *dst.Limits.PodLogs.SonobuoyNamespace
	}

	// Provide defaults but don't overwrite any customized configuration.
	mergo.Merge(dst, src)

	if emptyResources {
		dst.Resources = []string{}
	}
	if sonobuoyNamespace != nil {
		dst.Limits.PodLogs.SonobuoyNamespace = sonobuoyNamespace
	}
}
