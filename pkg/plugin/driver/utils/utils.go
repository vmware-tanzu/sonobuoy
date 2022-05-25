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

package utils

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"

	gouuid "github.com/satori/go.uuid"
	v1 "k8s.io/api/core/v1"
)

const (
	// terminatedContainerWindow is the amount of time after a plugins main container terminates
	// that we consider it a failure mode. This handles the situation where the plugin container
	// exits without returning results.
	terminatedContainerWindow = 5 * time.Minute

	// maxWaitForImageTime is the amount of time that we allow for pods to recover from failed image pulls.
	// This allows for transient image pull errors to be recovered from without marking the plugin as failed.
	maxWaitForImageTime = 5 * time.Minute
)

// GetSessionID generates a new session id.
// This is essentially an instance of a running plugin.
func GetSessionID() string {
	uuid, _ := gouuid.NewV4()
	ret := make([]byte, hex.EncodedLen(8))
	hex.Encode(ret, uuid.Bytes()[0:8])
	return string(ret)
}

// IsPodFailing returns whether a plugin's pod is failing and isn't likely to
// succeed.
func IsPodFailing(pod *v1.Pod) (bool, string) {
	// Check if the pod is unschedulable
	for _, cond := range pod.Status.Conditions {
		if cond.Reason == "Unschedulable" {
			return true, fmt.Sprintf("Can't schedule pod: %v", cond.Message)
		}
	}

	for _, cstatus := range pod.Status.ContainerStatuses {
		// Check if a container in the pod is restarting multiple times
		if cstatus.RestartCount > 2 {
			errstr := fmt.Sprintf("Container %v has restarted unsuccessfully %v times", cstatus.Name, cstatus.RestartCount)
			return true, errstr
		}

		// Check if it can't fetch its image within the maximum wait time
		if waiting := cstatus.State.Waiting; waiting != nil && pod.Status.StartTime != nil {
			elapsedPodTime := time.Since(pod.Status.StartTime.Time)
			if elapsedPodTime > maxWaitForImageTime && (waiting.Reason == "ImagePullBackOff" || waiting.Reason == "ErrImagePull") {
				errstr := fmt.Sprintf("Failed to pull image %v for container %v within %v. Container is in state %v", cstatus.Image, cstatus.Name, maxWaitForImageTime, waiting.Reason)
				return true, errstr
			}
		}

		// Container terminated without reporting results.
		if cstatus.State.Terminated != nil {
			// Ensure we give some time to process the results.
			if time.Since(cstatus.State.Terminated.FinishedAt.Time) > terminatedContainerWindow {
				errstr := fmt.Sprintf("Container %v is in a terminated state (exit code %v) due to reason: %v: %v",
					cstatus.Name,
					cstatus.State.Terminated.ExitCode,
					cstatus.State.Terminated.Reason,
					cstatus.State.Terminated.Message,
				)
				return true, errstr
			}
		}
	}

	return false, ""
}

// MakeErrorResult constructs a plugin.Result given an error message and error
// data.  errdata is a map that will be placed in the sonobuoy results tarball
// for this plugin as a JSON file, so it's what users will see for why the
// plugin failed.  If errdata["error"] is not set, it will be filled in with an
// "Unknown error" string.
func MakeErrorResult(resultType string, errdata map[string]interface{}, nodeName string) *plugin.Result {
	errJSON, _ := json.Marshal(errdata)

	errstr := "Unknown error"
	if e, ok := errdata["error"]; ok {
		errstr = e.(string)
	}

	return &plugin.Result{
		Body:       bytes.NewReader(errJSON),
		Error:      errstr,
		ResultType: resultType,
		NodeName:   nodeName,
		MimeType:   "application/json",
		Filename:   "error.json",
	}
}
