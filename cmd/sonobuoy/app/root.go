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
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

// once used just to initKlogs one time; the `gen cli` command will hit that path a second time
// and cause a panic otherwise.
var once sync.Once

func NewSonobuoyCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:               "sonobuoy",
		Short:             "Generate reports on your Kubernetes cluster by running plugins",
		Long:              "Sonobuoy is a Kubernetes component that generates reports on cluster conformance, configuration, and more",
		PersistentPreRunE: prerunChecks,
		Run:               rootCmd,
	}

	cmds.ResetFlags()

	cmds.AddCommand(NewCmdAggregator())
	cmds.AddCommand(NewCmdDelete())

	gen := NewCmdGen()
	genPlugin := NewCmdGenPluginDef()
	genPlugin.AddCommand(NewCmdGenE2E())
	genPlugin.AddCommand(NewCmdGenSystemdLogs())
	gen.AddCommand(genPlugin)
	gen.AddCommand(NewCmdGenConfig())
	gen.AddCommand(NewCmdGenImageRepoConfig())
	gen.AddCommand(NewCmdGenCLIDocs())
	cmds.AddCommand(gen)

	cmds.AddCommand(NewCmdLogs())
	cmds.AddCommand(NewCmdVersion())
	cmds.AddCommand(NewCmdStatus())
	cmds.AddCommand(NewCmdWorker())
	cmds.AddCommand(NewCmdRetrieve())
	cmds.AddCommand(NewCmdRun())
	cmds.AddCommand(NewCmdImages())
	cmds.AddCommand(NewCmdResults())
	cmds.AddCommand(NewCmdSplat())
	cmds.AddCommand(NewCmdWait())

	cmds.AddCommand(NewCmdQuery())
	cmds.AddCommand(NewCmdModes())

	cmds.AddCommand(NewCmdE2E())

	cmds.AddCommand(NewCmdPlugin())

	initKlog(cmds)
	cmds.PersistentFlags().Var(&errlog.LogLevel, "level", "Log level. One of {panic, fatal, error, warn, info, debug, trace}")

	// Previously just had debug flag but in desire to have fine grained control over output we opted to
	// have full ability to set level instead.
	cmds.PersistentFlags().BoolVarP(&errlog.DebugOutput, "debug", "d", false, "Enable debug output (includes stack traces)")
	if err := cmds.PersistentFlags().MarkHidden("debug"); err != nil {
		panic(err)
	}
	if err := cmds.PersistentFlags().MarkDeprecated("debug", "Use --level flag instead."); err != nil {
		panic(err)
	}

	return cmds

}

func rootCmd(cmd *cobra.Command, args []string) {
	// Sonobuoy does nothing when not given a subcommand
	cmd.Help()
}

// prerunChecks can be a kitchen sink of little checks. Since we have the command
// object we can get all the flags and do any complicated flag logic here.
func prerunChecks(cmd *cobra.Command, args []string) error {
	// Getting a list of all flags provided by the user.
	flagsSet := map[string]bool{}
	flagsDebug := []string{}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		flagsSet[f.Name] = true
		flagsDebug = append(flagsDebug, fmt.Sprintf("%v=%v", f.Name, f.Value.String()))
	})

	logrus.Tracef("Invoked command %v with args %v and flags %v", cmd.Name(), args, flagsDebug)

	// Difficult to do checks like this within the flag themselves (since they dont know
	// about each other). Splitting up checks into varous 'ifs' for ease of reading/writing even if not most succinct.
	if flagsSet["mode"] && (flagsSet["e2e-focus"] || flagsSet["e2e-skip"]) {
		logrus.Warnf("mode flag and e2e-focus/skip flags both provided and may cause unintended behavior")
	}

	if flagsSet["rerun-failed"] && (flagsSet["e2e-focus"] || flagsSet["e2e-skip"]) {
		logrus.Warnf("rerun-failed flag and e2e-focus/skip flags both provided and may cause unintended behavior")
	}

	if flagsSet["rerun-failed"] && flagsSet["mode"] {
		logrus.Warnf("rerun-failed flag and mode flags both provided and may cause unintended behavior")
	}

	if flagsSet["kube-conformance-image"] && (flagsSet["kubernetes-version"] || flagsSet["kube-conformance-image-version"]) {
		logrus.Warnf("kube-conformance-image flag and kubernetes-version/kube-conformance-image-version flags both set and may collide")
	}

	if flagsSet[e2eRegistryConfigFlag] && flagsSet[e2eRegistryFlag] {
		logrus.Warnf("%v and %v flags are both set and may collide", e2eRegistryConfigFlag, e2eRegistryFlag)
	}

	return nil
}

// initKlog flags but mark them hidden since they just make the help
// more verbose and dont directly speak to the sonobuoy flags themselves.
// Still usable if truly necessary.
func initKlog(cmd *cobra.Command) {
	initKlogsOnce := func() {
		klog.InitFlags(nil)
	}
	once.Do(initKlogsOnce)

	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	for _, f := range []string{
		"log_dir",
		"log_file",
		"log_file_max_size",
		"logtostderr",
		"alsologtostderr",
		"v",
		"add_dir_header",
		"skip_headers",
		"skip_log_headers",
		"stderrthreshold",
		"vmodule",
		"log_backtrace_at",
	} {
		if err := cmd.PersistentFlags().MarkHidden(f); err != nil {
			panic(err)
		}
	}
}
