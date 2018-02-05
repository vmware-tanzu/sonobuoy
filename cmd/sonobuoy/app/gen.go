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

	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/utils/mode"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var genopts ops.GenConfig

// GenCommand is exported so it can be extended
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "Generates a sonobuoy manifest for submission via kubectl",
	Run:   genManifest,
}

func init() {
	GenCommand.PersistentFlags().StringVar(
		&genopts.Path, "path", "./",
		"TBD: location to output",
	)

	GenCommand.PersistentFlags().StringVar(
		&genopts.Image, "sonobuoy-image",
		"gcr.io/heptio-images/sonobuoy:latest",
		"The Docker image (as a registry URL) to use for the Sonobuoy controller",
	)

	mode.AddFlag(&genopts.ModeName, GenCommand)

	RootCmd.AddCommand(GenCommand)
}

func genManifest(cmd *cobra.Command, args []string) {
	code := 0
	bytes, err := ops.GenerateManifest(genopts)
	if err == nil {
		fmt.Printf("%s\n", bytes)
	} else {
		errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
		code = 1
	}
	os.Exit(code)
}
