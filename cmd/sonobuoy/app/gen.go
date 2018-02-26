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
	"k8s.io/client-go/kubernetes"

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

var genopts ops.GenConfig

var genFlags struct {
	sonobuoyConfig SonobuoyConfig
	mode           ops.Mode
	rbacMode       RBACMode
	kubecfg        Kubeconfig
}

// GenCommand is exported so it can be extended.
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "Generates a sonobuoy manifest for submission via kubectl",
	Run:   genManifest,
	Args:  cobra.ExactArgs(0),
}

func init() {
	AddGenFlags(&genopts, GenCommand)

	AddModeFlag(&genFlags.mode, GenCommand)
	AddSonobuoyConfigFlag(&genFlags.sonobuoyConfig, GenCommand)
	AddKubeconfigFlag(&genFlags.kubecfg, GenCommand)
	AddE2EConfig(GenCommand)
	// Default to enabled here so we don't need a kubeconfig by default
	AddRBACMode(&genFlags.rbacMode, GenCommand, EnabledRBACMode)

	RootCmd.AddCommand(GenCommand)
}

// AddGenFlags adds generation flags to a command.
func AddGenFlags(gen *ops.GenConfig, cmd *cobra.Command) {
	AddNamespaceFlag(&gen.Namespace, cmd)
	AddSonobuoyImage(&gen.Image, cmd)
}

func genManifest(cmd *cobra.Command, args []string) {
	genopts.Config = GetConfigWithMode(&genFlags.sonobuoyConfig, genFlags.mode)

	e2ecfg, err := GetE2EConfig(genFlags.mode, cmd)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not retrieve E2E config"))
		os.Exit(1)
	}
	genopts.E2EConfig = e2ecfg

	genopts.EnableRBAC = getRBACOrExit(&genFlags.rbacMode, &genFlags.kubecfg)

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

	rbacEnabled, err := genFlags.rbacMode.Get(client)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't detect RBAC mode."))
		if errors.Cause(err) == RBACErrorNoClient {
			errlog.LogError(kubeError)
		}
		os.Exit(1)
	}
	return rbacEnabled
}
