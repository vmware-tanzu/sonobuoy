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
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var genPluginOpts ops.GenPluginConfig

func init() {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Generates the manifest Sonobuoy uses to run a worker for the given plugin",
		Run:   genPluginManifest,
		Args:  cobra.ExactArgs(1),
	}

	GenCommand.PersistentFlags().StringArrayVarP(
		&genPluginOpts.Paths, "paths", "p", []string{".", "./plugins.d/"},
		"the paths to search for the plugins in. Defaults to . and ./plugins.d/",
	)
	// TODO: Other options?
	GenCommand.AddCommand(cmd)
}

func genPluginManifest(cmd *cobra.Command, args []string) {
	genPluginOpts.PluginName = args[0]
	code := 0
	manifest, err := ops.GeneratePluginManifest(genPluginOpts)
	if err == nil {
		fmt.Printf("%s\n", manifest)
	} else {
		errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
		code = 1
	}
	os.Exit(code)
}
