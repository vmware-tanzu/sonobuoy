/*
Copyright the Sonobuoy contributors 2019

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
	"os"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	defaultPluginName    = "plugin"
	defaultPluginDriver  = "Job"
	defaultMountPath     = "/tmp/results"
	defaultMountName     = "results"
	defaultMountReadOnly = false
)

// GenPluginDefConfig are the input options for running
type GenPluginDefConfig struct {
	def manifest.Manifest

	// env holds the values that will be parsed into the manifest env vars.
	env EnvVars

	// driver holds the values that will be parsed into the plugin driver.
	// Allows validation during flag parsing by having a custom type.
	driver pluginDriver
}

// NewCmdGenPluginDef ...
func NewCmdGenPluginDef() *cobra.Command {
	genPluginOpts := GenPluginDefConfig{
		def:    defaultManifest(),
		env:    map[string]string{},
		driver: defaultPluginDriver,
	}

	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Generates the manifest Sonobuoy uses to define a plugin",
		Run:   genPluginDefWrapper(&genPluginOpts),
		Args:  cobra.NoArgs,
	}

	cmd.Flags().StringVarP(
		&genPluginOpts.def.SonobuoyConfig.PluginName, "name", "n", "",
		"Plugin name",
	)
	cmd.MarkFlagRequired("name")

	cmd.Flags().VarP(
		&genPluginOpts.driver, "type", "t",
		"Plugin Driver (job or daemonset)",
	)

	cmd.Flags().StringVarP(
		&genPluginOpts.def.Spec.Image, "image", "i", "",
		"Plugin image",
	)
	cmd.MarkFlagRequired("image")

	cmd.Flags().StringArrayVarP(
		&genPluginOpts.def.Spec.Command, "cmd", "c", []string{"./run.sh"},
		"Command to run when starting the plugin's container",
	)

	cmd.Flags().VarP(
		&genPluginOpts.env, "env", "e",
		"Env var values to set on the image (e.g. --env FOO=bar)",
	)

	return cmd
}

// defaultManifest returns the basic manifest the user's options will be placed
// on top of.
func defaultManifest() manifest.Manifest {
	m := manifest.Manifest{}
	m.Spec.Name = defaultPluginName
	m.SonobuoyConfig.Driver = defaultPluginDriver
	m.Spec.VolumeMounts = []v1.VolumeMount{
		v1.VolumeMount{
			MountPath: defaultMountPath,
			Name:      defaultMountName,
			ReadOnly:  defaultMountReadOnly,
		},
	}
	return m
}

// genPluginDefWrapper returns a closure around a given *GenPluginDefConfig that
// will adhere to the method signature needed by cobra. It prints the result of
// the action or logs an error and exits with non-zero status.
func genPluginDefWrapper(cfg *GenPluginDefConfig) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		s, err := genPluginDef(cfg)
		if err != nil {
			errlog.LogError(err)
			os.Exit(1)
		}
		fmt.Println(string(s))
	}
}

// genPluginDef returns the YAML for the plugin which Sonobuoy would expect as
// a configMap in order to run/gen a typical run.
func genPluginDef(cfg *GenPluginDefConfig) ([]byte, error) {
	// Result type just duplicates the name in most cases.
	cfg.def.SonobuoyConfig.ResultType = cfg.def.SonobuoyConfig.PluginName

	// Copy the validated value to the actual field.
	cfg.def.SonobuoyConfig.Driver = cfg.driver.String()

	// Add env vars to the container spec.
	cfg.def.Spec.Env = []v1.EnvVar{}
	for k, v := range cfg.env.Map() {
		cfg.def.Spec.Env = append(cfg.def.Spec.Env, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	yaml, err := kuberuntime.Encode(manifest.Encoder, &cfg.def)
	return yaml, errors.Wrap(err, "serializing as YAML")
}
