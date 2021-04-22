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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// RunningStatus means the sonobuoy run is still in progress.
	RunningStatus string = "running"
	// CompleteStatus means the sonobuoy run is complete.
	CompleteStatus string = "complete"
	// PostProcessingStatus means the plugins are complete. The state is not
	// put in the more finalized, complete, status until any postprocessing is
	// done.
	PostProcessingStatus string = "post-processing"
	// FailedStatus means one or more plugins has failed and the run will not complete successfully.
	FailedStatus string = "failed"
)

// PluginStatus represents the current status of an individual plugin.
type PluginStatus struct {
	Plugin string `json:"plugin"`
	Node   string `json:"node"`
	Status string `json:"status"`

	ResultStatus       string         `json:"result-status"`
	ResultStatusCounts map[string]int `json:"result-counts"`

	Progress *plugin.ProgressUpdate `json:"progress,omitempty"`
}

// Status represents the current status of a Sonobuoy run.
// TODO(EKF): Find a better name for this struct/package.
type Status struct {
	Plugins []PluginStatus `json:"plugins"`
	Status  string         `json:"status"`
	Tarball TarInfo        `json:"tar-info,omitempty"`
}

// TarInfo is the type that contains information regarding the tarball
// that a user would get after running `sonobuoy retrieve`.
type TarInfo struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created"`
	SHA256    string    `json:"sha256"`
	Size      int64     `json:"size"`
}

// Key returns a unique identifier for the plugin that these status values
// correspond to.
func (p PluginStatus) Key() string {
	nodeName := p.Node
	if p.Node == "" {
		nodeName = plugin.GlobalResult
	}

	return p.Plugin + "/" + nodeName
}

// updateStatus sets the overall status field based on the values of all of the plugins' status.
func (s *Status) updateStatus() error {
	status := PostProcessingStatus
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

	// WIP/POC for cloudEvents. Could insert them in other locations as well or differently label everything.
	c, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatalf("failed to create client, %v", err)
	}

	// Create an Event.
	event := cloudevents.NewEvent()
	event.SetSource("sonobuoy")
	event.SetType("sonobuoy.status")
	event.SetData(cloudevents.ApplicationJSON, status)

	// Set a target.
	ctx := cloudevents.ContextWithTarget(context.Background(), "http://localhost:8080/")

	// Send that Event.
	if result := c.Send(ctx, event); cloudevents.IsUndelivered(result) {
		log.Fatalf("failed to send, %v", result)
	}

	return nil
}

// GetStatus returns the current status status on the sonobuoy pod. If the pod
// does not exist, is not running, or is missing the status annotation, an error
// is returned.
func GetStatus(client kubernetes.Interface, namespace string) (*Status, error) {
	if _, err := client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrapf(err, "failed to get namespace %v", namespace)
	}

	// Determine sonobuoy pod name
	podName, err := GetAggregatorPodName(client, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the name of the aggregator pod to get the status from")
	}

	pod, err := client.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("could not retrieve sonobuoy pod")
	}

	if pod.Status.Phase != corev1.PodRunning {
		return nil, fmt.Errorf("pod has status %q", pod.Status.Phase)
	}

	statusJSON, ok := pod.Annotations[StatusAnnotationName]
	if !ok {
		return nil, fmt.Errorf("missing status annotation %q", StatusAnnotationName)
	}

	var status Status
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return nil, errors.Wrap(err, "couldn't unmarshal the JSON status annotation")
	}

	return &status, nil
}
