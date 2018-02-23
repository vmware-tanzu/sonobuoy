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

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

var genopts ops.GenConfig

var genFlags struct {
	sonobuoyConfig SonobuoyConfig
	mode           ops.Mode
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
	AddE2EConfig(GenCommand)

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

	bytes, err := ops.NewSonobuoyClient().GenerateManifest(&genopts)

	if err == nil {
		fmt.Printf("%s\n", bytes)
		return
	}
	errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
	os.Exit(1)
}
