/*
Copyright the Sonobuoy contributors 2019

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
	"fmt"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestPluginEnvSet(t *testing.T) {
	testCases := []struct {
		desc      string
		init      PluginEnvVars
		input     string
		expect    PluginEnvVars
		expectErr string
	}{
		{
			desc:  "Set value",
			init:  PluginEnvVars(map[string]map[string]string{}),
			input: "name.env=val",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
		}, {
			desc: "Set value again same plugin",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name.env2=val2",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val", "env2": "val2"},
			}),
		}, {
			desc: "Set value again different plugin",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name2.env2=val2",
			expect: PluginEnvVars(map[string]map[string]string{
				"name":  map[string]string{"env": "val"},
				"name2": map[string]string{"env2": "val2"},
			}),
		}, {
			desc: "Override value already in map",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name.env=val2",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val2"},
			}),
		}, {
			desc: "Empty string if no equals",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name.env2",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val", "env2": ""},
			}),
		}, {
			desc: "Empty string if equals but no value",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name.env2=",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val", "env2": ""},
			}),
		}, {
			desc: "Splits on first period",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name.env.with.dot=val2",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val", "env.with.dot": "val2"},
			}),
		}, {
			desc: "Splits on first equals and period",
			init: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
			input: "name.env.with.dot=val=with=equals",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val", "env.with.dot": "val=with=equals"},
			}),
		}, {
			desc:  "Starting with nil map",
			init:  PluginEnvVars(nil),
			input: "name.env=val",
			expect: PluginEnvVars(map[string]map[string]string{
				"name": map[string]string{"env": "val"},
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.init.Set(tc.input)
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

			if diff := pretty.Compare(tc.expect, tc.init); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
