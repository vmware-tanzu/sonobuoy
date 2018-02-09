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

	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/utils/image"
	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/utils/mode"
	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/utils/namespace"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var runopts ops.RunConfig

func init() {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Submits a sonobuoy run",
		Run:   submitSonobuoyRun,
	}
	mode.AddFlag(&runopts.GenConfig.ModeName, cmd)
	image.AddFlag(&runopts.GenConfig.Image, cmd)
	namespace.AddFlag(&runopts.GenConfig.Namespace, cmd)

	RootCmd.AddCommand(cmd)
}

func submitSonobuoyRun(cmd *cobra.Command, args []string) {
	if err := ops.Run(runopts); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}
	os.Exit(0)
}
