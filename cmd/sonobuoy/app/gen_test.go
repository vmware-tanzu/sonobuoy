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

package app

import (
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/buildinfo"
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	"github.com/kylelemons/godebug/pretty"
	v1 "k8s.io/api/core/v1"
)

const (
	rawInput = "not nil"
)

func TestResolveConfig(t *testing.T) {
	defaultPluginSearchPath := config.New().PluginSearchPath
	defaultAggr := plugin.AggregationConfig{TimeoutSeconds: 21600}
	dynamicConfigFileName := "*determinedAtRuntime*"

	tcs := []struct {
		name string

		// CLI input to parse.
		input string

		// Not every field of this will be tested at this time.
		expected *config.Config

		// If specified, will write the config to a temp file then append
		// the `--config=tmpfile` to the input. This way we can keep the config files
		// small and with the test rather than in a testdata file.
		configFileContents string

		// TODO(jschnake): This test previously was just testing the config.Config
		// and only certain fields. We may consider expanding this to testing the entire
		// genConfig object but for now I am expanding this just to ensure proper plugin loading.
		expectedGenConfigPlugins *client.GenConfig
	}{
		{
			name:  "NonDisruptiveConformance mode when supplied config is nil (nothing interesting happens)",
			input: "",
			expected: &config.Config{
				Namespace:       "sonobuoy",
				WorkerImage:     "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy: "IfNotPresent", // default
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"e2e", "systemd-logs"},
			},
		}, {
			name:               "Quick mode and a non-nil supplied config",
			input:              "--mode quick --config=" + dynamicConfigFileName,
			configFileContents: `{"Server":{"bindaddress":"10.0.0.1"}}`,
			expected: &config.Config{
				Namespace:       "sonobuoy",
				WorkerImage:     "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy: "IfNotPresent", // default
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
				},
				Aggregation: plugin.AggregationConfig{
					BindAddress:    "10.0.0.1",
					BindPort:       config.DefaultAggregationServerBindPort,
					TimeoutSeconds: 21600,
				},
				PluginSearchPath: defaultPluginSearchPath,
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
			},
		}, {
			name:               "NonDisruptiveConformance mode with plugin selection specified",
			input:              "--plugin systemd-logs --config=" + dynamicConfigFileName,
			configFileContents: `{"Plugins":[{"name":"systemd-logs"}]}`,
			expected: &config.Config{
				Namespace:       "sonobuoy",
				WorkerImage:     "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy: "IfNotPresent", // default
				PluginSelections: []plugin.Selection{
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation: plugin.AggregationConfig{
					BindAddress:    config.DefaultAggregationServerBindAddress,
					BindPort:       config.DefaultAggregationServerBindPort,
					TimeoutSeconds: config.DefaultAggregationServerTimeoutSeconds,
				},
				Resources: config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"systemd-logs"},
			},
		}, {
			name:               "NS flag prioritized over config value",
			configFileContents: `{"Namespace":"configNS","WorkerImage":"configImage","ImagePullPolicy":"IfNotPresent","Server":{"timeoutseconds":999}}`,
			input:              "--namespace=flagNS --sonobuoy-image=flagImage --image-pull-policy=Always --timeout 100 --config=" + dynamicConfigFileName,
			expected: &config.Config{
				Namespace:       "flagNS",
				WorkerImage:     "flagImage",
				ImagePullPolicy: "Always",
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      plugin.AggregationConfig{TimeoutSeconds: 100},
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"e2e", "systemd-logs"},
			},
		}, {
			name:               "Worker image and pull policy flags prioritized over config values",
			input:              "--sonobuoy-image=flagImage --image-pull-policy=Always --config " + dynamicConfigFileName,
			configFileContents: `{"Namespace":"configNS","WorkerImage":"configImage","ImagePullPolicy":"Never","Server":{"timeoutseconds":500}}`,
			expected: &config.Config{
				Namespace:       "configNS",
				WorkerImage:     "flagImage",
				ImagePullPolicy: "Always",
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      plugin.AggregationConfig{TimeoutSeconds: 500},
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"e2e", "systemd-logs"},
			},
		}, {
			name:               "Default flag values dont override config values",
			input:              "--config " + dynamicConfigFileName,
			configFileContents: `{"Namespace":"configNS","WorkerImage":"configImage","ImagePullPolicy":"Never","Server":{"timeoutseconds":500}}`,
			expected: &config.Config{
				Namespace:       "configNS",
				WorkerImage:     "configImage",
				ImagePullPolicy: "Never",
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      plugin.AggregationConfig{TimeoutSeconds: 500},
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"e2e", "systemd-logs"},
			},
		}, {
			name:  "Manually specified plugins should result in empty selection",
			input: "--plugin e2e",
			expected: &config.Config{
				Namespace:        "sonobuoy",
				WorkerImage:      "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy:  "IfNotPresent",
				PluginSelections: nil,
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
			},
		}, {
			name:  "Empty, non-nil resources and plugins should be preserved",
			input: "--config testdata/emptyQueryAndPlugins.conf",
			expected: &config.Config{
				Namespace:        "sonobuoy",
				WorkerImage:      "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy:  "IfNotPresent",
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				PluginSelections: []plugin.Selection{},
				Resources:        []string{},
			},
			expectedGenConfigPlugins: &client.GenConfig{},
		}, {
			name:  "Plugins can be loaded by directory",
			input: "--plugin testdata/testPluginDir",
			expected: &config.Config{
				Namespace:        "sonobuoy",
				WorkerImage:      "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy:  "IfNotPresent",
				PluginSelections: nil,
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				StaticPlugins: []*manifest.Manifest{
					{SonobuoyConfig: manifest.SonobuoyConfig{Driver: "Job", PluginName: "plugin1", SkipCleanup: false, ResultFormat: "raw"}, Spec: manifest.Container{Container: v1.Container{Name: "plugin", Image: "foo/bar:v1", Command: []string{"./run.sh"}, VolumeMounts: []v1.VolumeMount{{ReadOnly: false, Name: "results", MountPath: plugin.ResultsDir}}}}},
					{SonobuoyConfig: manifest.SonobuoyConfig{Driver: "Job", PluginName: "plugin2", SkipCleanup: false, ResultFormat: "raw"}, Spec: manifest.Container{Container: v1.Container{Name: "plugin", Image: "foo/bar:v2", Command: []string{"./run.sh"}, VolumeMounts: []v1.VolumeMount{{ReadOnly: false, Name: "results", MountPath: plugin.ResultsDir}}}}},
				},
			},
		}, {
			name:  "Plugins loaded by directory and by name",
			input: "--plugin e2e --plugin testdata/testPluginDir --plugin testdata/testPluginDir/pluginNotYAML.ext",
			expected: &config.Config{
				Namespace:        "sonobuoy",
				WorkerImage:      "sonobuoy/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy:  "IfNotPresent",
				PluginSelections: nil,
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				Resources:        config.DefaultResources,
			},
			expectedGenConfigPlugins: &client.GenConfig{
				StaticPlugins: []*manifest.Manifest{
					{SonobuoyConfig: manifest.SonobuoyConfig{Driver: "Job", PluginName: "plugin1", SkipCleanup: false, ResultFormat: "raw"}, Spec: manifest.Container{Container: v1.Container{Name: "plugin", Image: "foo/bar:v1", Command: []string{"./run.sh"}, VolumeMounts: []v1.VolumeMount{{ReadOnly: false, Name: "results", MountPath: plugin.ResultsDir}}}}},
					{SonobuoyConfig: manifest.SonobuoyConfig{Driver: "Job", PluginName: "plugin2", SkipCleanup: false, ResultFormat: "raw"}, Spec: manifest.Container{Container: v1.Container{Name: "plugin", Image: "foo/bar:v2", Command: []string{"./run.sh"}, VolumeMounts: []v1.VolumeMount{{ReadOnly: false, Name: "results", MountPath: plugin.ResultsDir}}}}},
					{SonobuoyConfig: manifest.SonobuoyConfig{Driver: "Job", PluginName: "plugin3", SkipCleanup: false, ResultFormat: "raw"}, Spec: manifest.Container{Container: v1.Container{Name: "plugin", Image: "foo/bar:v3", Command: []string{"./run.sh"}, VolumeMounts: []v1.VolumeMount{{ReadOnly: false, Name: "results", MountPath: plugin.ResultsDir}}}}},
				},
				DynamicPlugins: []string{"e2e"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Write and load a config file
			if len(tc.configFileContents) > 0 {
				tmpFile, err := ioutil.TempFile("", "")
				if err != nil {
					t.Fatalf("Failed to create temp file for test: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				err = ioutil.WriteFile(tmpFile.Name(), []byte(tc.configFileContents), 0666)
				if err != nil {
					t.Fatalf("Failed to write temp config file for test: %v", err)
				}
				tc.input = strings.Replace(tc.input, dynamicConfigFileName, tmpFile.Name(), -1)
			}

			// Simulate parsing of input via CLI.
			gflagset := &genFlags{}
			fs := GenFlagSet(gflagset, EnabledRBACMode)
			if err := fs.Parse(strings.Split(tc.input, " ")); err != nil {
				t.Fatalf("Failed to parse CLI input %q: %v", tc.input, err)
			}

			// Manually set KubeConformanceImage for all tests so that it does not error due to having 'auto' image
			// without a real cluster to target for version info
			gflagset.kubeConformanceImage = "testOnly"

			genConfig, err := gflagset.Config()
			if err != nil {
				t.Fatalf("Failed to generate GenConfig: %v", err)
			}

			if genConfig.Config.Namespace != tc.expected.Namespace {
				t.Errorf("Expected namespace %v but got %v", tc.expected.Namespace, genConfig.Config.Namespace)
			}

			if genConfig.Config.WorkerImage != tc.expected.WorkerImage {
				t.Errorf("Expected worker image %v but got %v", tc.expected.WorkerImage, genConfig.Config.WorkerImage)
			}

			if genConfig.Config.ImagePullPolicy != tc.expected.ImagePullPolicy {
				t.Errorf("Expected image pull policy %v but got %v", tc.expected.ImagePullPolicy, genConfig.Config.ImagePullPolicy)
			}

			if genConfig.Config.Aggregation.TimeoutSeconds != tc.expected.Aggregation.TimeoutSeconds {
				t.Errorf("Expected timeout %v but got %v", tc.expected.Aggregation.TimeoutSeconds, genConfig.Config.Aggregation.TimeoutSeconds)
			}

			if !reflect.DeepEqual(genConfig.Config.PluginSelections, tc.expected.PluginSelections) {
				t.Errorf("expected PluginSelections %v but got %v", tc.expected.PluginSelections, genConfig.Config.PluginSelections)
			}

			if !reflect.DeepEqual(genConfig.Config.Resources, tc.expected.Resources) {
				t.Errorf("expected resources %v but got %v", tc.expected.Resources, genConfig.Config.Resources)
			}

			if diff := pretty.Compare(genConfig.StaticPlugins, tc.expectedGenConfigPlugins.StaticPlugins); diff != "" {
				t.Errorf("expected static plugins to match but got diff:\n%s\n", diff)
			}

			if diff := pretty.Compare(genConfig.DynamicPlugins, tc.expectedGenConfigPlugins.DynamicPlugins); diff != "" {
				t.Errorf("expected dynamic plugins to match but got diff:\n%s\n", diff)
			}
		})
	}
}

func TestResolveConfigPodLogLimits(t *testing.T) {
	defaultSonobuoyNamespace := new(bool)
	*defaultSonobuoyNamespace = true

	g := &genFlags{
		sonobuoyConfig: SonobuoyConfig{
			Config: config.Config{},
			raw:    rawInput,
		},
	}

	testCases := []struct {
		name     string
		input    config.PodLogLimits
		expected config.PodLogLimits
	}{
		{
			name: "Nil config will be overwritten by default value",
			input: config.PodLogLimits{
				SonobuoyNamespace: nil,
			},
			expected: config.PodLogLimits{
				SonobuoyNamespace: defaultSonobuoyNamespace,
			},
		},
		{
			name: "Non-nil config should be preserved",
			input: config.PodLogLimits{
				SonobuoyNamespace: &[]bool{false}[0],
			},
			expected: config.PodLogLimits{
				SonobuoyNamespace: &[]bool{false}[0],
			},
		},
	}

	for _, tc := range testCases {
		g.sonobuoyConfig.Limits.PodLogs = tc.input
		conf := g.resolveConfig()

		if *conf.Limits.PodLogs.SonobuoyNamespace != *tc.expected.SonobuoyNamespace {
			t.Errorf("Expected Limits.PodLogs.SonobuoyNamespace %v but got %v",
				*tc.expected.SonobuoyNamespace, *conf.Limits.PodLogs.SonobuoyNamespace)
		}
	}
}
