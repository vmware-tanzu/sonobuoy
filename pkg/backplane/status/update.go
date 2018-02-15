package status

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

const annotationName = "sonobuoy.hept.io/status"

type key struct {
	node, name string
}

type Updater struct {
	sync.RWMutex
	positionLookup  map[key]*Plugin
	status          Status
	name, namespace string
	client          kubernetes.Interface
}

// NewUpdater creates an an updater that expects ExpectedResult
func NewUpdater(expected []plugin.ExpectedResult, name, namespace string, client kubernetes.Interface) *Updater {
	updater := &Updater{
		positionLookup: make(map[key]*Plugin),
		status: Status{
			Plugins: make([]Plugin, len(expected)),
			Status:  Running,
		},
		name:      name,
		namespace: namespace,
		client:    client,
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

// Receive recieves an update to an individual plugin
func (u *Updater) Receive(update *Plugin) error {
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

// Serialize json-encodes the status object
func (u *Updater) Serialize() (string, error) {
	u.RLock()
	defer u.RUnlock()
	bytes, err := json.Marshal(u.status)
	return string(bytes), errors.Wrap(err, "couldn't marshall status")
}

// Annotate serialises the status json, then updates the Sonobuoy pod with the annotation
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

func getPatch(annotation string) map[string]interface{} {
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				annotationName: annotation,
			},
		},
	}
}
