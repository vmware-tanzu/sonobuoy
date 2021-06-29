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
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
)

// getPlugins gets the list of plugins selected for this configuration.
func (cfg *Config) getPlugins() []plugin.Interface {
	return cfg.LoadedPlugins
}

func TestOpenConfigFile(t *testing.T) {

	// Set up 3 files with contents matching their path for this test. Cleanup afterwards.
	f1, f2 := "TestOpenConfigFile1", "TestOpenConfigFile2"
	for _, v := range []string{f1, f2} {
		err := ioutil.WriteFile(v, []byte(v), 0644)
		if err != nil {
			t.Fatalf("Failed to setup test files: %v", err)
		}
		defer os.Remove(v)
	}

	testCases := []struct {
		desc       string
		files      []string
		expectPath string
		expectErr  string
	}{
		{
			desc:       "Open existing file",
			files:      []string{f1},
			expectPath: f1,
		}, {
			desc:      "File DNE",
			files:     []string{"bad"},
			expectErr: "opening config file: open bad: no such file or directory",
		}, {
			desc:       "File DNE and falls back to next file",
			files:      []string{"bad", f1},
			expectPath: f1,
		}, {
			desc:       "Returns first good file",
			files:      []string{f2, f1},
			expectPath: f2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			f, fpath, err := openFiles(tc.files...)
			if f != nil {
				defer f.Close()
			}

			switch {
			case err != nil && len(tc.expectErr) == 0:
				t.Fatalf("Expected nil error but got %q", err)
			case err != nil && len(tc.expectErr) > 0:
				if fmt.Sprint(err) != tc.expectErr {
					t.Errorf("Expected error \n\t%q\nbut got\n\t%q", tc.expectErr, err)
				}
				return
			case err == nil && len(tc.expectErr) > 0:
				t.Fatalf("Expected error %q but got nil", tc.expectErr)
			default:
				// No error
			}

			if fpath != tc.expectPath {
				t.Errorf("Expected %v but got %v", tc.expectPath, fpath)
			}

			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatalf("Failed to read file %v: %v", fpath, err)
			}
			if string(b) != fpath {
				t.Errorf("Expected contents of file %v to be %v but got %v", fpath, fpath, string(b))
			}
		})
	}
}

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

	// We can't predict what advertise address we'll detect
	cfg2.Aggregation.AdvertiseAddress = cfg.Aggregation.AdvertiseAddress

	if reflect.DeepEqual(cfg2, cfg) {
		t.Fatalf("Defaults shouldnt match at first since the Loader adds UUID but did: \n\n%v\n\n%v", cfg2, cfg)
	}
	cfg.UUID = cfg2.UUID
	if !reflect.DeepEqual(cfg2, cfg) {
		t.Fatalf("Defaults should match but didn't \n\n%v\n\n%v", cfg2, cfg)
	}
}

func TestLoadConfigSetsUUID(t *testing.T) {
	cfg := New()

	if blob, err := json.Marshal(&cfg); err == nil {
		if err = ioutil.WriteFile("./config.json", blob, 0644); err != nil {
			t.Fatalf("Failed to write default config.json: %v", err)
		}
		defer os.Remove("./config.json")
	} else {
		t.Fatalf("Failed to serialize %v", err)
	}

	loadedCfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if loadedCfg.UUID == "" {
		t.Error("loaded config should have a UUID but was empty")
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
	if cfg.Resources == nil {
		t.Error("Empty resources should not be converted to nil")
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
	if cfg.Resources != nil {
		t.Error("Nil resources should stay nil when loaded to imply query all resources.")
	}

	// Check that specifying one resource results in one resource
	blob = `{"Resources": ["Pods"]}`
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
		PluginSearchPath: []string{"testdata/plugins.d"},
		PluginSelections: []plugin.Selection{
			{Name: "systemd-logs"},
			{Name: "e2e"},
			{Name: "heptio-e2e"},
		},
	}

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
