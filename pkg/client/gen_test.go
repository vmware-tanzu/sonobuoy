package client_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/plugin"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGenerateManifest(t *testing.T) {
	tcs := []struct {
		name     string
		inputcm  *client.GenConfig
		expected *config.Config
	}{
		{
			name: "Defaults in yield a default manifest.",
			inputcm: &client.GenConfig{
				E2EConfig: &client.E2EConfig{},
				Config:    &config.Config{},
			},
			expected: &config.Config{},
		},
		{
			name: "Overriding the bind address",
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
				},
			},
		},
		{
			name: "Overriding the plugin selection",
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
				Aggregation: plugin.AggregationConfig{},
			},
		},
		{
			name: "The plugin search path is not modified",
			inputcm: &client.GenConfig{
				E2EConfig: &client.E2EConfig{},
				Config: &config.Config{
					PluginSearchPath: []string{"a", "b", "c", "a"},
				},
			},
			expected: &config.Config{
				Aggregation:      plugin.AggregationConfig{},
				PluginSearchPath: []string{"a", "b", "c", "a"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sbc, err := client.NewSonobuoyClient(nil, nil)
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

				// TODO(chuckha) test other pieces of the generated yaml
				if cm.ObjectMeta.Name != "sonobuoy-config-cm" {
					continue
				}

				configuration := &config.Config{}
				err = json.Unmarshal([]byte(cm.Data["config.json"]), configuration)
				if err != nil {
					t.Errorf("got error %v", err)
				}
				if !reflect.DeepEqual(configuration, tc.expected) {
					t.Fatalf("Expected %v to equal %v", tc.expected, configuration)
				}
			}
		})
	}
}

func TestGenerateManifestSSH(t *testing.T) {
	tcs := []struct {
		name       string
		inputcm    *client.GenConfig
		goldenFile string
	}{
		{
			name: "Enabling SSH",
			inputcm: &client.GenConfig{
				E2EConfig:  &client.E2EConfig{},
				Config:     &config.Config{},
				SSHKeyPath: filepath.Join("testdata", "test_ssh.key"),
				SSHUser:    "ssh-user",
			},
			goldenFile: filepath.Join("testdata", "ssh.golden"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sbc, err := client.NewSonobuoyClient(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := sbc.GenerateManifest(tc.inputcm)
			if err != nil {
				t.Fatal(err)
			}

			if *update {
				ioutil.WriteFile(tc.goldenFile, manifest, 0666)
			} else {
				fileData, err := ioutil.ReadFile(tc.goldenFile)
				if err != nil {
					t.Fatalf("Failed to read golden file %v: %v", tc.goldenFile, err)
				}
				if !bytes.Equal(fileData, manifest) {
					t.Errorf("Expected manifest to equal goldenfile: %v but instead got: %v", tc.goldenFile, string(manifest))
				}
			}
		})
	}
}
