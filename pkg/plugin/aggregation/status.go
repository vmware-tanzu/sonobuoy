package aggregation

import "fmt"

const (
	// RunningStatus means the sonobuoy run is still in progress.
	RunningStatus string = "running"
	// CompleteStatus means the sonobuoy run is complete.
	CompleteStatus string = "complete"
	// FailedStatus means one or more plugins has failed and the run will not complete successfully.
	FailedStatus string = "failed"
)

// PluginStatus represents the current status of an individual plugin.
type PluginStatus struct {
	Plugin string `json:"plugin"`
	Node   string `json:"node"`
	Status string `json:"status"`
}

// Status represents the current status of a Sonobuoy run.
// TODO(EKF): Find a better name for this struct/package.
type Status struct {
	Plugins []PluginStatus `json:"plugins"`
	Status  string         `json:"status"`
}

func (s *Status) updateStatus() error {
	status := CompleteStatus
	for _, plugin := range s.Plugins {
		switch plugin.Status {
		case CompleteStatus:
			continue
		case FailedStatus:
			status = FailedStatus
		case RunningStatus:
			if status != FailedStatus {
				status = RunningStatus
			}
		default:
			return fmt.Errorf("unknown status %s", plugin.Status)
		}
	}
	s.Status = status
	return nil
}
