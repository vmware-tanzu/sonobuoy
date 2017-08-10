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

package worker

import (
	"os"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

func setConfigDefaults(ac *plugin.WorkerConfig) {
	ac.ResultsDir = "/tmp/results"
}

// LoadConfig loads the configuration for the sonobuoy worker from environment
// variables, returning a plugin.WorkerConfig struct with defaults applied
func LoadConfig() (*plugin.WorkerConfig, error) {
	config := &plugin.WorkerConfig{}
	var err error

	viper.SetConfigType("json")
	viper.SetConfigName("worker")
	viper.AddConfigPath("/etc/sonobuoy")
	viper.AddConfigPath(".")

	// Allow specifying a custom config file via the SONOBUOY_CONFIG env var
	if forceCfg := os.Getenv("SONOBUOY_CONFIG"); forceCfg != "" {
		viper.SetConfigFile(forceCfg)
	}

	viper.BindEnv("masterurl", "MASTER_URL")
	viper.BindEnv("nodename", "NODE_NAME")
	viper.BindEnv("resultsdir", "RESULTS_DIR")

	setConfigDefaults(config)

	if err = viper.ReadInConfig(); err != nil {
		return nil, errors.WithStack(err)
	}

	if err = viper.Unmarshal(config); err != nil {
		return nil, errors.WithStack(err)
	}

	return config, nil
}
