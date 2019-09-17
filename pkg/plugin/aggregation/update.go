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
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

const (
	StatusAnnotationName = "sonobuoy.hept.io/status"
	StatusPodLabel       = "run=sonobuoy-master"
	DefaultStatusPodName = "sonobuoy"
)

// NoPodWithLabelError represents an error encountered when a pod with a given label can't be found
type NoPodWithLabelError string

func (n NoPodWithLabelError) Error() string {
	return string(n)
}

// updater manages setting the Aggregator annotation with the current status
type updater struct {
	sync.RWMutex
	positionLookup map[string]*PluginStatus
	status         Status
	namespace      string
	client         kubernetes.Interface
}

// newUpdater creates an an updater that expects ExpectedResult.
func newUpdater(expected []plugin.ExpectedResult, namespace string, client kubernetes.Interface) *updater {
	u := &updater{
		positionLookup: make(map[string]*PluginStatus),
		status: Status{
			Plugins: make([]PluginStatus, len(expected)),
			Status:  RunningStatus,
		},
		namespace: namespace,
		client:    client,
	}

	for i, result := range expected {
		u.status.Plugins[i] = PluginStatus{
			Node:   result.NodeName,
			Plugin: result.ResultType,
			Status: RunningStatus,
		}

		u.positionLookup[result.ID()] = &u.status.Plugins[i]
	}

	return u
}

// Receive updates the updater's internal status of an individual plugin.
func (u *updater) Receive(update *PluginStatus) error {
	u.Lock()
	defer u.Unlock()

	status, ok := u.positionLookup[update.Key()]
	if !ok {
		return fmt.Errorf("couldn't find key for %v", update.Key())
	}

	// Updating status updates the u.status object.
	deepCopyPluginStatus(status, update)
	return u.status.updateStatus()
}

func deepCopyPluginStatus(dst, src *PluginStatus) {
	dst.Plugin = src.Plugin
	dst.Node = src.Node
	dst.Status = src.Status
	dst.ResultStatus = src.ResultStatus

	if src.ResultStatusCounts != nil {
		dst.ResultStatusCounts = map[string]int{}
		for k, v := range src.ResultStatusCounts {
			dst.ResultStatusCounts[k] = v
		}
	}
	if src.Progress != nil {
		dst.Progress = &plugin.ProgressUpdate{
			PluginName: src.Progress.PluginName,
			Node:       src.Progress.Node,
			Timestamp:  src.Progress.Timestamp,
			Message:    src.Progress.Message,
			Total:      src.Progress.Total,
			Completed:  src.Progress.Completed,
		}
		if src.Progress.Errors != nil {
			dst.Progress.Errors = make([]string, len(src.Progress.Errors))
			copy(dst.Progress.Errors, src.Progress.Errors)
		}
		if src.Progress.Failures != nil {
			dst.Progress.Failures = make([]string, len(src.Progress.Failures))
			copy(dst.Progress.Failures, src.Progress.Failures)
		}
	}
}

// Serialize json-encodes the status object.
func (u *updater) Serialize() (string, error) {
	u.RLock()
	defer u.RUnlock()
	bytes, err := json.Marshal(u.status)
	return string(bytes), errors.Wrap(err, "couldn't marshall status")
}

// Annotate serializes the status json, then annotates the aggregator pod with the status.
func (u *updater) Annotate(results map[string]*plugin.Result, progressUpdates map[string]*plugin.ProgressUpdate) error {
	u.ReceiveAll(results, progressUpdates)
	u.RLock()
	defer u.RUnlock()

	str, err := u.Serialize()
	if err != nil {
		return errors.Wrap(err, "couldn't serialize status")
	}

	patch := GetPatch(str)
	bytes, err := json.Marshal(patch)
	if err != nil {
		return errors.Wrap(err, "couldn't encode patch")
	}

	// Determine sonobuoy pod name
	podName, err := GetAggregatorPodName(u.client, u.namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get name of the aggregator pod to annotate")
	}

	_, err = u.client.CoreV1().Pods(u.namespace).Patch(podName, types.MergePatchType, bytes)
	return errors.Wrap(err, "couldn't patch pod annotation")
}

// TODO (tstclair): Evaluate if this should be exported.
// ReceiveAll takes a map of plugin.Result and plugin.ProgressUpdates and uses Recieve to turn
// update the updater's status.
func (u *updater) ReceiveAll(results map[string]*plugin.Result, progressUpdates map[string]*plugin.ProgressUpdate) {
	// With the addition of progress updates we have two maps of data; iterate once to get all the keys then once to process.
	keys := map[string]struct{}{}
	for k := range results {
		keys[k] = struct{}{}
	}
	for k := range progressUpdates {
		keys[k] = struct{}{}
	}

	// Could have race conditions, but will be eventually consistent
	for k := range keys {
		update := PluginStatus{}
		result, hasResult := results[k]
		switch {
		case !hasResult:
			update.Status = "running"
			update.Plugin = progressUpdates[k].PluginName
			update.Node = progressUpdates[k].Node
		case result.Error != "":
			update.Status = "failed"
			update.Plugin = results[k].ResultType
			update.Node = results[k].NodeName
		default:
			update.Status = "complete"
			update.Plugin = results[k].ResultType
			update.Node = results[k].NodeName
		}

		update.Progress = progressUpdates[k]
		if err := u.Receive(&update); err != nil {
			logrus.WithFields(
				logrus.Fields{
					"node":   update.Node,
					"plugin": update.Plugin,
					"status": update.Status,
				},
			).WithError(err).Info("couldn't update plugin")
		}
	}
}

// GetPatch takes a json encoded string and creates a map which can be used as
// a patch to indicate the Sonobuoy status.
func GetPatch(annotation string) map[string]interface{} {
	return map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": map[string]string{
				StatusAnnotationName: annotation,
			},
		},
	}
}

// GetAggregatorPod gets the sonobuoy aggregator pod based on its label.
// It returns NoPodWithLabelError in the case where a pod with sonobuoy aggregator label could not be found.
func GetAggregatorPod(client kubernetes.Interface, namespace string) (*v1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: StatusPodLabel,
	}

	podList, err := client.CoreV1().Pods(namespace).List(listOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list pods with label %q", StatusPodLabel)
	}

	switch {
	case len(podList.Items) == 0:
		logrus.Warningf("no pods found with label %q in namespace %s", StatusPodLabel, namespace)
		return nil, NoPodWithLabelError(fmt.Sprintf("no pods found with label %q in namespace %s", StatusPodLabel, namespace))

	case len(podList.Items) > 1:
		logrus.Warningf("Found more than one pod with label %q. Using pod with name %q", StatusPodLabel, podList.Items[0].GetName())
		return &podList.Items[0], nil
	default:
		return &podList.Items[0], nil
	}
}

// GetAggregatorPodName gets the sonobuoy aggregator pod name. It returns the default pod name
// if the pod cannot be found.
func GetAggregatorPodName(client kubernetes.Interface, namespace string) (string, error) {
	ap, err := GetAggregatorPod(client, namespace)

	if err != nil {
		switch err.(type) {
		case NoPodWithLabelError:
			logrus.Warningf("Aggregator pod not found, using default pod name %q: %v", DefaultStatusPodName, err)
			return DefaultStatusPodName, nil
		default:
			return "", errors.Wrap(err, "failed to get aggregator pod")
		}
	}

	return ap.GetName(), nil
}
