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
	"path/filepath"

	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	manifesthelper "github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest/helper"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	defaultPluginName    = "plugin"
	defaultPluginDriver  = "Job"
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

	// nodeSelector is the node selectors to put into the podSpec for the plugin.
	// Separate field here since the default podSpec is nil but we also have to deal
	// with defaults. Easy to reconcile this way.
	nodeSelector map[string]string

	// If set, the default pod spec used by Sonobuoy will be included in the output
	showDefaultPodSpec bool

	// configMapFiles is the list of files to read/store as configmaps for the plugin.
	configMapFiles []string
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

	genPluginSet := pflag.NewFlagSet("generate plugin", pflag.ExitOnError)

	genPluginSet.StringVarP(
		&genPluginOpts.def.SonobuoyConfig.PluginName, "name", "n", "",
		"Plugin name",
	)

	genPluginSet.VarP(
		&genPluginOpts.driver, "type", "t",
		"Plugin Driver (job or daemonset)",
	)

	genPluginSet.StringVarP(
		&genPluginOpts.def.SonobuoyConfig.ResultFormat, "format", "f", results.ResultFormatRaw,
		"Result format (junit or raw)",
	)

	genPluginSet.StringToStringVar(
		&genPluginOpts.nodeSelector, "node-selector", nil,
		`Node selector for the plugin (key=value). Usually set to specify OS via kubernetes.io/os=windows. Can be set multiple times.`,
	)

	genPluginSet.StringVarP(
		&genPluginOpts.def.Spec.Image, "image", "i", "",
		"Plugin image",
	)

	genPluginSet.StringArrayVarP(
		&genPluginOpts.def.Spec.Command, "cmd", "c", []string{"./run.sh"},
		`Command to run when starting the plugin's container. Can be set multiple times (e.g. --cmd /bin/sh -c "-c")`,
	)

	genPluginSet.VarP(
		&genPluginOpts.env, "env", "e",
		"Env var values to set on the plugin (e.g. --env FOO=bar)",
	)

	genPluginSet.StringArrayVarP(
		&genPluginOpts.def.Spec.Args, "arg", "a", []string{},
		"Arg values to set on the plugin. Can be set multiple times (e.g. --arg 'arg 1' --arg arg2)",
	)

	genPluginSet.StringArrayVar(
		&genPluginOpts.configMapFiles, "configmap", nil,
		`Specifies files to read and add as configMaps. Will be mounted to the plugin at /tmp/sonobuoy/configs/<filename>.`,
	)

	AddShowDefaultPodSpecFlag(&genPluginOpts.showDefaultPodSpec, genPluginSet)

	cmd.Flags().AddFlagSet(genPluginSet)
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("image")
	return cmd
}

// defaultManifest returns the basic manifest the user's options will be placed
// on top of.
func defaultManifest() manifest.Manifest {
	m := manifest.Manifest{}
	m.Spec.Name = defaultPluginName
	m.SonobuoyConfig.Driver = defaultPluginDriver
	m.Spec.VolumeMounts = []v1.VolumeMount{
		{
			MountPath: plugin.ResultsDir,
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

	if cfg.showDefaultPodSpec {
		cfg.def.PodSpec = &manifest.PodSpec{
			PodSpec: driver.DefaultPodSpec(cfg.def.SonobuoyConfig.Driver),
		}
	}

	if cfg.nodeSelector != nil {
		// Initialize podSpec first
		if cfg.def.PodSpec == nil {
			cfg.def.PodSpec = &manifest.PodSpec{}
		}
		cfg.def.PodSpec.NodeSelector = cfg.nodeSelector
	}

	if len(cfg.configMapFiles) > 0 {
		cfg.def.ConfigMap = map[string]string{}
	}
	for _, v := range cfg.configMapFiles {
		fData, err := os.ReadFile(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file %q", v)
		}
		base := filepath.Base(v)
		cfg.def.ConfigMap[base] = string(fData)
	}

	yaml, err := kuberuntime.Encode(manifest.Encoder, &cfg.def)
	return yaml, errors.Wrap(err, "serializing as YAML")
}

func NewCmdGenE2E() *cobra.Command {
	var genE2Eflags genFlags
	configMapFiles := []string{}

	var cmd = &cobra.Command{
		Use:   "e2e",
		Short: "Generates the e2e plugin definition based on the given options",
		RunE:  genManifestForPlugin(&genE2Eflags, e2ePlugin),
		Args:  cobra.NoArgs,
	}
	cmd.Flags().AddFlagSet(GenFlagSet(&genE2Eflags, EnabledRBACMode))
	cmd.Flags().StringArrayVar(
		&configMapFiles, "configmap", nil,
		`Specifies files to read and add as configMaps. Will be mounted to the plugin at /tmp/sonobuoy/configs/<filename>.`,
	)
	return cmd
}

func genManifestForPlugin(genflags *genFlags, pluginName string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := genflags.Config()
		if err != nil {
			return err
		}

		// Generate does not require any client configuration
		sbc := &client.SonobuoyClient{}

		_, plugins, err := sbc.GenerateManifestAndPlugins(cfg)
		if err != nil {
			return errors.Wrap(err, "error attempting to generate sonobuoy manifest")
		}

		for _, p := range plugins {
			if p.Spec.Name == pluginName {
				yaml, err := manifesthelper.ToYAML(p, cfg.ShowDefaultPodSpec)
				if err != nil {
					return errors.Wrap(err, "error attempting to serialize plugin")
				}
				fmt.Print(string(yaml))
			}
		}
		return nil
	}
}

func NewCmdGenSystemdLogs() *cobra.Command {
	var genSystemdLogsflags genFlags
	var cmd = &cobra.Command{
		Use:   "systemd-logs",
		Short: "Generates the systemd-logs plugin definition based on the given options",
		RunE:  genManifestForPlugin(&genSystemdLogsflags, systemdLogsPlugin),
		Args:  cobra.NoArgs,
	}
	cmd.Flags().AddFlagSet(GenFlagSet(&genSystemdLogsflags, EnabledRBACMode))
	return cmd
}
