package app

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"k8s.io/test-infra/prow/config"
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
