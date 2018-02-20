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

	updater := NewUpdater(
		expected,
		"sonobuoy",
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
