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

	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/args"
	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

var runopts ops.RunConfig

func init() {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submits a sonobuoy run",
		Run:   submitSonobuoyRun,
		Args:  cobra.ExactArgs(0),
	}
	args.AddModeFlag(&runopts.GenConfig.ModeName, cmd)
	args.AddSonobuoyImageFlag(&runopts.GenConfig.Image, cmd)
	args.AddNamespaceFlag(&runopts.GenConfig.Namespace, cmd)
	args.AddKubeconfigFlag(&runopts.Kubecfg, cmd)

	RootCmd.AddCommand(cmd)
}

func submitSonobuoyRun(cmd *cobra.Command, args []string) {
	restConfig, err := runopts.Kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
		os.Exit(1)
	}

	if err := ops.Run(runopts, restConfig); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}
	os.Exit(0)
}
