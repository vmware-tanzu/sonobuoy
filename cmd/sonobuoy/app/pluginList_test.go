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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	"github.com/kylelemons/godebug/pretty"
)

func TestSetPluginList(t *testing.T) {
	serveFile := func(filepath string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			b, err := ioutil.ReadFile(filepath)
			if err != nil {
				t.Fatal(err)
			}
			w.Write(b)
		})
	}
	ts := httptest.NewServer(serveFile("testdata/goodmanifest.yaml"))
	defer ts.Close()

	testCases := []struct {
		desc      string
		list      pluginList
		input     string
		expect    pluginList
		expectErr string
	}{
		{
			desc:      "empty filename",
			expectErr: `unable to stat "": stat : no such file or directory`,
		}, {
			desc:      "file does not exist",
			input:     "no-file",
			expectErr: `unable to stat "no-file": stat no-file: no such file or directory`,
		}, {
			desc:      "bad manifest",
			input:     "testdata/badmanifest.yaml",
			expectErr: `loading plugin from file "testdata/badmanifest.yaml": failed to load plugin: couldn't decode yaml for plugin definition: couldn't get version/kind; json parse error: json: cannot unmarshal string into Go value of type struct { APIVersion string "json:\"apiVersion,omitempty\""; Kind string "json:\"kind,omitempty\"" }`,
		}, {
			desc:   "loading e2e",
			input:  "e2e",
			list:   pluginList{},
			expect: pluginList{DynamicPlugins: []string{"e2e"}},
		}, {
			desc:   "loading systemd-logs",
			input:  "systemd-logs",
			list:   pluginList{},
			expect: pluginList{DynamicPlugins: []string{"systemd-logs"}},
		}, {
			desc:  "loading from file",
			input: "testdata/goodmanifest.yaml",
			list:  pluginList{},
			expect: pluginList{StaticPlugins: []*manifest.Manifest{
				{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "test"}},
			}},
		}, {
			desc:  "dynamic and static",
			input: "e2e",
			list: pluginList{StaticPlugins: []*manifest.Manifest{
				{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "test"}},
			}},
			expect: pluginList{
				StaticPlugins: []*manifest.Manifest{
					{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "test"}},
				},
				DynamicPlugins: []string{"e2e"},
			},
		}, {
			desc:  "multiple dynamic",
			input: "systemd-logs",
			list:  pluginList{DynamicPlugins: []string{"e2e"}},
			expect: pluginList{
				DynamicPlugins: []string{"e2e", "systemd-logs"},
			},
		}, {
			desc:  "loading from url",
			input: ts.URL,
			list:  pluginList{},
			expect: pluginList{StaticPlugins: []*manifest.Manifest{
				{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "test"}},
			}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.list.Set(tc.input)
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

			// We don't want to worry about diffs in the plugin cache location so just ignore those fields here.
			tc.list.InstallDir, tc.list.initInstallDir = "", false

			if diff := pretty.Compare(tc.expect, tc.list); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
