package status

import (
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

type key struct {
	node, name string
}

type Updater struct {
	positionLookup map[key]*Plugin
	status         Status
}

// NewUpdater creates an an updater that expects ExpectedResult
func NewUpdater(expected []plugin.ExpectedResult) *Updater {
	updater := &Updater{
		positionLookup: make(map[key]*Plugin),
		status: Status{
			Plugins: make([]Plugin, len(expected)),
			Status:  Running,
		},
	}

	for i, result := range expected {
		updater.status.Plugins[i] = Plugin{
			Node:   result.NodeName,
			Plugin: result.ResultType,
			Status: Running,
		}

		updater.positionLookup[expectedToKey(result)] = &updater.status.Plugins[i]
	}

	return updater
}

func expectedToKey(result plugin.ExpectedResult) key {
	return key{node: result.NodeName, name: result.ResultType}
}

func (u *Updater) Receive(update *Plugin) error {
	k := key{node: update.Node, name: update.Plugin}
	status, ok := u.positionLookup[k]
	if !ok {
		return fmt.Errorf("couldn't find key for %v", k)
	}

	status.Status = update.Status
	return u.status.updateStatus()
}
