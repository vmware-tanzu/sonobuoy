/*
Copyright 2017 Heptio Inc.

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
	"encoding/json"
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin"

	v1 "k8s.io/api/core/v1"
)

// IsPodFailing returns whether a plugin's pod is failing and isn't likely to
// succeed.
// TODO: this may require more revisions as we get more experience with
// various types of failures that can occur.
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

		// Check if it can't fetch its image
		if waiting := cstatus.State.Waiting; waiting != nil {
			if waiting.Reason == "ImagePullBackOff" || waiting.Reason == "ErrImagePull" {
				errstr := fmt.Sprintf("Container %v is in state %v", cstatus.Name, waiting.Reason)
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
		Extension:  ".json",
	}
}

// ApplyDefaultLabels applies a default label set to the given
// map[string]string.  All our resources should have a commmon label set,
// particularly a unique sesssion ID for sonobuoy run. This can allow fallback
// cleanup for this session by deleting any resources with
// `sonobuoy-run=<sessionID>`
func ApplyDefaultLabels(p plugin.Interface, labels map[string]string) map[string]string {
	labels["component"] = "sonobuoy"
	labels["tier"] = "analysis"
	labels["sonobuoy-run"] = p.GetSessionID()
	labels["sonobuoy-plugin"] = p.GetName()

	return labels
}
