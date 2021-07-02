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
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

type genFlags struct {
	sonobuoyConfig     SonobuoyConfig
	rbacMode           RBACMode
	kubecfg            Kubeconfig
	dnsNamespace       string
	dnsPodLabels       []string
	sshKeyPath         string
	k8sVersion         imagepkg.ConformanceImageVersion
	showDefaultPodSpec bool

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

	pluginTransforms map[string][]func(*manifest.Manifest) error
}

func GenFlagSet(cfg *genFlags, rbac RBACMode) *pflag.FlagSet {
	genset := pflag.NewFlagSet("generate", pflag.ExitOnError)
	cfg.sonobuoyConfig.Config = *config.New()
	cfg.pluginTransforms = map[string][]func(*manifest.Manifest) error{}
	AddSonobuoyConfigFlag(&cfg.sonobuoyConfig, genset)
	AddKubeconfigFlag(&cfg.kubecfg, genset)
	AddRBACModeFlags(&cfg.rbacMode, genset, rbac)
	AddImagePullPolicyFlag(&cfg.sonobuoyConfig.ImagePullPolicy, genset)
	AddTimeoutFlag(&cfg.sonobuoyConfig.Aggregation.TimeoutSeconds, genset)
	AddShowDefaultPodSpecFlag(&cfg.showDefaultPodSpec, genset)

	AddNamespaceFlag(&cfg.sonobuoyConfig.Namespace, genset)
	AddDNSNamespaceFlag(&cfg.dnsNamespace, genset)
	AddDNSPodLabelsFlag(&cfg.dnsPodLabels, genset)
	AddSonobuoyImage(&cfg.sonobuoyConfig.WorkerImage, genset)
	AddSSHKeyPathFlag(&cfg.sshKeyPath, &cfg.pluginTransforms, genset)

	AddPluginSetFlag(&cfg.plugins, genset)
	AddPluginEnvFlag(&cfg.pluginEnvs, genset)
	AddLegacyE2EFlags(&cfg.pluginEnvs, &cfg.pluginTransforms, genset)

	AddNodeSelectorsFlag(&cfg.nodeSelectors, genset)

	AddKubeConformanceImageVersion(&cfg.k8sVersion, &cfg.pluginTransforms, genset)
	AddKubernetesVersionFlag(&cfg.k8sVersion, &cfg.pluginTransforms, genset)

	AddPluginImage(&cfg.pluginTransforms, genset)
	AddKubeConformanceImage(&cfg.pluginTransforms, genset)
	AddSystemdLogsImage(&cfg.pluginTransforms, genset)

	return genset
}

func (g *genFlags) Config() (*client.GenConfig, error) {
	if len(g.plugins.DynamicPlugins) == 0 && len(g.plugins.StaticPlugins) == 0 {
		g.plugins.DynamicPlugins = []string{e2ePlugin, systemdLogsPlugin}
	}

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

	var k8sVersion string
	switch g.k8sVersion {
	case "", imagepkg.ConformanceImageVersionAuto, imagepkg.ConformanceImageVersionLatest, imagepkg.ConformanceImageVersionIgnore:
		var discoveryClient discovery.ServerVersionInterface
		if kubeclient != nil {
			discoveryClient = kubeclient.DiscoveryClient
		}

		// `auto` k8s version needs resolution as well as any static plugins which use the
		// variable SONOBUOY_K8S_VERSION. Just check for it all by default but allow skipping
		// errors/resolution via flag.
		_, k8sVersion, err = g.k8sVersion.Get(discoveryClient, imagepkg.DevVersionURL)
		if err != nil {
			if errors.Cause(err) == imagepkg.ErrImageVersionNoClient &&
				g.k8sVersion != imagepkg.ConformanceImageVersionIgnore {
				return nil, errors.Wrap(err, kubeError.Error())
			}
			return nil, err
		}
	default:
		k8sVersion = g.k8sVersion.String()
	}

	return &client.GenConfig{
		Config:             &g.sonobuoyConfig.Config,
		EnableRBAC:         rbacEnabled,
		ImagePullPolicy:    g.sonobuoyConfig.ImagePullPolicy,
		SSHKeyPath:         g.sshKeyPath,
		DynamicPlugins:     g.plugins.DynamicPlugins,
		StaticPlugins:      g.plugins.StaticPlugins,
		PluginEnvOverrides: g.pluginEnvs,
		ShowDefaultPodSpec: g.showDefaultPodSpec,
		NodeSelectors:      g.nodeSelectors,
		KubeVersion:        k8sVersion,
		PluginTransforms:   g.pluginTransforms,
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
