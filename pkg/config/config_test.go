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

package config_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
)

func TestDefaults(t *testing.T) {
	cfg1 := config.New()
	cfg2 := config.New()

	if !reflect.DeepEqual(&cfg2, &cfg1) {
		t.Fatalf("Defaults should match but didn't")
	}
}

func TestEmptySlicePreservation(t *testing.T) {
	for _, tc := range []struct {
		desc    string
		cfgfunc func() *config.Config
	}{
		{
			desc:    "Default config",
			cfgfunc: config.New,
		}, {
			desc: "Empty resources and plugin selection",
			cfgfunc: func() *config.Config {
				cfg := config.New()
				cfg.Resources = []string{}
				cfg.PluginSelections = []plugin.Selection{}
				return cfg
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			cfg1 := tc.cfgfunc()
			b, err := json.Marshal(cfg1)
			if err != nil {
				t.Fatalf("Unable to marshal config: %v", err)
			}

			var cfg2 *config.Config
			err = json.Unmarshal(b, &cfg2)
			if err != nil {
				t.Fatalf("Unable to unmarshal config: %v", err)
			}

			if !reflect.DeepEqual(cfg1, cfg2) {
				t.Fatalf("Values did not match after serialization/deserialization: \nGot: %#v\n\nWant: %#v\n", cfg2, cfg1)
			}
		})
	}
}
