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
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

func NewCmdWait() *cobra.Command {
	var f genFlags
	fs := GenFlagSet(&f, DetectRBACMode)
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Waits on the Sonobuoy run in the targeted namespace.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkFlagValidity(fs, f)
		},
		Run:  waitOnRun(&f),
		Args: cobra.ExactArgs(0),
	}

	cmd.Flags().AddFlagSet(fs)
	return cmd
}

func waitOnRun(f *genFlags) func(cmd *cobra.Command, args []string) {
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

		if err := sbc.WaitForRun(runCfg); err != nil {
			errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
			os.Exit(1)
		}
	}
}
