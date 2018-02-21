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
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

var genopts ops.GenConfig
var mode string

// GenCommand is exported so it can be extended
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "Generates a sonobuoy manifest for submission via kubectl",
	Run:   genManifest,
	Args:  cobra.ExactArgs(0),
}

func init() {
	AddGenFlags(&genopts, GenCommand)
	RootCmd.AddCommand(GenCommand)
}

// AddGenFlags adds generation flags to a command
func AddGenFlags(gen *ops.GenConfig, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		&mode, "e2e-mode", "", string(ops.Conformance),
		fmt.Sprintf("What mode to run sonobuoy in. [%s]", strings.Join(ops.GetModes(), ", ")),
	)
	// TODO(timothysc) This variable default needs saner image defaults from ops.f(n) or config
	cmd.PersistentFlags().StringVarP(
		&gen.Image, "sonobuoy-image", "", "gcr.io/heptio-images/sonobuoy:master",
		"Container image override for the sonobuoy worker and container",
	)
	// TODO(timothysc) This variable default needs saner image defaults from ops.f(n) or config
	cmd.PersistentFlags().StringVarP(
		&gen.Namespace, "namespace", "n", "heptio-sonobuoy",
		"The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously.",
	)
	// TODO(timothysc) Need to provide ability to override config structure and allow for sane defaults
	// TODO(timothysc) Need to provide ability to override e2e-focus
	// TODO(timothysc) Need to provide ability to override e2e-skip
}

func genManifest(cmd *cobra.Command, args []string) {
	err := genopts.ModeName.Set(mode)
	if err == nil {
		bytes, err := genopts.GenerateManifest()
		if err == nil {
			fmt.Printf("%s\n", bytes)
			os.Exit(0)
		}
	}
	errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
	os.Exit(1)
}
