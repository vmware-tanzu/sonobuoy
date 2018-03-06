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

import (
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestCreateUpdater(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd"},
		{NodeName: "node2", ResultType: "systemd"},
		{NodeName: "", ResultType: "e2e"},
	}

	updater := newUpdater(
		expected,
		"heptio-sonobuoy-test",
		nil,
	)

	if err := updater.Receive(&PluginStatus{
		Status: FailedStatus,
		Node:   "node1",
		Plugin: "systemd",
	}); err != nil {
		t.Errorf("unexpected error receiving update %v", err)
	}

	if updater.status.Status != FailedStatus {
		t.Errorf("expected status to be failed, got %v", updater.status.Status)
	}
}
