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
	"compress/gzip"
	"fmt"
	"os"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

var e2eFlags struct {
	show  string
	rerun bool
}

func init() {
	cmd := &cobra.Command{
		Use:   "e2e archive.tar.gz",
		Short: "Inspect e2e test results. Optionally rerun failed tests",
		Run:   e2es,
		Args:  cobra.ExactArgs(1),
	}
	AddGenFlags(&runopts.GenConfig, cmd)
	AddModeFlag(&runFlags.mode, cmd)
	AddSonobuoyConfigFlag(&runFlags.sonobuoyConfig, cmd)
	AddKubeconfigFlag(&runFlags.kubecfg, cmd)
	// Default to detect since we need a kubeconfig regardless
	AddRBACModeFlags(&runFlags.rbacMode, cmd, DetectRBACMode)
	AddSkipPreflightFlag(&runopts.SkipPreflight, cmd)

	cmd.PersistentFlags().StringVar(&e2eFlags.show, "show", "failed", "Defines which tests to show, options are [passed, failed (default) or all]. Cannot be combined with --rerun-failed.")
	cmd.PersistentFlags().BoolVar(&e2eFlags.rerun, "rerun-failed", false, "Rerun the failed tests reported by the archive. The --show flag will be ignored.")

	RootCmd.AddCommand(cmd)
}

func e2es(cmd *cobra.Command, args []string) {
	f, err := os.Open(args[0])
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not open sonobuoy archive: %v", args[0]))
		os.Exit(1)
	}
	defer f.Close()
	// As documented, ignore show if we are doing a rerun of failed tests.
	if e2eFlags.rerun {
		e2eFlags.show = "failed"
	}
	gzr, err := gzip.NewReader(f)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not make a gzip reader"))
		os.Exit(1)
	}
	defer gzr.Close()
	sonobuoy := client.NewSonobuoyClient()
	testCases, err := sonobuoy.GetTests(gzr, e2eFlags.show)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not get tests from archive"))
		os.Exit(1)
	}

	// If we are not doing a rerun, print and exit.
	if !e2eFlags.rerun {
		fmt.Printf("%v tests\n", e2eFlags.show)
		fmt.Println(client.PrintableTestCases(testCases))
		return
	}

	restConfig, err := runFlags.kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't create Kubernetes clientset"))
		os.Exit(1)
	}

	rbacEnabled, err := runFlags.rbacMode.Enabled(clientset)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't detect RBAC mode"))
		os.Exit(1)
	}
	runopts.GenConfig.EnableRBAC = rbacEnabled

	runopts.Config = GetConfigWithMode(&runFlags.sonobuoyConfig, runFlags.mode)
	runopts.E2EConfig = &client.E2EConfig{
		Focus: client.Focus(testCases),
		Skip:  "",
	}

	fmt.Printf("Rerunning %d tests:\n", len(testCases))
	if err := sonobuoy.Run(&runopts, restConfig); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to rerun failed tests"))
		os.Exit(1)
	}
}
