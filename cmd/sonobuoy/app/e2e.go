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
	"compress/gzip"
	"fmt"
	"os"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type e2eFlags struct {
	runFlags
	show  string
	rerun bool
}

func E2EFlagSet(cfg *e2eFlags) *pflag.FlagSet {
	e2eset := pflag.NewFlagSet("e2e", pflag.ExitOnError)
	e2eset.AddFlagSet(RunFlagSet(&cfg.runFlags))

	e2eset.StringVar(&cfg.show, "show", "failed", "Defines which tests to show, options are [passed, failed (default) or all]. Cannot be combined with --rerun-failed.")
	e2eset.BoolVar(&cfg.rerun, "rerun-failed", false, "Rerun the failed tests reported by the archive. The --show flag will be ignored.")

	return e2eset
}

var e2eflags e2eFlags

func NewCmdE2E() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "e2e archive.tar.gz",
		Short: "Inspect e2e test results. Optionally rerun failed tests",
		Run:   e2es,
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().AddFlagSet(E2EFlagSet(&e2eflags))

	return cmd
}

func e2es(cmd *cobra.Command, args []string) {
	f, err := os.Open(args[0])
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not open sonobuoy archive: %v", args[0]))
		os.Exit(1)
	}
	defer f.Close()
	// As documented, ignore show if we are doing a rerun of failed tests.
	if e2eflags.rerun {
		e2eflags.show = "failed"
	}
	gzr, err := gzip.NewReader(f)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not make a gzip reader"))
		os.Exit(1)
	}
	defer gzr.Close()

	var cfg *rest.Config
	// If we are doing a rerun, only then, we need kubeconfig
	if e2eflags.rerun {
		cfg, err = e2eflags.kubecfg.Get()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
			os.Exit(1)
		}
	}
	sonobuoy, err := getSonobuoyClient(cfg)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}
	testCases, err := sonobuoy.GetTests(gzr, e2eflags.show)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not get tests from archive"))
		os.Exit(1)
	}

	// If we are not doing a rerun, print and exit.
	if !e2eflags.rerun {
		numTests := len(testCases)
		fmt.Printf("%v tests: %v\n", e2eflags.show, numTests)
		if numTests > 0 {
			fmt.Println(client.PrintableTestCases(testCases))
		}
		return
	}

	runCfg, err := e2eflags.Config()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't make a Run config"))
		os.Exit(1)
	}

	if !e2eflags.skipPreflight {
		errs := sonobuoy.PreflightChecks(&client.PreflightConfig{
			Namespace: e2eflags.namespace,
		})
		if len(errs) > 0 {
			errlog.LogError(errors.New("Preflight checks failed"))
			for _, err := range errs {
				errlog.LogError(err)
			}
			os.Exit(1)
		}
	}

	fmt.Printf("Rerunning %d tests:\n", len(testCases))
	if err := sonobuoy.Run(runCfg); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to rerun failed tests"))
		os.Exit(1)
	}
}
