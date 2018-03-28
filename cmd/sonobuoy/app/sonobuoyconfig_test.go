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
	"testing"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestGetConfigWithMode(t *testing.T) {
	defaultPluginSearchPath := config.New().PluginSearchPath
	tcs := []struct {
		name     string
		mode     client.Mode
		inputcm  *SonobuoyConfig
		expected *config.Config
	}{
		{
			name:    "Conformance mode when supplied config is nil (nothing interesting happens)",
			mode:    client.Conformance,
			inputcm: &SonobuoyConfig{},
			expected: &config.Config{
				PluginSelections: []plugin.Selection{
					plugin.Selection{Name: "e2e"},
					plugin.Selection{Name: "systemd-logs"},
				},
				PluginSearchPath: defaultPluginSearchPath,
			},
		},
		{
			name: "Quick mode and a non-nil supplied config",
			mode: client.Quick,
			inputcm: &SonobuoyConfig{
				Config: config.Config{
					Aggregation: plugin.AggregationConfig{
						BindAddress: "10.0.0.1",
					},
				},
				// TODO(chuckha) consider exporting raw or not depending on it.
				raw: "not nil",
			},
			expected: &config.Config{
				PluginSelections: []plugin.Selection{
					plugin.Selection{Name: "e2e"},
				},
				Aggregation: plugin.AggregationConfig{
					BindAddress: "10.0.0.1",
					BindPort:    config.DefaultAggregationServerBindPort,
				},
				PluginSearchPath: defaultPluginSearchPath,
			},
		},
		{
			name: "Conformance mode with plugin selection specified",
			mode: client.Conformance,
			inputcm: &SonobuoyConfig{
				Config: config.Config{
					PluginSelections: []plugin.Selection{
						plugin.Selection{
							Name: "systemd-logs",
						},
					},
				},
				raw: "not empty",
			},
			expected: &config.Config{
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
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			conf := GetConfigWithMode(tc.inputcm, tc.mode)
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
					t.Fatalf("looking for plugin selection %v but did not find it", ps)
				}
			}
		})
	}
}
