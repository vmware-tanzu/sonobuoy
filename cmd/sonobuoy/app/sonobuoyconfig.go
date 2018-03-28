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
	"encoding/json"
	"io/ioutil"

	"github.com/imdario/mergo"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

// SonobuoyConfig is a config.Config that implements pflag.Value from a file path
type SonobuoyConfig struct {
	config.Config
	raw string
}

var _ pflag.Value = &SonobuoyConfig{}

// String is needed for pflag.Value.
func (c *SonobuoyConfig) String() string {
	return c.raw
}

// Type is needed for pflag.Value.
func (c *SonobuoyConfig) Type() string { return "Sonobuoy config" }

// Set attempts to read a file, then deserialise the json into a config.Config struct.
func (c *SonobuoyConfig) Set(str string) error {
	bytes, err := ioutil.ReadFile(str)
	if err != nil {
		return errors.Wrap(err, "cloudn't open config file")
	}

	if err := json.Unmarshal(bytes, &c.Config); err != nil {
		return errors.Wrap(err, "couldn't Unmarshal sonobuoy config")
	}

	c.raw = string(bytes)
	return nil
}

// Get will return the config.Config if one is available, otherwise nil.
func (c *SonobuoyConfig) Get() *config.Config {
	// Don't just return zero structs
	if c.raw == "" {
		return nil
	}

	return &c.Config
}

// GetConfigWithMode creates a config with the following algorithm:
// If no config is supplied defaults will be returned.
// If a config is supplied then the default values will be merged into the supplied config
// in order to allow users to supply a minimal config that will still work.
func GetConfigWithMode(sonobuoyCfg *SonobuoyConfig, mode client.Mode) *config.Config {
	conf := config.New()

	suppliedConfig := sonobuoyCfg.Get()
	if suppliedConfig != nil {
		// Provide defaults but don't overwrite any customized configuration.
		mergo.Merge(suppliedConfig, conf)
		conf = suppliedConfig
	}

	// if there are no plugins yet, set some based on the mode, otherwise use whatever was supplied.
	if len(conf.PluginSelections) == 0 {
		modeConfig := mode.Get()
		if modeConfig != nil {
			conf.PluginSelections = modeConfig.Selectors
		}
	}
	return conf
}
