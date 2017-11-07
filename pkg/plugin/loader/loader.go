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
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/job"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// LoadAllPlugins loads all plugins by finding plugin definitions in the given
// directory, taking a user's plugin selections, and a sonobuoy phone home
// address (host:port) and returning all of the active, configured plugins for
// this sonobuoy run.
func LoadAllPlugins(namespace string, searchPath []string, selections []plugin.Selection, masterAddress string) (ret []plugin.Interface, err error) {
	var defns []plugin.Definition

	for _, dir := range searchPath {
		wd, _ := os.Getwd()
		logrus.Infof("Scanning plugins in %v (pwd: %v)", dir, wd)

		// We only care about configured plugin directories that exist,
		// since we may have a broad search path.
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Infof("Directory (%v) does not exist", dir)
			continue
		}

		definitions, err := scanPlugins(dir)
		if err != nil {
			return ret, err
		}

		defns = append(defns, definitions...)
	}

	for _, selection := range selections {
		for _, pluginDef := range defns {
			if selection.Name == pluginDef.Name {
				p, err := loadPlugin(namespace, pluginDef, masterAddress)
				if err != nil {
					return ret, err
				}
				ret = append(ret, p)
			}
		}
	}
	return ret, nil
}

// loadPlugin loads an individual plugin by instantiating a plugin driver with
// the settings from the given plugin definition and selection
func loadPlugin(namespace string, dfn plugin.Definition, masterAddress string) (plugin.Interface, error) {
	// TODO(chuckha): We don't use the cfg for anything except passing a string around. Consider removing this struct.
	cfg := &plugin.WorkerConfig{}
	logrus.Infof("Loading plugin driver %v", dfn.Driver)
	switch dfn.Driver {
	case "DaemonSet":
		cfg.MasterURL = "http://" + masterAddress + "/api/v1/results/by-node"
		return daemonset.NewPlugin(namespace, dfn, cfg), nil
	case "Job":
		cfg.MasterURL = "http://" + masterAddress + "/api/v1/results/global"
		return job.NewPlugin(namespace, dfn, cfg), nil
	default:
		return nil, errors.Errorf("Unknown driver %v", dfn.Driver)
	}
}

// scanPlugins looks for Plugin Definition files in the given directory,
// and returns an array of PluginDefinition structs.
func scanPlugins(dir string) ([]plugin.Definition, error) {
	var pluginDfns []plugin.Definition

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read plugin directory")
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".tmpl" {
			logrus.WithField("filename", file.Name()).Info("unknown template type")
			continue
		}

		// Read the template file into memory
		fullPath := path.Join(dir, file.Name())
		pluginTemplate, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		dfn, err := loadTemplate(pluginTemplate)
		if err != nil {
			logrus.WithError(err).WithField("filename", file.Name()).Info("failed to load plugin")
			continue
		}
		pluginDfns = append(pluginDfns, *dfn)
	}

	return pluginDfns, err
}

func loadTemplate(tmpl []byte) (*plugin.Definition, error) {
	t, err := template.New("plugin").Parse(string(tmpl))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse template")
	}
	var b bytes.Buffer
	// We just trying to get a kubernetes object here we don't really care about values rn
	err = t.Execute(&b, &plugin.DefinitionTemplateData{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}
	var x unstructured.Unstructured
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), b.Bytes(), &x); err != nil {
		return nil, errors.Wrap(err, "failed to turn executed template into an unstructured")
	}
	return &plugin.Definition{
		Driver:     x.GetAnnotations()["sonobuoy-driver"],
		Name:       x.GetAnnotations()["sonobuoy-plugin"],
		ResultType: x.GetAnnotations()["sonobuoy-result-type"],
		Template:   t,
	}, nil
}
