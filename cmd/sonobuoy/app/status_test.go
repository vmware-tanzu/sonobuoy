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
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
)

var exampleStatus = aggregation.Status{
	Status: "running",
	Plugins: []aggregation.PluginStatus{
		{
			Plugin:       "e2e",
			Node:         "global",
			Status:       "complete",
			ResultStatus: "failed",
			Progress: &plugin.ProgressUpdate{
				PluginName: "e2e", Node: "global",
				Total: 2, Completed: 1, Failures: []string{"a"},
			},
		},
		{
			Plugin: "systemd_logs",
			Node:   "node01",
			Status: "running",
		},
		{
			Plugin:       "systemd_logs",
			Node:         "node02",
			Status:       "complete",
			ResultStatus: "failed",
			Progress: &plugin.ProgressUpdate{
				PluginName: "systemd_logs", Node: "node02",
				Total: 3, Completed: 1, Failures: []string{"a"},
			},
		},
		{
			Plugin: "systemd_logs",
			Node:   "node03",
			Status: "running",
		},
	},
}

func TestPrintStatus(t *testing.T) {
	tests := []struct {
		expectFile string
		name       string
		f          func(w io.Writer, s *aggregation.Status) error
	}{
		{
			expectFile: "testdata/expectedSummary.golden",
			name:       "StatusSummary",
			f:          printSummary,
		},
		{
			expectFile: "testdata/expectedShowAll.golden",
			name:       "StatusShowAll",
			f:          printAll,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b bytes.Buffer
			err := tc.f(&b, &exampleStatus)
			if err != nil {
				t.Fatalf("expected err to be nil, got %v", err)
			}

			if *update {
				ioutil.WriteFile(tc.expectFile, b.Bytes(), 0666)
			} else {
				fileData, err := ioutil.ReadFile(tc.expectFile)
				if err != nil {
					t.Fatalf("Failed to read golden file %v: %v", tc.expectFile, err)
				}
				if !bytes.Equal(fileData, b.Bytes()) {
					t.Errorf("expected output to match goldenfile %v but got \n%v", tc.expectFile, b.String())
				}
			}
		})
	}
}
