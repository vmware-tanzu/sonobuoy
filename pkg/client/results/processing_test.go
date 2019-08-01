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

package results

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/job"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"

	"github.com/kylelemons/godebug/pretty"
)

var update = flag.Bool("update", false, "update golden files")

// TestPostProcessPlugin runs a series of checks against basic combinations
// of options: (job|daemonset)+(raw|junit)+(specify a specific file or not)
// and confirms the resulting Item is accurate.
func TestPostProcessPlugin(t *testing.T) {
	getPlugin := func(key, pluginDriver, format, outputFile string) plugin.Interface {
		switch pluginDriver {
		case "job":
			return &job.Plugin{Base: driver.Base{
				Definition: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName:   key,
						ResultFormat: format,
						ResultFile:   outputFile,
					},
				},
			}}
		case "daemonset":
			return &daemonset.Plugin{Base: driver.Base{
				Definition: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName:   key,
						ResultFormat: format,
						ResultFile:   outputFile,
					},
				},
			}}
		default:
			t.Fatalf("Invalid driver specified: %v", pluginDriver)
		}
		return nil
	}

	mockDataDir := func(key string) string {
		return filepath.Join("testdata", "mockResults")
	}
	expectResults := func(key string) string {
		return filepath.Join("testdata", "mockResults", "plugins", key, key+".golden.json")
	}

	testCases := []struct {
		desc        string
		plugin      plugin.Interface
		expectedErr string

		// key is used to lookup both the directory and the expected results.
		key string
	}{
		{
			desc:   "Job junit 2 files",
			key:    "job-junit-02",
			plugin: getPlugin("job-junit-02", "job", "junit", ""),
		}, {
			desc:   "Job junit 1 file, others ignored",
			key:    "job-junit-01",
			plugin: getPlugin("job-junit-01", "job", "junit", "output.xml"),
		}, {
			desc:   "Daemonset junit 2 files",
			key:    "ds-junit-02",
			plugin: getPlugin("ds-junit-02", "daemonset", "junit", ""),
		}, {
			desc:   "Daemonset junit 1 file, others ignored",
			key:    "ds-junit-01",
			plugin: getPlugin("ds-junit-01", "daemonset", "junit", "output.xml"),
		}, {
			desc:   "Job raw 2 files",
			key:    "job-raw-02",
			plugin: getPlugin("job-raw-02", "job", "raw", ""),
		}, {
			desc:   "Job raw 1 file, others ignored",
			key:    "job-raw-01",
			plugin: getPlugin("job-raw-01", "job", "raw", "output.xml"),
		}, {
			desc:   "Daemonset raw 2 files",
			key:    "ds-raw-02",
			plugin: getPlugin("ds-raw-02", "daemonset", "raw", ""),
		}, {
			desc:   "Daemonset raw 1 file, others ignored",
			key:    "ds-raw-01",
			plugin: getPlugin("ds-raw-01", "daemonset", "raw", "output.xml"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			item, err := PostProcessPlugin(tc.plugin, mockDataDir(tc.key))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if *update {
				// Update all the golden files instead of actually testing against them.
				itemBytes, err := json.Marshal(item)
				if err != nil {
					t.Fatalf("Failed to marshal item: %v", err)
				}
				ioutil.WriteFile(expectResults(tc.key), itemBytes, 0666)
			} else {
				// Read in golden file and unmarshal. Easier to debug differences in the items than
				// comparing the bytes directly.
				fileData, err := ioutil.ReadFile(expectResults(tc.key))
				if err != nil {
					t.Fatalf("Failed to read golden file %v: %v", expectResults(tc.key), err)
				}
				var expectedItem Item
				err = json.Unmarshal(fileData, &expectedItem)
				if err != nil {
					t.Fatalf("Failed to unmarshal golden file %v: %v", expectResults(tc.key), err)
				}
				if diff := pretty.Compare(expectedItem, item); diff != "" {
					t.Fatalf("\n\n%s\n", diff)
				}
			}

		})
	}
}

func TestAggregateStatus(t *testing.T) {
	tcs := []struct {
		desc     string
		input    []Item
		expected string

		// Ensures the items are actually updated despite their initial values.
		expectedItems []Item
	}{
		{
			desc:     "Empty defaults to passed",
			expected: StatusPassed,
		}, {
			desc:          "Single pass passes",
			input:         []Item{{Status: StatusPassed}},
			expectedItems: []Item{{Status: StatusPassed}},
			expected:      StatusPassed,
		}, {
			desc:          "Single fail fails",
			input:         []Item{{Status: StatusFailed}},
			expectedItems: []Item{{Status: StatusFailed}},
			expected:      StatusFailed,
		}, {
			desc:          "Misc other values pass",
			input:         []Item{{Status: "foobar"}},
			expectedItems: []Item{{Status: "foobar"}},
			expected:      StatusPassed,
		}, {
			desc: "Single failure in group causes failure",
			input: []Item{
				{Status: StatusPassed},
				{Status: StatusFailed},
			},
			expectedItems: []Item{
				{Status: StatusPassed},
				{Status: StatusFailed},
			},
			expected: StatusFailed,
		}, {
			desc: "Nested failure causes failure",
			input: []Item{
				{
					Status: StatusPassed,
					Items: []Item{
						{Status: StatusFailed},
					},
				},
				{Status: StatusPassed},
			},
			expectedItems: []Item{
				{
					Status: StatusFailed,
					Items: []Item{
						{Status: StatusFailed},
					},
				},
				{Status: StatusPassed},
			},
			expected: StatusFailed,
		}, {
			desc: "Deep branches should aggregate their items and return if failure",
			input: []Item{
				{
					Name:   "top of a branch",
					Status: StatusPassed,
					Items: []Item{
						{
							Name: "passing node",
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								},
							},
						},
						{
							Name: "failing node",
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								}, {
									Name:   "second leaf fails and should fail branch",
									Status: StatusFailed,
								}, {
									Name:   "third leaf passes as well",
									Status: StatusPassed,
								},
							},
						},
					},
				},
			},
			expectedItems: []Item{
				{
					Name:   "top of a branch",
					Status: StatusFailed,
					Items: []Item{
						{
							Name:   "passing node",
							Status: StatusPassed,
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								},
							},
						},
						{
							Name:   "failing node",
							Status: StatusFailed,
							Items: []Item{
								{
									Name:   "first leaf passes",
									Status: StatusPassed,
								}, {
									Name:   "second leaf fails and should fail branch",
									Status: StatusFailed,
								}, {
									Name:   "third leaf passes as well",
									Status: StatusPassed,
								},
							},
						},
					},
				},
			},
			expected: StatusFailed,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out := aggregateStatus(tc.input...)
			if out != tc.expected {
				t.Errorf("Expected %v but got %v", tc.expected, out)
			}

			if diff := pretty.Compare(tc.expectedItems, tc.input); diff != "" {
				t.Errorf("\n\n%s\n", diff)
			}
		})
	}
}
