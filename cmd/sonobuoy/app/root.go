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
	"flag"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/spf13/cobra"
)

func NewSonobuoyCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "sonobuoy",
		Short: "Generate reports on your kubernetes cluster",
		Long:  "Sonobuoy is an introspective kubernetes component that generates reports on cluster conformance, configuration, and more",
		Run:   rootCmd,
	}

	cmds.ResetFlags()
	cmds.AddCommand(NewCmdMaster())
	cmds.AddCommand(NewCmdDelete())
	cmds.AddCommand(NewCmdE2E())
	cmds.AddCommand(NewCmdGen())
	cmds.AddCommand(NewCmdLogs())
	cmds.AddCommand(NewCmdGenPlugin())
	cmds.AddCommand(NewCmdVersion())
	cmds.AddCommand(NewCmdStatus())
	cmds.AddCommand(NewCmdWorker())
	cmds.AddCommand(NewCmdRetrieve())
	cmds.AddCommand(NewCmdRun())
	cmds.AddCommand(NewCmdGenPlugin())

	cmds.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	cmds.PersistentFlags().BoolVarP(&errlog.DebugOutput, "debug", "d", false, "Enable debug output (includes stack traces)")
	return cmds

}

func rootCmd(cmd *cobra.Command, args []string) {
	// Sonobuoy does nothing when not given a subcommand
	cmd.Help()
}
