package client_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/plugin"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestGenerateManifest(t *testing.T) {
	tcs := []struct {
		name     string
		inputcm  *client.GenConfig
		expected *config.Config
	}{
		{
			name: "easy",
			inputcm: &client.GenConfig{
				E2EConfig: &client.E2EConfig{},
				Config:    &config.Config{},
			},
			expected: &config.Config{
				Aggregation: plugin.AggregationConfig{
					BindAddress: "0.0.0.0",
					BindPort:    8080,
				},
			},
		},
		{
			name: "override bind address",
			inputcm: &client.GenConfig{
				E2EConfig: &client.E2EConfig{},
				Config: &config.Config{
					Aggregation: plugin.AggregationConfig{
						BindAddress: "10.0.0.1",
					},
				},
			},
			expected: &config.Config{
				Aggregation: plugin.AggregationConfig{
					BindAddress: "10.0.0.1",
					BindPort:    8080,
				},
			},
		},
		{
			name: "https://github.com/heptio/sonobuoy/issues/390",
			inputcm: &client.GenConfig{
				E2EConfig: &client.E2EConfig{},
				Config: &config.Config{
					PluginSelections: []plugin.Selection{
						plugin.Selection{
							Name: "systemd-logs",
						},
					},
				},
			},
			expected: &config.Config{
				PluginSelections: []plugin.Selection{
					plugin.Selection{
						Name: "systemd-logs",
					},
				},
				Aggregation: plugin.AggregationConfig{
					BindAddress: "0.0.0.0",
					BindPort:    8080,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sbc, err := client.NewSonobuoyClient(nil)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := sbc.GenerateManifest(tc.inputcm)
			if err != nil {
				t.Fatal(err)
			}

			// TODO(chuckha) this is not my favorite thing.
			items := bytes.Split(manifest, []byte("---"))

			decoder := scheme.Codecs.UniversalDeserializer()
			for _, item := range items {
				o, gvk, err := decoder.Decode(item, nil, nil)
				if err != nil || gvk.Kind != "ConfigMap" {
					continue
				}

				cm, ok := o.(*v1.ConfigMap)
				if !ok {
					t.Fatal("was not a config map...")
				}
				// Ignore everything but the config map we're looking for
				if cm.ObjectMeta.Name != "sonobuoy-config-cm" {
					continue
				}

				configuration := &config.Config{}
				fmt.Println(cm.Data["config.json"])
				err = json.Unmarshal([]byte(cm.Data["config.json"]), configuration)
				if err != nil {
					t.Errorf("got error %v", err)
				}
				if configuration.UUID == "" {
					t.Error("Expected UUID to not be empty")
				}
				if configuration.Aggregation.BindAddress != tc.expected.Aggregation.BindAddress {
					t.Errorf("Expected %v but got %v", tc.expected.Aggregation.BindAddress, configuration.Aggregation.BindAddress)
				}
				if configuration.Aggregation.BindPort != tc.expected.Aggregation.BindPort {
					t.Errorf("Expected %v but got %v", tc.expected.Aggregation.BindPort, configuration.Aggregation.BindPort)
				}
				if len(configuration.PluginSelections) != len(tc.expected.PluginSelections) {
					t.Fatalf("Expected %v plugins but found %v", len(configuration.PluginSelections), len(tc.expected.PluginSelections))
				}
				for i, ps := range configuration.PluginSelections {
					if tc.expected.PluginSelections[i] != ps {
						t.Fatalf("Expected plugin %v but found plugin %v", tc.expected.PluginSelections[i], ps)
					}
				}
			}
		})
	}
}
