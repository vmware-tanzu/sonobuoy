/*
Copyright 2017 Heptio Inc.

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
	cfg := NewWithDefaults()

	cfg.Filters.Namespaces = "funky*"

	if blob, err := json.Marshal(&cfg); err == nil {
		if err = ioutil.WriteFile("./config.json", blob, 0644); err != nil {
			t.Fatalf("Failed to write default config.json: %v", err)
		}
		defer os.Remove("./config.json")
	} else {
		t.Fatalf("Failed to serialize ", err)
	}

	cfg2, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	// Tests shouldn't fail just because KUBECONFIG is customized
	cfg2.Kubeconfig = cfg.Kubeconfig
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
		PluginSearchPath: []string{"./plugins.d"},
		PluginSelections: []plugin.Selection{
			plugin.Selection{Name: "systemd_logs"},
			plugin.Selection{Name: "e2e"},
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
	if len(plugins) != 2 {
		t.Fatalf("Should have constructed 2 plugins, got %v", len(plugins))
	}

	dsplugin := plugins[0]
	if name := dsplugin.GetName(); name != "systemd_logs" {
		t.Fatalf("First result of LoadAllPlugins has the wrong name: %v != systemd_logs", name)
	}

	if len(dsplugin.GetPodSpec().Containers) != 2 {
		t.Fatalf("DaemonSetPlugin should have 2 containers, got %v", len(dsplugin.GetPodSpec().Containers))
	}

	firstContainerName := dsplugin.GetPodSpec().Containers[0].Name
	if firstContainerName != "systemd-logs" {
		t.Fatalf("systemd_logs plugin had unexpected container name (%v != %v)", firstContainerName, "systemd-logs")
	}

	jobplugin := plugins[1]
	if name := jobplugin.GetName(); name != "e2e" {
		t.Fatalf("Second result of LoadAllPlugins has the wrong name: %v != e2e", name)
	}

	if len(dsplugin.GetPodSpec().Containers) != 2 {
		t.Fatalf("JobPlugin should have 1 container, got 2", len(jobplugin.GetPodSpec().Containers))
	}

	firstContainerName = jobplugin.GetPodSpec().Containers[0].Name
	if firstContainerName != "e2e" {
		t.Fatalf("e2e plugin had unexpected container name (%v != %v)", firstContainerName, "e2e")
	}
}
