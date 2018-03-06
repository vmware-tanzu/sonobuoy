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

package aggregation

import "testing"

func TestUpdateStatus(t *testing.T) {
	statusTests := []struct {
		name           string
		pluginStatuses []string
		expectedStatus string
	}{
		{
			name:           "empty is complete",
			pluginStatuses: []string{},
			expectedStatus: "complete",
		},
		{
			name:           "all completed is complete",
			pluginStatuses: []string{"complete", "complete", "complete"},
			expectedStatus: "complete",
		},
		{
			name:           "one running is running",
			pluginStatuses: []string{"complete", "running", "complete"},
			expectedStatus: "running",
		},
		{
			name:           "one failed is failed",
			pluginStatuses: []string{"running", "failed", "complete"},
			expectedStatus: "failed",
		},
	}

	for _, test := range statusTests {
		t.Run(test.name, func(t *testing.T) {
			plugins := make([]PluginStatus, len(test.pluginStatuses))
			for i, pluginStatus := range test.pluginStatuses {
				plugins[i] = PluginStatus{Status: pluginStatus}
			}

			status := &Status{Plugins: plugins}
			err := status.updateStatus()
			if err != nil {
				t.Errorf("got unexpected error updating status: %v", err)
			}
			if status.Status != test.expectedStatus {
				t.Errorf("expected status to be %q, got %q", test.expectedStatus, status.Status)
			}
		})
	}
}

func TestUpdateInvalidStatus(t *testing.T) {
	status := &Status{
		Plugins: []PluginStatus{
			{Status: "unknown"},
		},
		Status: "",
	}

	if err := status.updateStatus(); err == nil {
		t.Error("expected err to be unknown status, got nil")
	}
}
