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
	"github.com/spf13/pflag"

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

type runFlags struct {
	genFlags
}

var runopts ops.RunConfig
var runflags runFlags

func (r *runFlags) AddFlags(flags *pflag.FlagSet, cfg *ops.RunConfig) {
	runset := pflag.NewFlagSet("run", pflag.ExitOnError)
	// Default to detect since we need kubeconfig regardless
	runflags.genFlags.AddFlags(runset, &runopts.GenConfig, DetectRBACMode)
	AddSkipPreflightFlag(&runopts.SkipPreflight, runset)
	flags.AddFlagSet(runset)
}

func (r *runFlags) FillConfig(cfg *ops.RunConfig) error {
	return r.genFlags.FillConfig(&runopts.GenConfig)
}

func init() {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submits a sonobuoy run",
		Run:   submitSonobuoyRun,
		Args:  cobra.ExactArgs(0),
	}

	runflags.AddFlags(cmd.Flags(), &runopts)
	RootCmd.AddCommand(cmd)
}

func submitSonobuoyRun(cmd *cobra.Command, args []string) {
	restConfig, err := runflags.kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
		os.Exit(1)
	}

	if err := runflags.FillConfig(&runopts); err != nil {
		errlog.LogError(errors.Wrap(err, "could not retrieve E2E config"))
		os.Exit(1)
	}

	// TODO(timothysc) Need to add checks which include (detection-rbac, preflight-DNS, ...)
	if err := ops.NewSonobuoyClient().Run(&runopts, restConfig); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}
}
