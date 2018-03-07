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

package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestSaveAndLoad(t *testing.T) {
	cfg := New()

	cfg.Filters.Namespaces = "funky*"

	if blob, err := json.Marshal(&cfg); err == nil {
		if err = ioutil.WriteFile("./config.json", blob, 0644); err != nil {
			t.Fatalf("Failed to write default config.json: %v", err)
		}
		defer os.Remove("./config.json")
	} else {
		t.Fatalf("Failed to serialize %v", err)
	}

	cfg2, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	// And we can't predict what advertise address we'll detect
	cfg2.Aggregation.AdvertiseAddress = cfg.Aggregation.AdvertiseAddress
	// And UUID's won't match either
	cfg.UUID = ""
	cfg2.UUID = ""

	if !reflect.DeepEqual(cfg2, cfg) {
		t.Fatalf("Defaults should match but didn't \n\n%v\n\n%v", cfg2, cfg)
	}

}

func TestDefaultResources(t *testing.T) {
	// Check that giving empty resources results in empty resources
	blob := `{"Resources":[]}`
	if err := ioutil.WriteFile("./config.json", []byte(blob), 0644); err != nil {
		t.Fatalf("Failed to write default config.json: %v", err)
	}
	defer os.Remove("./config.json")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Resources) != 0 {
		t.Error("Default resources should not be applied if specified in config")
	}

	// Check that not specifying resources results in all the defaults
	blob = `{}`
	if err = ioutil.WriteFile("./config.json", []byte(blob), 0644); err != nil {
		t.Fatalf("Failed to write default config.json: %v", err)
	}
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Resources) == 0 {
		t.Error("Default resources should be applied if not in config")
	}

	// Check that specifying one resource results in one resource
	blob = `{"Resources": "Pods"}`
	if err = ioutil.WriteFile("./config.json", []byte(blob), 0644); err != nil {
		t.Fatalf("Failed to write default config.json: %v", err)
	}
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Resources) != 1 || cfg.Resources[0] != "Pods" {
		t.Error("Incorrect resources in config")
	}

}

func TestLoadAllPlugins(t *testing.T) {
	cfg := &Config{
		PluginSearchPath: []string{"./examples/plugins.d"},
		PluginSelections: []plugin.Selection{
			plugin.Selection{Name: "systemd-logs"},
			plugin.Selection{Name: "e2e"},
			plugin.Selection{Name: "heptio-e2e"},
		},
	}

	// Make sure we pick up the plugins directory from the root of the repo
	oldwd, _ := os.Getwd()
	os.Chdir("../..")
	defer os.Chdir(oldwd)

	err := loadAllPlugins(cfg)
	if err != nil {
		t.Fatal(err.Error())
	}
	plugins := cfg.getPlugins()
	if len(plugins) != len(cfg.PluginSelections) {
		t.Fatalf("Should have constructed %v plugins, got %v", len(cfg.PluginSelections), len(plugins))
	}

	// Get the names of all the loaded plugins for output on test failure.
	pluginNames := make([]string, len(plugins))
	for i, plugin := range plugins {
		pluginNames[i] = plugin.GetName()
	}

	for _, selection := range cfg.PluginSelections {
		found := false
		for _, loadedPlugin := range plugins {
			if loadedPlugin.GetName() == selection.Name {
				found = true
			}
		}
		if !found {
			t.Fatalf("Expected to find %v in %v", selection.Name, pluginNames)
		}
	}
}
