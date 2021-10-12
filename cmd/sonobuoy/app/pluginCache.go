/*
Copyright the Sonobuoy contributors 2021

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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/loader"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	SonobuoyDirEnvKey = "SONOBUOY_DIR"
)

var (
	defaultSonobuoyDir = filepath.Join("~", ".sonobuoy")
)

// pluginNotFoundError is a custom type so we can tell whether or not loading the plugin
// from the installation directory FAILED or if it just wasn't found.
type pluginNotFoundError struct{ e error }

func (p *pluginNotFoundError) Error() string {
	return p.e.Error()
}

func NewCmdPlugin() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugin",
		Aliases: []string{"plugins"},
		Short:   "Manage your installed plugins",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listInstalledPlugins(getPluginCacheLocation())
		},
	}

	showCmd := &cobra.Command{
		Use:   "show <plugin filename>",
		Short: "Print the full definition of the named plugin file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showInstalledPlugin(getPluginCacheLocation(), filenameFromArg(args[0], ".yaml"))
		},
	}

	installCmd := &cobra.Command{
		Use:   "install <save-as-filename> <source filename or URL>",
		Short: "Install a plugin so that it can be run via just its filename rather than a full path or URL.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installPlugin(getPluginCacheLocation(), filenameFromArg(args[0], ".yaml"), args[1])
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall <plugin filename>",
		Short: "Uninstall a plugin. You can continue to run any plugin via specifying a file or URL.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return uninstallPlugin(getPluginCacheLocation(), filenameFromArg(args[0], ".yaml"))
		},
	}

	cmd.AddCommand(listCmd, showCmd, installCmd, uninstallCmd)

	return cmd
}

// getPluginCacheLocation will return the location of the plugin cache (defaults to ~/.sonobuoy)
// If no override is set in the env, the default is assumed. If we are unable to expand to an abosolute
// path then we return the empty string signalling the feature should be disabled.
func getPluginCacheLocation() string {
	usePath := os.Getenv(SonobuoyDirEnvKey)
	if len(usePath) == 0 {
		usePath = defaultSonobuoyDir
	}
	expandedPath, err := expandPath(usePath)
	if err != nil {
		logrus.Errorf("failed to expand sonobuoy directory %q: %v", usePath, err)
		return ""
	}

	if _, err := os.Stat(expandedPath); err != nil && os.IsNotExist(err) {
		logrus.Debugf("sonobuoy plugin location %q does not exist, creating it.", expandedPath)
		if err := os.Mkdir(expandedPath, 0777); err != nil {
			logrus.Errorf("failed to create directory for installed plugins %q: %v", expandedPath, err)
			return ""
		}
	}

	logrus.Debugf("Using plugin cache location %q", expandedPath)
	return expandedPath
}

func listInstalledPlugins(installedDir string) error {
	if len(installedDir) == 0 {
		return errors.New("unable to list plugins; installation directory unavailable")
	}

	pluginFiles, _ := loadPlugins(installedDir)
	filenames := []string{}
	for filename := range pluginFiles {
		filenames = append(filenames, filename)
	}
	sort.StringSlice(filenames).Sort()

	prefix := ""
	first := true
	for _, filename := range filenames {
		p := pluginFiles[filename]
		if !first {
			prefix = "---\n"
		}
		fmt.Printf("%vRun as: %v\nFilename: %v\nPlugin name (in aggregator): %v\nSource URL: %v\nDescription: %v\n",
			prefix, strings.TrimSuffix(filepath.Base(filename), ".yaml"), filename, p.SonobuoyConfig.PluginName, p.SonobuoyConfig.SourceURL, p.SonobuoyConfig.Description)
		first = false
	}

	return nil
}

// loadPlugins loads all plugins from the given directory (recursively) and
// returns a map of their filename and contents. Returns the first error encountered
// but continues to load and return as many plugins as possible. All errors are logged.
// The resulting map, therefore, can be used even if an error is returned.
func loadPlugins(installedDir string) (map[string]*manifest.Manifest, error) {
	pluginMap := map[string]*manifest.Manifest{}
	var firstErr error

	err := filepath.Walk(installedDir, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			m, err := loader.LoadDefinitionFromFile(path)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				logrus.Error(errors.Wrapf(err, "failed to load definition from file %q", path))
				return nil
			}
			pluginMap[path] = m
		}
		return nil
	})
	if err != nil {
		if firstErr == nil {
			firstErr = err
		}
		logrus.Errorf("error walking the path %q: %v\n", installedDir, err)
	}

	return pluginMap, firstErr
}

func loadPlugin(installedDir, reqFile string) (*manifest.Manifest, error) {
	// Ignore error since it may not have to do with this plugin at all. Only return error if not found.
	// The loadPlugins will log errors as needed for visibility anyways.
	manMap, _ := loadPlugins(installedDir)
	reqManifest := manMap[filepath.Join(installedDir, reqFile)]
	if reqManifest != nil {
		return reqManifest, nil
	}
	return nil, &pluginNotFoundError{fmt.Errorf("failed to find plugin file %v within directory %v", reqFile, installedDir)}
}

// filenameFromArg will ensure the arg has the extension requested. This allows
// users to provide `-p foo` while loading the plugin file `foo.yaml`.
func filenameFromArg(arg, extension string) string {
	if filepath.Ext(arg) != extension {
		return fmt.Sprintf("%v%v", arg, extension)
	}
	return arg
}

// showInstalledPlugin returns the YAML of the plugin specified in the given file relative
// to the given installation directory.
func showInstalledPlugin(installedDir, reqPluginFile string) error {
	if len(installedDir) == 0 {
		return errors.New("unable to show plugin; installation directory unavailable")
	}

	plugin, err := loadPlugin(installedDir, reqPluginFile)
	if err != nil {
		return errors.Wrap(err, "failed to load installed plugins")
	}

	yaml, err := kuberuntime.Encode(manifest.Encoder, plugin)
	if err != nil {
		return errors.Wrap(err, "serializing as YAML")
	}
	fmt.Println(string(yaml))
	return nil
}

// installPlugin will read the plugin at src (URL or file) then install it into the
// installation directory with the given filename. If too many or too few plugins
// are loaded, errors are returned. The returned string is a human-readable description
// of the action taken.
func installPlugin(installedDir, filename, src string) error {
	if len(installedDir) == 0 {
		return errors.New("unable to install plugins; installation directory unavailable")
	}

	newPath := filepath.Join(installedDir, filename)
	var pl pluginList
	if err := pl.Set(src); err != nil {
		return err
	}

	if len(pl.StaticPlugins) > 1 {
		return fmt.Errorf("may only install one plugin at a time, found %v", len(pl.StaticPlugins))
	}
	if len(pl.StaticPlugins) < 1 {
		return fmt.Errorf("expected 1 plugin, found %v", len(pl.StaticPlugins))
	}

	yaml, err := kuberuntime.Encode(manifest.Encoder, pl.StaticPlugins[0])
	if err != nil {
		return errors.Wrap(err, "failed to encode plugin")
	}
	if err := os.WriteFile(newPath, yaml, 0666); err != nil {
		return err
	}
	fmt.Printf("Installed plugin %v into file %v from source %v\n", pl.StaticPlugins[0].Spec.Name, newPath, src)
	return nil
}

func uninstallPlugin(installedDir, filename string) error {
	if len(installedDir) == 0 {
		return errors.New("unable to uninstall plugins; installation directory unavailable")
	}

	pluginPath := filepath.Join(installedDir, filename)

	_, err := loadPlugin(installedDir, filename)
	if err != nil {
		return errors.Wrap(err, "failed to load installed plugins")
	}

	if err := os.Remove(pluginPath); err != nil {
		return errors.Wrapf(err, "failed to uninstall plugin file %v", pluginPath)
	}

	fmt.Printf("Uninstalled plugin file %v\n", pluginPath)
	return nil
}

func expandPath(path string) (string, error) {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return path, nil
}
