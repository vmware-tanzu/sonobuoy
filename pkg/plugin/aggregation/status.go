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
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return nil
}

// GetStatus returns the current status status on the sonobuoy pod. If the pod
// does not exist, is not running, or is missing the status annotation, an error
// is returned.
func GetStatus(client kubernetes.Interface, namespace string) (*Status, error) {
	if _, err := client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{}); err != nil {
		return nil, errors.Wrap(err, "sonobuoy namespace does not exist")
	}

	// Determine sonobuoy pod name
	podName, err := GetStatusPodName(client, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the name of the aggregator pod to get the status from")
	}

	pod, err := client.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
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
