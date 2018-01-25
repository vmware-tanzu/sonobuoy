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

package loader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/job"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

// LoadAllPlugins loads all plugins by finding plugin definitions in the given
// directory, taking a user's plugin selections, and a sonobuoy phone home
// address (host:port) and returning all of the active, configured plugins for
// this sonobuoy run.
func LoadAllPlugins(namespace string, searchPath []string, selections []plugin.Selection) (ret []plugin.Interface, err error) {
	pluginFiles := []string{}
	for _, dir := range searchPath {
		wd, _ := os.Getwd()
		logrus.Infof("Scanning plugins in %v (pwd: %v)", dir, wd)

		// We only care about configured plugin directories that exist,
		// since we may have a broad search path.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Infof("Directory (%v) does not exist", dir)
			continue
		}

		files, err := findPlugins(dir)
		if err != nil {
			return []plugin.Interface{}, errors.Wrapf(err, "couldn't scan %v for plugins", dir)
		}
		pluginFiles = append(pluginFiles, files...)
	}

	pluginDefs := []*pluginDefinition{}
	for _, file := range pluginFiles {
		pluginDef, err := loadDefinition(file)
		if err != nil {
			return []plugin.Interface{}, errors.Wrapf(err, "couldn't load plugin definition %v", file)
		}
		pluginDefs = append(pluginDefs, pluginDef)
	}

	pluginDefs = filterPluginDef(pluginDefs, selections)

	plugins := []plugin.Interface{}
	for _, def := range pluginDefs {
		pluginIface, err := loadPlugin(def, namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "couldn't load plugin %v", def.SonobuoyConfig.PluginName)
		}
		plugins = append(plugins, pluginIface)
	}

	return plugins, nil
}

// loadPlugin loads an individual plugin by instantiating a plugin driver with
// the settings from the given plugin definition and selection
// func loadPlugin(namespace string, dfn plugin.Definition, masterAddress string) (plugin.Interface, error) {
// 	// TODO(chuckha): We don't use the cfg for anything except passing a string around. Consider removing this struct.
// 	cfg := &plugin.WorkerConfig{}
// 	logrus.Infof("Loading plugin driver %v", dfn.Driver)
// 	switch dfn.Driver {
// 	case "DaemonSet":
// 		cfg.MasterURL = "http://" + masterAddress + "/api/v1/results/by-node"
// 		return daemonset.NewPlugin(namespace, dfn, cfg), nil
// 	case "Job":
// 		cfg.MasterURL = "http://" + masterAddress + "/api/v1/results/global"
// 		return job.NewPlugin(namespace, dfn, cfg), nil
// 	default:
// 		return nil, errors.Errorf("Unknown driver %v", dfn.Driver)
// 	}
// }

func findPlugins(dir string) ([]string, error) {
	return filepath.Glob(path.Join(dir, "*.yml"))
}

type sonobuoyConfig struct {
	Driver     string `json:"driver"`
	PluginName string `json:"plugin-name"`
	ResultType string `json:"result-type"`
}

type pluginDefinition struct {
	SonobuoyConfig sonobuoyConfig   `json:"sonobuoy-config"`
	Spec           corev1.Container `json:"spec"`
}

func loadDefinition(file string) (*pluginDefinition, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't open lugin definition %v", file)
	}

	// convert to JSON because corev1.Container only has JSON tags
	var decoded interface{}
	if err = yaml.Unmarshal(bytes, &decoded); err != nil {
		return nil, errors.Wrapf(err, "couldn't decode yaml for plugin definition %v", file)
	}

	decoded = convert(decoded)

	jsonBytes, err := json.Marshal(decoded)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't encode yaml as json for plugin definition %v", file)
	}

	var internalDef pluginDefinition
	if err = json.Unmarshal(jsonBytes, &internalDef); err != nil {
		return nil, errors.Wrapf(err, "couldn't decode json for plugin definition %v", file)
	}

	return &internalDef, nil
}

func loadPlugin(def *pluginDefinition, namespace string) (plugin.Interface, error) {
	pluginDef := plugin.Definition{
		Name:       def.SonobuoyConfig.PluginName,
		ResultType: def.SonobuoyConfig.ResultType,
		Spec:       def.Spec,
	}

	switch def.SonobuoyConfig.Driver {
	case "Job":
		return job.NewPlugin(pluginDef, namespace), nil
	case "DaemonSet":
		return daemonset.NewPlugin(pluginDef, namespace), nil
	default:
		return nil, fmt.Errorf("unknown driver %q for plugin %v",
			def.SonobuoyConfig.Driver, def.SonobuoyConfig.PluginName)
	}
}

func filterPluginDef(defs []*pluginDefinition, selections []plugin.Selection) []*pluginDefinition {
	m := make(map[string]bool)
	for _, selection := range selections {
		m[selection.Name] = true
	}

	filtered := []*pluginDefinition{}
	for _, def := range defs {
		if m[def.SonobuoyConfig.PluginName] {
			filtered = append(filtered, def)
		}
	}
	return filtered
}

// From https://stackoverflow.com/questions/40737122/convert-yaml-to-json-without-struct-golang
func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}
