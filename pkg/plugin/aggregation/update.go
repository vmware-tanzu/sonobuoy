package aggregation

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

const annotationName = "sonobuoy.hept.io/status"

type key struct {
	node, name string
}

// Updater manages setting the Aggregator annotation with the current status
type Updater struct {
	sync.RWMutex
	positionLookup  map[key]*PluginStatus
	status          Status
	name, namespace string
	client          kubernetes.Interface
}

// NewUpdater creates an an updater that expects ExpectedResult.
func NewUpdater(expected []plugin.ExpectedResult, name, namespace string, client kubernetes.Interface) *Updater {
	updater := &Updater{
		positionLookup: make(map[key]*PluginStatus),
		status: Status{
			Plugins: make([]PluginStatus, len(expected)),
			Status:  RunningStatus,
		},
		name:      name,
		namespace: namespace,
		client:    client,
	}

	for i, result := range expected {
		updater.status.Plugins[i] = PluginStatus{
			Node:   result.NodeName,
			Plugin: result.ResultType,
			Status: RunningStatus,
		}

		updater.positionLookup[expectedToKey(result)] = &updater.status.Plugins[i]
	}

	return updater
}

func expectedToKey(result plugin.ExpectedResult) key {
	return key{node: result.NodeName, name: result.ResultType}
}

// Receive updates an individual plugin's status.
func (u *Updater) Receive(update *PluginStatus) error {
	u.Lock()
	defer u.Unlock()
	k := key{node: update.Node, name: update.Plugin}
	status, ok := u.positionLookup[k]
	if !ok {
		return fmt.Errorf("couldn't find key for %v", k)
	}

	status.Status = update.Status
	return u.status.updateStatus()
}

// Serialize json-encodes the status object.
func (u *Updater) Serialize() (string, error) {
	u.RLock()
	defer u.RUnlock()
	bytes, err := json.Marshal(u.status)
	return string(bytes), errors.Wrap(err, "couldn't marshall status")
}

// Annotate serialises the status json, then annotates the aggregator pod with the status.
func (u *Updater) Annotate() error {
	u.RLock()
	defer u.RUnlock()
	str, err := u.Serialize()
	if err != nil {
		return errors.Wrap(err, "couldn't serialize status")
	}

	patch := getPatch(str)
	bytes, err := json.Marshal(patch)
	if err != nil {
		return errors.Wrap(err, "couldn't encode patch")
	}

	_, err = u.client.CoreV1().Pods(u.namespace).Patch(u.name, types.MergePatchType, bytes)
	return errors.Wrap(err, "couldn't patch pod annotation")
}

// ReceiveAll takes a map of plugin.Result and calls Receive on all of them.
func (u *Updater) ReceiveAll(results map[string]*plugin.Result) {
	// Could have race conditions, but will be eventually consistent
	for _, result := range results {
		state := "complete"
		if result.Error != "" {
			state = "failed"
		}
		update := PluginStatus{
			Node:   result.NodeName,
			Plugin: result.ResultType,
			Status: state,
		}

		if err := u.Receive(&update); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"node":   update.Node,
					"plugin": update.Plugin,
					"status": state,
				},
			).WithError(err).Info("couldn't update plugin")
		}
	}
}

func getPatch(annotation string) map[string]interface{} {
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				annotationName: annotation,
			},
		},
	}
}
