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
	"strings"
	"testing"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestGetConfig(t *testing.T) {
	defaultPluginSearchPath := config.New().PluginSearchPath

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
				WorkerImage:     "gcr.io/heptio-images/sonobuoy:latest",
				ImagePullPolicy: "Always", // default
				PluginSelections: []plugin.Selection{
					plugin.Selection{Name: "e2e"},
					plugin.Selection{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
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
					// TODO(chuckha) consider exporting raw or not depending on it.
					raw: "not nil",
				},
			},
			expected: &config.Config{
				Namespace:       "heptio-sonobuoy",
				WorkerImage:     "gcr.io/heptio-images/sonobuoy:latest",
				ImagePullPolicy: "Always", // default
				PluginSelections: []plugin.Selection{
					plugin.Selection{Name: "e2e"},
				},
				Aggregation: plugin.AggregationConfig{
					BindAddress: "10.0.0.1",
					BindPort:    config.DefaultAggregationServerBindPort,
				},
				PluginSearchPath: defaultPluginSearchPath,
			},
		}, {
			name: "Conformance mode with plugin selection specified",
			input: &genFlags{
				mode: client.Conformance,
				sonobuoyConfig: SonobuoyConfig{
					Config: config.Config{
						PluginSelections: []plugin.Selection{
							plugin.Selection{
								Name: "systemd-logs",
							},
						},
					},
					raw: "not empty",
				},
			},
			expected: &config.Config{
				Namespace:       "heptio-sonobuoy",
				WorkerImage:     "gcr.io/heptio-images/sonobuoy:latest",
				ImagePullPolicy: "Always", // default
				PluginSelections: []plugin.Selection{
					plugin.Selection{
						Name: "systemd-logs",
					},
				},
				PluginSearchPath: defaultPluginSearchPath,
				Aggregation: plugin.AggregationConfig{
					BindAddress: config.DefaultAggregationServerBindAddress,
					BindPort:    config.DefaultAggregationServerBindPort,
				},
			},
		}, {
			name: "Flags should override the config settings when set",
			input: &genFlags{
				sonobuoyConfig: SonobuoyConfig{
					Config: config.Config{Namespace: "configNS"},
				},
			},
			cliInput: "--namespace=flagNS --sonobuoy-image=flagImage --image-pull-policy=IfNotPresent",
			expected: &config.Config{
				Namespace:       "flagNS",
				WorkerImage:     "flagImage",
				ImagePullPolicy: "IfNotPresent",
				PluginSelections: []plugin.Selection{
					plugin.Selection{Name: "e2e"},
					plugin.Selection{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
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
					plugin.Selection{Name: "e2e"},
					plugin.Selection{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
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
					plugin.Selection{Name: "e2e"},
					plugin.Selection{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate parsing of input via CLI. Making this optional to avoid complicating
			// setup for other tests which just explicitly set the values.
			if len(tc.cliInput) > 0 {
				fs := GenFlagSet(tc.input, EnabledRBACMode, ConformanceImageVersionLatest)
				if err := fs.Parse(strings.Split(tc.cliInput, " ")); err != nil {
					t.Fatalf("Failed to parse CLI input %q: %v", tc.cliInput, err)
				}
			}

			conf := tc.input.getConfig()

			if conf.Namespace != tc.expected.Namespace {
				t.Errorf("Expected namespace %v but got %v", tc.expected.Namespace, conf.Namespace)
			}

			if conf.WorkerImage != tc.expected.WorkerImage {
				t.Errorf("Expected worker image %v but got %v", tc.expected.WorkerImage, conf.WorkerImage)
			}

			if conf.ImagePullPolicy != tc.expected.ImagePullPolicy {
				t.Errorf("Expected image pull policy %v but got %v", tc.expected.ImagePullPolicy, conf.ImagePullPolicy)
			}

			if len(conf.PluginSelections) != len(tc.expected.PluginSelections) {
				t.Fatalf("expected %v plugin selections but found %v", tc.expected.PluginSelections, conf.PluginSelections)
			}
			for _, ps := range conf.PluginSelections {
				found := false
				for _, expectedPs := range tc.expected.PluginSelections {
					if ps.Name == expectedPs.Name {
						found = true
					}
				}
				if !found {
					t.Errorf("looking for plugin selection %v but did not find it", ps)
				}
			}
		})
	}
}
