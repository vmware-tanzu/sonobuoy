/*
Copyright 2021 Sonobuoy Contributors

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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/discovery"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

type queryInput struct {
	kubecfg Kubeconfig
	cfg     *SonobuoyConfig
	outDir  string
}

func NewCmdQuery() *cobra.Command {
	// Default to an empty config for CLI users but allow us to load
	// it from disc for the in-cluster case.
	input := queryInput{
		cfg: &SonobuoyConfig{},
	}
	input.cfg.Config = *config.New()
	input.cfg.Config.Resolve()

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Runs queries against your cluster in order to aid in debugging.",
		Run: func(cmd *cobra.Command, args []string) {
			restConf, err := input.kubecfg.Get()
			if err != nil {
				errlog.LogError(errors.Wrap(err, "getting kubeconfig"))
				os.Exit(1)
			}

			// Override the query results directory. Since config is a param too just avoid complication
			// and override the default value here.
			if len(input.outDir) > 0 {
				input.cfg.QueryDir = input.outDir
			} else {
				// UUID instead of default (aggregatorResultsPath) since we need to be OS-agnostic.
				input.cfg.QueryDir = input.cfg.UUID
			}

			logrus.Tracef("Querying using config %#v", &input.cfg.Config)
			logrus.Tracef("Query results will be placed in %v", input.cfg.Config.QueryOutputDir())
			if err := discovery.QueryCluster(restConf, &input.cfg.Config); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}

			// Print the output directory for scripting.
			fmt.Println(input.cfg.Config.QueryOutputDir())
		},
	}

	AddSonobuoyConfigFlag(input.cfg, cmd.Flags())
	cmd.Flags().StringVarP(&input.outDir, "output", "o", "", "Directory to output results into. If empty, will default to a UUID folder in the pwd.")

	return cmd
}
