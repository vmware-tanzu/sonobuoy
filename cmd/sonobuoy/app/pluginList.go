/*
Copyright Sonobuoy Contributors 2019

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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

// pluginList represents a []manifest.Manifest objects describing plugins.
type pluginList struct {
	// StaticPlugins are plugins which do not depend on other values and can be
	// written to YAML as-is.
	StaticPlugins []*manifest.Manifest

	// DynamicPlugins are ones which require all the other gen input in order to finalize.
	// E.g. the e2e plugin was templated to use all those other values.
	DynamicPlugins []string
}

const (
	pluginE2E         = "e2e"
	pluginSystemdLogs = "systemd-logs"
	fileExtensionYAML = ".yaml"
)

// Make sure pluginList implements Value properly
var _ pflag.Value = &pluginList{}

// String needed for pflag.Value
func (p *pluginList) String() string {
	pluginNames := make(
		[]string,
		len(p.DynamicPlugins)+len(p.StaticPlugins),
		len(p.DynamicPlugins)+len(p.StaticPlugins),
	)
	for i := range p.StaticPlugins {
		pluginNames[i] = p.StaticPlugins[i].SonobuoyConfig.PluginName
	}
	pluginNames = append(pluginNames, p.DynamicPlugins...)
	return strings.Join(pluginNames, ",")
}

// Type needed for pflag.Value
func (p *pluginList) Type() string { return "pluginList" }

// Set sets the explicit path of the loader to the provided config file
func (p *pluginList) Set(str string) error {
	switch str {
	case pluginE2E:
		p.DynamicPlugins = append(p.DynamicPlugins, str)
	case pluginSystemdLogs:
		p.DynamicPlugins = append(p.DynamicPlugins, str)
	default:
		finfo, err := os.Stat(str)
		if err != nil {
			return errors.Wrapf(err, "unable to stat %q", str)
		}

		if finfo.IsDir() {
			return p.loadPluginsDir(str)
		}
		return p.loadSinglePlugin(str)
	}

	return nil
}

// loadPluginsDir loads every plugin in the given directory. It does not traverse recursively
// into the directory. A plugin must have the '.yaml' extension to be considered.
// It returns the first error encountered and stops processing.
func (p *pluginList) loadPluginsDir(dirpath string) error {
	files, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return errors.Wrapf(err, "failed to read directory %q", dirpath)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), fileExtensionYAML) {
			if err := p.loadSinglePlugin(filepath.Join(dirpath, file.Name())); err != nil {
				return errors.Wrapf(err, "failed to load plugin in file %q", file.Name())
			}
		}
	}

	return nil
}

// loadSinglePlugin loads a single plugin located at the given path.
func (p *pluginList) loadSinglePlugin(filepath string) error {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return errors.Wrapf(err, "unable to read file %q", filepath)
	}

	newPlugin, err := loadManifest(b)
	if err != nil {
		return errors.Wrapf(err, "failed to load plugin file %q", filepath)
	}

	p.StaticPlugins = append(p.StaticPlugins, newPlugin)
	return nil
}

func loadManifest(bytes []byte) (*manifest.Manifest, error) {
	var def manifest.Manifest
	err := kuberuntime.DecodeInto(manifest.Decoder, bytes, &def)
	return &def, errors.Wrap(err, "couldn't decode yaml for plugin definition")
}
