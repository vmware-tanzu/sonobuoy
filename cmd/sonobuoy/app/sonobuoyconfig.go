package app

import (
	"encoding/json"
	"io/ioutil"

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

// GetConfigWithMode creates a config with the following algorithim:
// If the SonobuoyConfig isn't nil, use that
// If not, use the supplied Mode to modify a default config
func GetConfigWithMode(sonobuoyCfg *SonobuoyConfig, mode client.Mode) *config.Config {
	suppliedConfig := sonobuoyCfg.Get()
	if suppliedConfig != nil {
		return suppliedConfig
	}

	defaultConfig := config.NewWithDefaults()
	modeConfig := mode.Get()
	if modeConfig != nil {
		defaultConfig.PluginSelections = modeConfig.Selectors
	}
	return defaultConfig
}
