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

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

type genFlags struct {
	sonobuoyConfig SonobuoyConfig
	mode           ops.Mode
	rbacMode       RBACMode
	kubecfg        Kubeconfig
}

var genopts ops.GenConfig
var genflags genFlags

func (g *genFlags) AddFlags(flags *pflag.FlagSet, cfg *ops.GenConfig, rbac RBACMode) {
	genset := pflag.NewFlagSet("generate", pflag.ExitOnError)
	AddModeFlag(&g.mode, genset)
	AddSonobuoyConfigFlag(&g.sonobuoyConfig, genset)
	AddKubeconfigFlag(&g.kubecfg, genset)
	AddE2EConfigFlags(genset)
	AddRBACModeFlags(&g.rbacMode, genset, rbac)
	flags.AddFlagSet(genset)

	AddNamespaceFlag(&cfg.Namespace, flags)
	AddSonobuoyImage(&cfg.Image, flags)
}

func (g *genFlags) FillConfig(cmd *cobra.Command, cfg *ops.GenConfig) error {
	cfg.Config = GetConfigWithMode(&g.sonobuoyConfig, g.mode)

	cfg.EnableRBAC = getRBACOrExit(&g.rbacMode, &g.kubecfg)

	e2ecfg, err := GetE2EConfig(genflags.mode, cmd.Flags())
	if err != nil {
		return errors.Wrap(err, "could not retrieve E2E config")
	}
	cfg.E2EConfig = e2ecfg

	return nil
}

// GenCommand is exported so it can be extended.
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "Generates a sonobuoy manifest for submission via kubectl",
	Run:   genManifest,
	Args:  cobra.ExactArgs(0),
}

func init() {
	// Default to enabled here so we don't need a kubeconfig by default
	genflags.AddFlags(GenCommand.Flags(), &genopts, EnabledRBACMode)

	RootCmd.AddCommand(GenCommand)
}

func genManifest(cmd *cobra.Command, args []string) {
	if err := genflags.FillConfig(cmd, &genopts); err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	bytes, err := ops.NewSonobuoyClient().GenerateManifest(&genopts)

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
