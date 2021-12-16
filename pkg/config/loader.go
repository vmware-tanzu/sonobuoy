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

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/buildinfo"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	pluginloader "github.com/vmware-tanzu/sonobuoy/pkg/plugin/loader"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	defaultCfgFileName  = "config.json"
	fallbackCfgFileName = "/etc/sonobuoy/config.json"
)

// LoadConfig will load the current sonobuoy configuration using the filesystem
// and environment variables, and returns a config object
func LoadConfig(pathsToTry ...string) (*Config, error) {
	cfg := &Config{}

	envCfgFileName := os.Getenv("SONOBUOY_CONFIG")
	if envCfgFileName != "" {
		pathsToTry = append(pathsToTry, envCfgFileName)
	} else {
		pathsToTry = append(pathsToTry, defaultCfgFileName, fallbackCfgFileName)
	}

	jsonFile, fpath, err := openFiles(pathsToTry...)
	if err != nil {
		return nil, errors.Wrap(err, "open config")
	}
	defer jsonFile.Close()
	logrus.Tracef("Loading config from file: %v", fpath)

	b, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read config file %q", fpath)
	}
	err = json.Unmarshal(b, cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal config file %q", fpath)
	}

	cfg.Resolve()

	if err := loadAllPlugins(cfg); err != nil {
		return nil, err
	}

	// 6 - Return any validation errors
	validationErrs := cfg.Validate()
	if len(validationErrs) > 0 {
		errstrs := make([]string, len(validationErrs))
		for i := range validationErrs {
			errstrs[i] = validationErrs[i].Error()
		}

		return nil, errors.Errorf("invalid configuration: %v", strings.Join(errstrs, ", "))
	}

	logrus.Tracef("Config loaded: %#v", *cfg)
	return cfg, err
}

func (cfg *Config) Resolve() {
	// Figure out what address we will tell pods to dial for aggregation
	if cfg.Aggregation.AdvertiseAddress == "" {
		if ip := os.Getenv("SONOBUOY_ADVERTISE_IP"); ip != "" {
			cfg.Aggregation.AdvertiseAddress = fmt.Sprintf("[%v]:%d", ip, cfg.Aggregation.BindPort)
		} else {
			hostname, _ := os.Hostname()
			if hostname != "" {
				cfg.Aggregation.AdvertiseAddress = fmt.Sprintf("%v:%d", hostname, cfg.Aggregation.BindPort)
			}
		}
	}

	cfg.Version = buildinfo.Version

	// Make the results dir overridable with an environment variable
	if resultsDir, ok := os.LookupEnv("RESULTS_DIR"); ok {
		cfg.ResultsDir = resultsDir
	}

	if cfg.UUID == "" {
		cfgUuid, _ := uuid.NewV4()
		cfg.UUID = cfgUuid.String()
	}
}

// Validate returns a list of errors for the configuration, if any are found.
func (cfg *Config) Validate() (errorsList []error) {
	podLogLimits := &cfg.Limits.PodLogs

	if podLogLimits.SinceTime != nil && podLogLimits.SinceSeconds != nil {
		errorsList = append(errorsList, errors.New("Only one of sinceSeconds or sinceTime may be specified."))
	}

	return errorsList
}

// loadAllPlugins takes the given sonobuoy configuration and gives back a
// plugin.Interface for every plugin specified by the configuration.
func loadAllPlugins(cfg *Config) error {
	var plugins []plugin.Interface

	// Load all Plugins
	plugins, err := pluginloader.LoadAllPlugins(
		cfg.Namespace,
		cfg.WorkerImage,
		cfg.ImagePullPolicy,
		cfg.ImagePullSecrets,
		cfg.CustomAnnotations,
		cfg.PluginSearchPath,
		cfg.PluginSelections,
	)
	if err != nil {
		return err
	}

	// Find any selected plugins that weren't loaded
	for _, sel := range cfg.PluginSelections {
		found := false
		for _, p := range plugins {
			if p.GetName() == sel.Name {
				found = true
			}
		}

		if !found {
			return errors.Errorf("Configured plugin %v does not exist", sel.Name)
		}
	}

	for _, p := range plugins {
		cfg.addPlugin(p)
	}

	return nil
}

// openFiles tries opening each of the files given, returning the first file/error
// that either opens correctly or provides an error which is not os.IsNotExist(). The
// string is the filename corresponding to the file/error returned.
func openFiles(paths ...string) (*os.File, string, error) {
	var f *os.File
	var path string
	var err error

	for _, path = range paths {
		logrus.Tracef("Trying path: %v", path)
		f, err = os.Open(path)
		switch {
		case err == nil:
			return f, path, nil
		case err != nil && os.IsNotExist(err):
			logrus.Tracef("File %q does not exist", path)
			continue
		default:
			return nil, path, errors.Wrap(err, "opening config file")
		}
	}

	return f, path, errors.Wrap(err, "opening config file")
}
