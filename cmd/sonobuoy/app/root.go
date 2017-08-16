/*
Copyright 2017 Heptio Inc.

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
	"flag"
	"os"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/spf13/cobra"
)

func init() {
	// import `flag` flags into this command to support glog flags
	RootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	RootCmd.PersistentFlags().BoolVarP(&errlog.DebugOutput, "debug", "d", false, "Enable debug output (includes stack traces)")
}

// RootCmd is the root command that is executed when sonobuoy is run without
// any subcommands.
var RootCmd = &cobra.Command{
	Use:   "sonobuoy",
	Short: "Generate reports on your kubernetes cluster",
	Long:  "Sonobuoy is an introspective kubernetes component that generates reports on cluster conformance, configuration, and more",
	Run:   rootCmd,
}

func rootCmd(cmd *cobra.Command, args []string) {
	// Sonobuoy does nothing when not given a subcommand
	cmd.Help()
	os.Exit(0)
}
