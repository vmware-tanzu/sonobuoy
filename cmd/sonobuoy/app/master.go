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
	"os"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/discovery"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var noExit bool

func init() {
	cmd := &cobra.Command{
		Use:   "master",
		Short: "Generate reports on your kubernetes cluster",
		Long:  "Sonobuoy is an introspective kubernetes component that generates reports on cluster conformance, configuration, and more",
		Run:   runMaster,
	}
	cmd.PersistentFlags().BoolVar(
		&noExit, "no-exit", false,
		"Use this if you want sonobuoy to block and not exit. Useful when you want to explicitly grab results.tar.gz",
	)
	RootCmd.AddCommand(cmd)
}

func runMaster(cmd *cobra.Command, args []string) {
	exit := 0

	cfg, err := config.LoadConfig()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error loading sonobuoy configuration"))
		os.Exit(1)
	}

	// Load a kubernetes client
	kubeClient, err := config.LoadClient(cfg)
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	// Run Discovery (gather API data, run plugins)
	if errcount := discovery.Run(kubeClient, cfg); errcount > 0 {
		exit = 1
	}

	if noExit {
		logrus.Info("no-exit was specified, sonobuoy is now blocking")
		select {}
	}

	os.Exit(exit)
}
