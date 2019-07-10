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
	"reflect"
	"strings"
	"testing"

	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/plugin"
)

const (
	rawInput = "not nil"
)

// TestResolveConformanceImage tests the temporary logic of ensuring that given
// a certain string version, the proper conformance image is used (upstream
// vs Heptio).
func TestResolveConformanceImage(t *testing.T) {
	tcs := []struct {
		name             string
		requestedVersion string
		expected         string
	}{
		{
			name:             "Comparison is lexical",
			requestedVersion: "foo",
			expected:         "gcr.io/heptio-images/kube-conformance",
		}, {
			name:             "Prior to v1.14.0 uses heptio and major.minor",
			requestedVersion: "v1.13.99",
			expected:         "gcr.io/heptio-images/kube-conformance",
		}, {
			name:             "v1.14.0 uses heptio and major.minor",
			requestedVersion: "v1.14.0",
			expected:         "gcr.io/heptio-images/kube-conformance",
		}, {
			name:             "v1.14.1 and after uses upstream and major.minor.patch",
			requestedVersion: "v1.14.1",
			expected:         "gcr.io/google-containers/conformance",
		}, {
			name:             "v1.14.0 and after uses upstream and major.minor.patch",
			requestedVersion: "v1.15.1",
			expected:         "gcr.io/google-containers/conformance",
		}, {
			name:             "latest should use upstream image",
			requestedVersion: "latest",
			expected:         "gcr.io/google-containers/conformance",
		}, {
			name:             "explicit version before v1.14.0 should use heptio image and given version",
			requestedVersion: "v1.12+.0.alpha+",
			expected:         "gcr.io/heptio-images/kube-conformance",
		}, {
			name:             "explicit version after v1.14.0 should use upstream and use given version",
			requestedVersion: "v1.14.1",
			expected:         "gcr.io/google-containers/conformance",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			out := resolveConformanceImage(tc.requestedVersion)
			if out != tc.expected {
				t.Errorf("Expected image %q but got %q", tc.expected, out)
			}
		})
	}
}

func TestResolveConfig(t *testing.T) {
	defaultPluginSearchPath := config.New().PluginSearchPath
	defaultAggr := plugin.AggregationConfig{TimeoutSeconds: 10800}

	tcs := []struct {
		name     string
		input    *genFlags
		expected *config.Config
		cliInput string
	}{
		{
			name: "Conformance mode when supplied config is nil (nothing interesting happens)",
			input: &genFlags{
				mode:           client.Conformance,
				sonobuoyConfig: SonobuoyConfig{},
			},
			expected: &config.Config{
				Namespace:       "heptio-sonobuoy",
				WorkerImage:     "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy: "IfNotPresent", // default
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				Resources:        config.DefaultResources,
			},
		}, {
			name: "Quick mode and a non-nil supplied config",
			input: &genFlags{
				mode: client.Quick,
				sonobuoyConfig: SonobuoyConfig{
					Config: config.Config{
						Aggregation: plugin.AggregationConfig{
							BindAddress: "10.0.0.1",
						},
					},
					raw: rawInput,
				},
			},
			expected: &config.Config{
				Namespace:       "heptio-sonobuoy",
				WorkerImage:     "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy: "IfNotPresent", // default
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
				},
				Aggregation: plugin.AggregationConfig{
					BindAddress:    "10.0.0.1",
					BindPort:       config.DefaultAggregationServerBindPort,
					TimeoutSeconds: 10800,
				},
				PluginSearchPath: defaultPluginSearchPath,
				Resources:        config.DefaultResources,
			},
		}, {
			name: "Conformance mode with plugin selection specified",
			input: &genFlags{
				mode: client.Conformance,
				sonobuoyConfig: SonobuoyConfig{
					Config: config.Config{
						PluginSelections: []plugin.Selection{
							{Name: "systemd-logs"},
						},
					},
					raw: "not empty",
				},
			},
			expected: &config.Config{
				Namespace:       "heptio-sonobuoy",
				WorkerImage:     "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version,
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
		}, {
			name: "Flags should override the config settings when set",
			input: &genFlags{
				sonobuoyConfig: SonobuoyConfig{
					Config: config.Config{Namespace: "configNS"},
				},
			},
			cliInput: "--namespace=flagNS --sonobuoy-image=flagImage --image-pull-policy=Always --timeout 100",
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
		}, {
			name:     "Flags shouldn't override the config settings unless set",
			input:    &genFlags{},
			cliInput: "--sonobuoy-image=flagImage --config testdata/sonobuoy.conf",
			expected: &config.Config{
				Namespace:       "configNS",
				WorkerImage:     "flagImage",
				ImagePullPolicy: "Never",
				PluginSelections: []plugin.Selection{
					{Name: "e2e"},
					{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      plugin.AggregationConfig{TimeoutSeconds: 500},
				Resources:        config.DefaultResources,
			},
		}, {
			name:     "Flags shouldn't override the config settings unless set",
			input:    &genFlags{},
			cliInput: "--config testdata/sonobuoy.conf",
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
		}, {
			name:     "Manually specified plugins should result in empty selection",
			input:    &genFlags{},
			cliInput: "--plugin e2e",
			expected: &config.Config{
				Namespace:        "heptio-sonobuoy",
				WorkerImage:      "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy:  "IfNotPresent",
				PluginSelections: nil,
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				Resources:        config.DefaultResources,
			},
		}, {
			name:     "Empty, non-nil resources and plugins should be preserved",
			input:    &genFlags{},
			cliInput: "--config testdata/emptyQueryAndPlugins.conf",
			expected: &config.Config{
				Namespace:        "heptio-sonobuoy",
				WorkerImage:      "gcr.io/heptio-images/sonobuoy:" + buildinfo.Version,
				ImagePullPolicy:  "IfNotPresent",
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation:      defaultAggr,
				PluginSelections: []plugin.Selection{},
				Resources:        []string{},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate parsing of input via CLI. Making this optional to avoid complicating
			// setup for other tests which just explicitly set the values.
			if len(tc.cliInput) > 0 {
				fs := GenFlagSet(tc.input, EnabledRBACMode)
				if err := fs.Parse(strings.Split(tc.cliInput, " ")); err != nil {
					t.Fatalf("Failed to parse CLI input %q: %v", tc.cliInput, err)
				}
			}

			conf := tc.input.resolveConfig()

			if conf.Namespace != tc.expected.Namespace {
				t.Errorf("Expected namespace %v but got %v", tc.expected.Namespace, conf.Namespace)
			}

			if conf.WorkerImage != tc.expected.WorkerImage {
				t.Errorf("Expected worker image %v but got %v", tc.expected.WorkerImage, conf.WorkerImage)
			}

			if conf.ImagePullPolicy != tc.expected.ImagePullPolicy {
				t.Errorf("Expected image pull policy %v but got %v", tc.expected.ImagePullPolicy, conf.ImagePullPolicy)
			}

			if conf.Aggregation.TimeoutSeconds != tc.expected.Aggregation.TimeoutSeconds {
				t.Errorf("Expected timeout %v but got %v", tc.expected.Aggregation.TimeoutSeconds, conf.Aggregation.TimeoutSeconds)
			}

			if !reflect.DeepEqual(conf.PluginSelections, tc.expected.PluginSelections) {
				t.Errorf("expected PluginSelections %v but got %v", tc.expected.PluginSelections, conf.PluginSelections)
			}

			if !reflect.DeepEqual(conf.Resources, tc.expected.Resources) {
				t.Errorf("expected resources %v but got %v", tc.expected.Resources, conf.Resources)
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
			raw: rawInput,
		},
	}

	testCases := []struct {
		name     string
		input    config.PodLogLimits
		expected config.PodLogLimits
	}{
		{
			name:     "Nil config will be overwritten by default value",
			input:    config.PodLogLimits {
				SonobuoyNamespace: nil,
			},
			expected: config.PodLogLimits{
				SonobuoyNamespace: defaultSonobuoyNamespace,
			},
		},
		{
			name:     "Non-nil config should be preserved",
			input:    config.PodLogLimits {
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
				*tc.expected.SonobuoyNamespace, *conf.Limits.PodLogs.SonobuoyNamespace )
		}
	}
}
