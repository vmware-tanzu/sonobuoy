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
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

var runopts ops.RunConfig

var runFlags struct {
	sonobuoyConfig SonobuoyConfig
	mode           ops.Mode
	kubecfg        Kubeconfig
	rbacMode       RBACMode
}

func init() {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submits a sonobuoy run",
		Run:   submitSonobuoyRun,
		Args:  cobra.ExactArgs(0),
	}
	AddGenFlags(&runopts.GenConfig, cmd)

	AddModeFlag(&runFlags.mode, cmd)
	AddSonobuoyConfigFlag(&runFlags.sonobuoyConfig, cmd)
	AddE2EConfig(cmd)
	AddKubeconfigFlag(&runFlags.kubecfg, cmd)
	// Default to detect since we need a kubeconfig regardless
	AddRBACMode(&runFlags.rbacMode, cmd, DetectRBACMode)

	RootCmd.AddCommand(cmd)
}

func submitSonobuoyRun(cmd *cobra.Command, args []string) {
	restConfig, err := runFlags.kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't create Kubernetes client"))
		os.Exit(1)
	}

	rbacEnabled, err := runFlags.rbacMode.Get(client)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't detect RBAC mode"))
		os.Exit(1)
	}
	runopts.GenConfig.EnableRBAC = rbacEnabled

	runopts.Config = GetConfigWithMode(&runFlags.sonobuoyConfig, runFlags.mode)
	e2ecfg, err := GetE2EConfig(runFlags.mode, cmd)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not retrieve E2E config"))
		os.Exit(1)
	}
	genopts.E2EConfig = e2ecfg

	// TODO(timothysc) Need to add checks which include (detection-rbac, preflight-DNS, ...)
	if err := ops.NewSonobuoyClient().Run(&runopts, restConfig); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}
}
