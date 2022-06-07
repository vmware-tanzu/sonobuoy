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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/features"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

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

	// InstallDir is the directory to check for plugins rather than only relying on files
	// in the present working directory or a URL. If blank, the list won't be able to load
	// plugins without specfying the whole path (or a relative one from the cwd).
	InstallDir string

	// initInstallDir is a flag to show us if we have looked up the plugin cache location yet.
	// Don't want to do this too early or else it happens before flags like `--level=trace` have been
	// parsed, leading to misleading logs.
	initInstallDir bool
}

const (
	pluginE2E         = "e2e"
	pluginSystemdLogs = "systemd-logs"
	fileExtensionYAML = ".yaml"

	renameAsSeperator = "@"
)

// Make sure pluginList implements Value properly
var _ pflag.Value = &pluginList{}

// String needed for pflag.Value
func (p *pluginList) String() string {
	pluginNames := make(
		[]string,
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
	// Parse as plugin@customName
	strSlice := strings.Split(str, renameAsSeperator)
	str = strSlice[0]
	renameAs := ""
	if len(strSlice) > 1 {
		renameAs = strSlice[1]
	}

	// Load first from cache, then special cases (e2e/systemd-logs), then local file.
	if p.GetInstallDir() != "" {
		handled, err := p.loadPluginsFromInstalled(str, renameAs)
		if handled {
			if err != nil {
				return errors.Wrapf(err, "unable to load plugin %v from installed plugins", str)
			}
			return nil
		}
	}

	switch str {
	case pluginE2E:
		p.DynamicPlugins = append(p.DynamicPlugins, str)
		if renameAs != "" {
			return fmt.Errorf("Cannot use @ renaming of plugins not loaded from file or URL")
		}
	case pluginSystemdLogs:
		p.DynamicPlugins = append(p.DynamicPlugins, str)
		if renameAs != "" {
			return fmt.Errorf("Cannot use @ renaming of plugins not loaded from file or URL")
		}
	default:
		if isURL(str) {
			return p.loadSinglePluginFromURL(str, renameAs)
		}
		return p.loadPluginsFromFilesystem(str, renameAs)
	}

	return nil
}

func isURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func (p *pluginList) loadPluginsFromInstalled(str, renameAs string) (handled bool, returnErr error) {
	// If empty, disable cache instead of err.
	if len(p.GetInstallDir()) == 0 {
		return false, nil
	}

	m, err := loadPlugin(p.InstallDir, filenameFromArg(str, ".yaml"))
	if isNotExist(err) {
		return false, err
	}
	if err != nil {
		return true, err
	}
	if len(renameAs) > 0 {
		m.SonobuoyConfig.PluginName = renameAs
	}
	p.StaticPlugins = append(p.StaticPlugins, m)
	return true, nil
}

func (p *pluginList) loadPluginsFromFilesystem(str, renameAs string) error {
	finfo, err := os.Stat(str)
	if err != nil {
		return errors.Wrapf(err, "unable to stat %q", str)
	}

	if finfo.IsDir() {
		if len(renameAs) > 0 {
			return fmt.Errorf("plugin renaming via @ is not valid if targeting a directory")
		}
		return p.loadPluginsDir(str)
	}
	return p.loadSinglePluginFromFile(str, renameAs)
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
			if err := p.loadSinglePluginFromFile(filepath.Join(dirpath, file.Name()), ""); err != nil {
				return errors.Wrapf(err, "failed to load plugin in file %q", file.Name())
			}
		}
	}

	return nil
}

// loadSinglePluginFromURL loads a single plugin located at the given path.
func (p *pluginList) loadSinglePluginFromURL(url, renameAs string) error {
	c := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := c.Get(url)
	if err != nil {
		return errors.Wrapf(err, "unable to GET URL %q", url)
	}
	if resp.StatusCode > 399 {
		return fmt.Errorf("unexpected HTTP response code %v", resp.StatusCode)
	}

	return errors.Wrapf(p.loadSinglePlugin(resp.Body, renameAs), "loading plugin from URL %q", url)
}

// loadSinglePluginFromFile loads a single plugin located at the given path.
func (p *pluginList) loadSinglePluginFromFile(filepath, renameAs string) error {
	f, err := os.Open(filepath)
	if err != nil {
		return errors.Wrapf(err, "unable to read file %q", filepath)
	}
	return errors.Wrapf(p.loadSinglePlugin(f, renameAs), "loading plugin from file %q", filepath)
}

// loadSinglePlugin reads the data from the reader and loads the plugin.
func (p *pluginList) loadSinglePlugin(r io.ReadCloser, renameAs string) error {
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "failed to read data for plugin")
	}

	newPlugin, err := loadManifest(b)
	if err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}

	if len(renameAs) > 0 {
		newPlugin.SonobuoyConfig.PluginName = renameAs
	}
	p.StaticPlugins = append(p.StaticPlugins, newPlugin)
	return nil
}

func loadManifest(bytes []byte) (*manifest.Manifest, error) {
	var def manifest.Manifest
	err := kuberuntime.DecodeInto(manifest.Decoder, bytes, &def)
	return &def, errors.Wrap(err, "couldn't decode yaml for plugin definition")
}

// isNotExist returns true if the cause of the error was that the plugin did not exist.
func isNotExist(e error) bool {
	_, ok := errors.Cause(e).(*pluginNotFoundError)
	return ok
}

// GetInstallDir should be used instead of referencing InstallDir directly. It allows
// for lazy lookup of the cache location.
func (p *pluginList) GetInstallDir() string {
	if p.initInstallDir {
		return p.InstallDir
	}

	if !features.Enabled(features.PluginInstallation) {
		p.InstallDir = ""
	} else {
		p.InstallDir = getPluginCacheLocation()
	}
	p.initInstallDir = true

	return p.InstallDir
}
