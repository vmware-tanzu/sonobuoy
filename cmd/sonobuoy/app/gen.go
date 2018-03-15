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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

type genFlags struct {
	sonobuoyConfig       SonobuoyConfig
	mode                 client.Mode
	rbacMode             RBACMode
	kubecfg              Kubeconfig
	e2eflags             *pflag.FlagSet
	namespace            string
	sonobuoyImage        string
	kubeConformanceImage string
	imagePullPolicy      ImagePullPolicy
}

var genflags genFlags

func GenFlagSet(cfg *genFlags, rbac RBACMode) *pflag.FlagSet {
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

	return genset
}

func (g *genFlags) Config() (*client.GenConfig, error) {
	e2ecfg, err := GetE2EConfig(g.mode, g.e2eflags)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve E2E config")
	}

	return &client.GenConfig{
		E2EConfig:            e2ecfg,
		Config:               GetConfigWithMode(&g.sonobuoyConfig, g.mode),
		Image:                g.sonobuoyImage,
		Namespace:            g.namespace,
		EnableRBAC:           getRBACOrExit(&g.rbacMode, &g.kubecfg),
		ImagePullPolicy:      g.imagePullPolicy.String(),
		KubeConformanceImage: g.kubeConformanceImage,
	}, nil
}

// GenCommand is exported so it can be extended.
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "Generates a sonobuoy manifest for submission via kubectl",
	Run:   genManifest,
	Args:  cobra.ExactArgs(0),
}

func init() {
	GenCommand.Flags().AddFlagSet(GenFlagSet(&genflags, EnabledRBACMode))
	RootCmd.AddCommand(GenCommand)
}

func genManifest(cmd *cobra.Command, args []string) {
	cfg, err := genflags.Config()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
	kubeCfg, err := genflags.kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get kubernetes config"))
		os.Exit(1)
	}
	sbc, err := client.NewSonobuoyClient(kubeCfg)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	bytes, err := sbc.GenerateManifest(cfg)
	if err == nil {
		fmt.Printf("%s\n", bytes)
		return
	}
	errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
	os.Exit(1)
}

// getRBACOrExit is a helper function for working with RBACMode. RBACMode is a bit of a special case
// because it only needs a kubeconfig for detect, otherwise errors from kubeconfig can be ignored.
// This function returns a bool because it os.Exit()s in error cases.
func getRBACOrExit(mode *RBACMode, kubeconfig *Kubeconfig) bool {

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

	rbacEnabled, err := genflags.rbacMode.Enabled(client)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't detect RBAC mode."))
		if errors.Cause(err) == ErrRBACNoClient {
			errlog.LogError(kubeError)
		}
		os.Exit(1)
	}
	return rbacEnabled
}
