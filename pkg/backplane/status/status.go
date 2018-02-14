package status

import "fmt"

const (
	// Running means the sonobuoy run is still in progress
	Running string = "running"
	// Complete means the sonobuoy run is complete
	Complete string = "complete"
	// Failed means one or more plugins has failed and the run will not complete successfully
	Failed string = "failed"
)

// Plugin represents the current status of an individual plugin
type Plugin struct {
	Plugin string `json:"plugin"`
	Node   string `json:"node"`
	Status string `json:"status"`
}

// Status represents the current status of a Sonobuoy run
type Status struct {
	Plugins []Plugin `json:"plugins"`
	Status  string   `json:"status"`
}

func (s *Status) updateStatus() error {
	status := Complete
	for _, plugin := range s.Plugins {
		switch plugin.Status {
		case Complete:
			continue
		case Failed:
			status = Failed
		case Running:
			if status != Failed {
				status = Running
			}
		default:
			return fmt.Errorf("unknown status %s", plugin.Status)
		}
	}
	s.Status = status
	return nil
}
