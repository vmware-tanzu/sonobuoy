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

	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

var allowedGenFlagsWithRunFile = []string{kubeconfig, kubecontext}

func givenAnyGenConfigFlags(fs *pflag.FlagSet, allowedFlagNames []string) bool {
	changed := false
	fs.Visit(func(f *pflag.Flag) {
		if changed {
			return
		}
		if f.Changed && !stringInList(allowedFlagNames, f.Name) {
			changed = true
		}
	})
	return changed
}

func NewCmdRun() *cobra.Command {
	var f genFlags
	fs := GenFlagSet(&f, DetectRBACMode)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Starts a Sonobuoy run by launching the Sonobuoy aggregator and plugin pods.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkFlagValidity(fs, f)
		},
		Run:  submitSonobuoyRun(&f),
		Args: cobra.ExactArgs(0),
	}

	cmd.Flags().AddFlagSet(fs)
	return cmd
}

func checkFlagValidity(fs *pflag.FlagSet, rf genFlags) error {
	if rf.genFile != "" && givenAnyGenConfigFlags(fs, allowedGenFlagsWithRunFile) {
		return fmt.Errorf("setting the --file flag is incompatible with any other options besides %v", allowedGenFlagsWithRunFile)
	}
	return nil
}

func submitSonobuoyRun(f *genFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		sbc, err := getSonobuoyClientFromKubecfg(f.kubecfg)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		runCfg, err := f.RunConfig()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not retrieve E2E config"))
			os.Exit(1)
		}

		if !contains(f.skipPreflight, "true") && !contains(f.skipPreflight, "*") {
			pcfg := &client.PreflightConfig{
				Namespace:           f.sonobuoyConfig.Namespace,
				DNSNamespace:        f.dnsNamespace,
				DNSPodLabels:        f.dnsPodLabels,
				PreflightChecksSkip: f.skipPreflight,
			}
			if errs := sbc.PreflightChecks(pcfg); len(errs) > 0 {
				errlog.LogError(errors.New("Preflight checks failed"))
				for _, err := range errs {
					errlog.LogError(err)
				}
				os.Exit(1)
			}
		}

		if err := sbc.Run(runCfg); err != nil {
			errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
			os.Exit(1)
		}
	}
}

func stringInList(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
