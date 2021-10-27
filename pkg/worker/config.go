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

package worker

import (
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const (
	defaultProgressUpdatesPort = "8099"
)

// LoadConfig loads the configuration for the sonobuoy worker from environment
// variables, returning a plugin.WorkerConfig struct with defaults applied
func LoadConfig() (*plugin.WorkerConfig, error) {
	config := &plugin.WorkerConfig{}
	var err error

	viper.BindEnv("aggregatorurl", "AGGREGATOR_URL")
	viper.BindEnv("nodename", "NODE_NAME")
	viper.BindEnv("resultsdir", "SONOBUOY_RESULTS_DIR")
	viper.BindEnv("resulttype", "RESULT_TYPE")

	viper.BindEnv("cacert", "CA_CERT")
	viper.BindEnv("clientcert", "CLIENT_CERT")
	viper.BindEnv("clientkey", "CLIENT_KEY")
	viper.BindEnv("progressport", "SONOBUOY_PROGRESS_PORT")

	viper.SetDefault("resultsdir", plugin.ResultsDir)
	viper.SetDefault("progressport", defaultProgressUpdatesPort)

	if err = viper.Unmarshal(config); err != nil {
		return nil, errors.WithStack(err)
	}

	return config, nil
}
