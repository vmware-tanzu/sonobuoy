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
	"strings"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/worker"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	workerCmd.AddCommand(singleNodeCmd)
	workerCmd.AddCommand(globalCmd)

	RootCmd.AddCommand(workerCmd)
}

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Gather and send data to the sonobuoy master instance",
	Run:   runGather,
}

var globalCmd = &cobra.Command{
	Use:   "global",
	Short: "Submit results scoped to the whole cluster",
	Run:   runGatherGlobal,
}

var singleNodeCmd = &cobra.Command{
	Use:   "single-node",
	Short: "Submit results scoped to a single node",
	Run:   runGatherSingleNode,
}

func runGather(cmd *cobra.Command, args []string) {
	cmd.Help()
}

// loadAndValidateConfig loads the config for this sonobuoy worker, validating
// that we have enough information to proceed.
func loadAndValidateConfig() (*plugin.WorkerConfig, error) {
	cfg, err := worker.LoadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "error loading agent configuration")
	}

	var errlst []string
	if cfg.MasterURL == "" {
		errlst = append(errlst, "MasterURL not set")
	}
	if cfg.ResultsDir == "" {
		errlst = append(errlst, "ResultsDir not set")
	}
	if cfg.ResultType == "" {
		errlst = append(errlst, "ResultsType not set")
	}

	if len(errlst) > 0 {
		joinedErrs := strings.Join(errlst, ", ")
		return nil, errors.Errorf("invalid agent configuration: (%v)", joinedErrs)
	}

	return cfg, nil
}

func runGatherSingleNode(cmd *cobra.Command, args []string) {
	cfg, err := loadAndValidateConfig()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	// A single-node results URL looks like:
	// http://sonobuoy-master:8080/api/v1/results/by-node/node1/systemd_logs
	url := cfg.MasterURL + "/" + cfg.NodeName + "/" + cfg.ResultType

	err = worker.GatherResults(cfg.ResultsDir+"/done", url)
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

}

func runGatherGlobal(cmd *cobra.Command, args []string) {
	cfg, err := loadAndValidateConfig()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	// A global results URL looks like:
	// http://sonobuoy-master:8080/api/v1/results/global/systemd_logs
	url := cfg.MasterURL + "/" + cfg.ResultType

	err = worker.GatherResults(cfg.ResultsDir+"/done", url)
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
}
