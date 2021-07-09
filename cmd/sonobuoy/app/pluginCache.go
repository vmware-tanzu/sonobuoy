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

	"github.com/mitchellh/go-homedir"
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

func NewCmdPlugin() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "plugin",
		Aliases: []string{"plugins"},
		Short:   "Manage your installed plugins",
		Hidden:  true,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all installed plugins",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listInstalledPlugins(getPluginCacheLocation(cmd))
		},
	}

	showCmd := &cobra.Command{
		Use:   "show <plugin filename>",
		Short: "Print the full definition of the named plugin file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showInstalledPlugin(getPluginCacheLocation(cmd), filenameFromArg(args[0], ".yaml"))
		},
	}

	installCmd := &cobra.Command{
		Use:   "install <save-as-filename> <source filename or URL>",
		Short: "Install a plugin so that it can be run via just its filename rather than a full path or URL.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return installPlugin(getPluginCacheLocation(cmd), filenameFromArg(args[0], ".yaml"), args[1])
		},
	}

	uninstallCmd := &cobra.Command{
		Use:   "uninstall <plugin filename>",
		Short: "Uninstall a plugin. You can continue to run any plugin via specifying a file or URL.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return deletePlugin(getPluginCacheLocation(cmd), filenameFromArg(args[0], ".yaml"))
		},
	}

	cmd.AddCommand(listCmd, showCmd, installCmd, uninstallCmd)

	return cmd
}

// getPluginCacheLocation will return the location of the plugin cache (defaults to ~/.sonobuoy)
// If no override is set in the env, the default is assumed. If we are unable to expand to an abosolute
// path then we return the empty string signalling the feature should be disabled.
func getPluginCacheLocation(cmd *cobra.Command) string {
	usePath := os.Getenv(SonobuoyDirEnvKey)
	if len(usePath) == 0 {
		usePath = defaultSonobuoyDir
	}
	expandedPath, err := homedir.Expand(usePath)
	if err != nil {
		logrus.Errorf("failed to expand sonobuoy directory %q: %v", usePath, err)
		return ""
	}

	if _, err := os.Stat(expandedPath); err != nil && os.IsNotExist(err) {
		logrus.Debugf("sonobuoy plugin location %q does not exist, creating it.", expandedPath)
		if err := os.Mkdir(expandedPath, 0777); err != nil {
			logrus.Errorf("failed to create directory for installed plugins %q: %v", expandedPath, err)
		}
	}

	logrus.Debugf("Using plugin cache location %q", expandedPath)
	return expandedPath
}

func listInstalledPlugins(installedDir string) error {
	pluginFiles, err := loadPlugins(installedDir)
	if err != nil {
		return errors.Wrap(err, "failed to load installed plugins")
	}

	prefix := ""
	first := true
	for filename, p := range pluginFiles {
		if !first {
			prefix = "---\n"
		}
		fmt.Printf("%vfilename: %v\nplugin name: %v\nsource URL: %v\ndescription: %v\n",
			prefix, filename, p.SonobuoyConfig.PluginName, p.SonobuoyConfig.SourceURL, p.SonobuoyConfig.Description)
		first = false
	}

	return nil
}

// loadPlugins loads all plugins from the given directory (recursively) and
// returns a map of their filename and contents. Returns the first error encountered
// but continues to load and return as many plugins as possible.
func loadPlugins(installedDir string) (map[string]*manifest.Manifest, error) {
	pluginMap := map[string]*manifest.Manifest{}
	var firstErr error

	err := filepath.Walk(installedDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".yaml" {
			m, err := loader.LoadDefinitionFromFile(path)
			if err != nil {
				fmt.Printf("failed to load path %q: %v\n", path, err)
				return errors.Wrapf(err, "failed to load definition from file %q", path)
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
	var reqManifest *manifest.Manifest
	err := filepath.Walk(installedDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if reqManifest != nil {
			return nil
		}

		if !info.IsDir() && path == filepath.Join(installedDir, reqFile) {
			m, err := loader.LoadDefinitionFromFile(path)
			if err != nil {
				fmt.Printf("failed to load path %q: %v\n", path, err)
				return errors.Wrapf(err, "failed to load definition from file %q", path)
			}
			reqManifest = m
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path %q: %v", installedDir, err)
	}

	if reqManifest != nil {
		return reqManifest, nil
	}
	return nil, fmt.Errorf("failed to find plugin file %v within directory %v", reqFile, installedDir)
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

func deletePlugin(installedDir, filename string) error {
	pluginPath := filepath.Join(installedDir, filename)

	_, err := loadPlugin(installedDir, filename)
	if err != nil {
		return errors.Wrap(err, "failed to load installed plugins")
	}

	if err := os.Remove(pluginPath); err != nil {
		return errors.Wrapf(err, "failed to delete file %v", pluginPath)
	}

	fmt.Printf("Deleted plugin file %v\n", pluginPath)
	return nil
}
