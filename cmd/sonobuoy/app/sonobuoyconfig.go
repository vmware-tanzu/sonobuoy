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
	"reflect"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
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

// Set attempts to read a file, then deserialize the json into a config.Config struct.
func (c *SonobuoyConfig) Set(str string) error {
	if !reflect.DeepEqual(c.Config, *config.New()) {
		return errors.New("if a custom config file is set, it must be set before other flags that modify configuration fields")
	}

	bytes, err := ioutil.ReadFile(str)
	if err != nil {
		return errors.Wrap(err, "couldn't open config file")
	}

	if err := json.Unmarshal(bytes, &c.Config); err != nil {
		return errors.Wrap(err, "couldn't Unmarshal sonobuoy config")
	}

	c.raw = string(bytes)
	return nil
}

// Get will return the config.Config.
func (c *SonobuoyConfig) Get() *config.Config {
	return &c.Config
}
