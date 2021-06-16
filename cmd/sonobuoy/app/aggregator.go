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

	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/discovery"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type aggregatorInput struct {
	noExit  bool
	kubecfg Kubeconfig
}

// NewCmdAggregator returns the command that runs Sonobuoy as an aggregator. It will
// load the config, launch plugins, gather results, and query the cluster for data.
func NewCmdAggregator() *cobra.Command {
	input := aggregatorInput{}
	cmd := &cobra.Command{
		Use:   "aggregator",
		Short: "Runs the aggregator component (for internal use)",
		Long:  "Sonobuoy is an introspective kubernetes component that generates reports on cluster conformance, configuration, and more",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runAggregator(&input); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
		Hidden: true,
		Args:   cobra.ExactArgs(0),
	}
	cmd.PersistentFlags().BoolVar(
		&input.noExit, "no-exit", false,
		"Use this if you want sonobuoy to block and not exit. Useful when you want to explicitly grab results.tar.gz",
	)
	AddKubeconfigFlag(&input.kubecfg, cmd.Flags())
	return cmd
}

func runAggregator(input *aggregatorInput) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return errors.Wrap(err, "error loading sonobuoy configuration")
	}

	kcfg, err := input.kubecfg.Get()
	if err != nil {
		return errors.Wrap(err, "getting kubeconfig")
	}

	// Run Discovery (gather API data, run plugins)
	errcount := discovery.Run(kcfg, cfg)

	if input.noExit {
		logrus.Info("no-exit was specified, sonobuoy is now blocking")
		select {}
	}

	if errcount > 0 {
		return fmt.Errorf("%v errors encountered during execution", errcount)
	}

	return nil
}
