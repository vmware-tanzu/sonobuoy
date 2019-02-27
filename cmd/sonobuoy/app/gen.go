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

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/errlog"
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
	sonobuoyImage               string
	kubeConformanceImage        string
	sshKeyPath                  string
	sshUser                     string
	kubeConformanceImageVersion ConformanceImageVersion
	imagePullPolicy             ImagePullPolicy

	// These two fields are here since to properly squash settings down into nested
	// configs we need to tell whether or not values are default values or the user
	// provided them on the command line/config file.
	e2eflags *pflag.FlagSet
	genflags *pflag.FlagSet
}

var genflags genFlags

func GenFlagSet(cfg *genFlags, rbac RBACMode, version ConformanceImageVersion) *pflag.FlagSet {
	genset := pflag.NewFlagSet("generate", pflag.ExitOnError)
	AddModeFlag(&cfg.mode, genset)
	AddSonobuoyConfigFlag(&cfg.sonobuoyConfig, genset)
	AddKubeconfigFlag(&cfg.kubecfg, genset)
	cfg.e2eflags = AddE2EConfigFlags(genset)
	AddRBACModeFlags(&cfg.rbacMode, genset, rbac)
	AddImagePullPolicyFlag(&cfg.imagePullPolicy, genset)

	AddNamespaceFlag(&cfg.namespace, genset)
	AddSonobuoyImage(&cfg.sonobuoyImage, genset)
	AddKubeConformanceImage(&cfg.kubeConformanceImage, genset)
	AddKubeConformanceImageVersion(&cfg.kubeConformanceImageVersion, genset, version)
	AddSSHKeyPathFlag(&cfg.sshKeyPath, genset)
	AddSSHUserFlag(&cfg.sshUser, genset)

	cfg.genflags = genset
	return genset
}

func (g *genFlags) Config() (*client.GenConfig, error) {
	e2ecfg, err := GetE2EConfig(g.mode, g.e2eflags)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve E2E config")
	}

	// TODO: Refactor this logic to be less convuled: https://github.com/heptio/sonobuoy/issues/481

	// In some configurations, the kube client isn't actually needed for correct executation
	// Therefore, delay reporting the error until we're sure we need the client
	kubeclient, kubeError := getClient(&g.kubecfg)

	// the Enabled and Disabled modes of rbacmode don't need the client, so kubeclient can be nil.
	// if kubeclient is needed, ErrRBACNoClient will be returned and that error can be reported back up.
	rbacEnabled, err := genflags.rbacMode.Enabled(kubeclient)
	if err != nil {
		if errors.Cause(err) == ErrRBACNoClient {
			return nil, errors.Wrap(err, kubeError.Error())
		}
		return nil, err
	}

	var discoveryClient discovery.ServerVersionInterface
	var image string

	// --kube-conformance-image overrides --kube-conformance-image-version
	if g.kubeConformanceImage != "" {
		image = g.kubeConformanceImage
	} else {
		// kubeclient can be null. Prevent a null-pointer exception by gating on that to retrieve the discovery client
		if kubeclient != nil {
			discoveryClient = kubeclient.DiscoveryClient
		}

		// Only the `auto`  value requires the discovery client to be non-nil
		// if discoveryClient is needed, ErrImageVersionNoClient will be returned and that error can be reported back up
		imageVersion, err := g.kubeConformanceImageVersion.Get(discoveryClient)
		if err != nil {
			if errors.Cause(err) == ErrImageVersionNoClient {
				return nil, errors.Wrap(err, kubeError.Error())
			}
			return nil, err
		}

		image = config.DefaultKubeConformanceImageURL + ":" + imageVersion
	}

	return &client.GenConfig{
		E2EConfig:            e2ecfg,
		Config:               g.getConfig(),
		Image:                g.sonobuoyImage,
		Namespace:            g.namespace,
		EnableRBAC:           rbacEnabled,
		ImagePullPolicy:      g.imagePullPolicy.String(),
		KubeConformanceImage: image,
		SSHKeyPath:           g.sshKeyPath,
		SSHUser:              g.sshUser,
	}, nil
}

func NewCmdGen() *cobra.Command {
	var GenCommand = &cobra.Command{
		Use:   "gen",
		Short: "Generates a sonobuoy manifest for submission via kubectl",
		Run:   genManifest,
		Args:  cobra.ExactArgs(0),
	}
	GenCommand.Flags().AddFlagSet(GenFlagSet(&genflags, EnabledRBACMode, ConformanceImageVersionLatest))
	return GenCommand
}

func genManifest(cmd *cobra.Command, args []string) {
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

// getConfig creates a config with the following algorithm:
// If no config is supplied defaults will be returned.
// If a config is supplied then the default values will be merged into the supplied config
//   in order to allow users to supply a minimal config that will still work.
// Lastly, options provided on the command line will override
//   any values in the config.
func (g *genFlags) getConfig() *config.Config {
	if g == nil {
		return config.New()
	}

	conf := config.New()

	suppliedConfig := g.sonobuoyConfig.Get()
	if suppliedConfig != nil {
		// Provide defaults but don't overwrite any customized configuration.
		mergo.Merge(suppliedConfig, conf)
		conf = suppliedConfig
	}

	// if there are no plugins yet, set some based on the mode, otherwise use whatever was supplied.
	if len(conf.PluginSelections) == 0 {
		modeConfig := g.mode.Get()
		if modeConfig != nil {
			conf.PluginSelections = modeConfig.Selectors
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

	return conf
}
