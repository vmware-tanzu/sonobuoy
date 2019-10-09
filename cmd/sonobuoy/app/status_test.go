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
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
)

var expectedSummary = `         PLUGIN     STATUS   RESULT   COUNT
            e2e   complete   passed       102 Jan 06 15:04 UTC
   systemd_logs   complete   failed       102 Jan 06 15:04 UTC
   systemd_logs    running                202 Jan 06 15:04 UTC

Sonobuoy is still running. Runs can take up to 60 minutes.
`
var expectedShowAll = `         PLUGIN     NODE     STATUS   RESULT
            e2e            complete   passed
   systemd_logs   node01    running         
   systemd_logs   node02   complete   failed
   systemd_logs   node03    running         

Sonobuoy is still running. Runs can take up to 60 minutes.
`

var exampleStatus = aggregation.Status{
	Status: "running",
	Plugins: []aggregation.PluginStatus{
		{
			Plugin:       "e2e",
			Node:         "",
			Status:       "complete",
			ResultStatus: "passed",
			CurrentTime:  "02 Jan 06 15:04 UTC",
		},
		{
			Plugin:      "systemd_logs",
			Node:        "node01",
			Status:      "running",
			CurrentTime: "02 Jan 06 15:04 UTC",
		},
		{
			Plugin:       "systemd_logs",
			Node:         "node02",
			Status:       "complete",
			ResultStatus: "failed",
			CurrentTime:  "02 Jan 06 15:04 UTC",
		},
		{
			Plugin:      "systemd_logs",
			Node:        "node03",
			Status:      "running",
			CurrentTime: "02 Jan 06 15:04 UTC",
		},
	},
}

func TestPrintStatus(t *testing.T) {
	tests := []struct {
		expected string
		name     string
		f        func(w io.Writer, s *aggregation.Status) error
	}{
		{
			expected: expectedSummary,
			name:     "StatusSummary",
			f:        printSummary,
		},
		{
			expected: expectedShowAll,
			name:     "StatusShowAll",
			f:        printAll,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b bytes.Buffer
			err := test.f(&b, &exampleStatus)
			if err != nil {
				t.Fatalf("expected err to be nil, got %v", err)
			}
			if b.String() != test.expected {
				t.Errorf("expected output to be \n%v, got \n%v", test.expected, b.String())
			}
		})
	}
}
