/*
Copyright the Sonobuoy contributors 2019

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

package client

import (
	"os"
	"strings"
	"encoding/json"
	"reflect"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestRunInvalidConfig(t *testing.T) {
	testcases := []struct {
		desc             string
		config           *RunConfig
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			desc:             "Passing a nil config results in an error",
			config:           nil,
			expectedErrorMsg: "nil RunConfig provided",
		},
		{
			desc: "Passing a file takes priority over config flags",
			config: &RunConfig{
				GenFile: "foo.yaml",
			},
			expectedErrorMsg: "no such file or directory",
		},
		// NOTE: Running non-failing logic here is not supported at this time due to the fact
		// that it tries to actually start executing logic with the dynamic client which
		// is nil.
	}

	c, err := NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			expectedError := len(tc.expectedErrorMsg) > 0
			err = c.Run(tc.config)
			if !expectedError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if expectedError {
				if err == nil {
					t.Errorf("Expected provided config to be invalid but got no error")
				} else if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tc.expectedErrorMsg, err.Error())
				}
			}
		})
	}
}

func TestPrintPodStatus(t *testing.T) {
	expected := []string{
		"Status: Pending, Reason: ContainersNotReady, containers with unready status: [kube-sonobuoy]\nDetails of containers that are not ready:\nkube-sonobuoy: waiting: ImagePullBackOff, Back-off pulling image \"schnake/sonobuoy:987\"\n",
		"Status: Pending, Reason: ContainersNotReady, containers with unready status: [kube-sonobuoy]\nDetails of containers that are not ready:\nkube-sonobuoy: waiting: ContainerCreating\n",
		"Status: Pending, Reason: Unschedulable, 0/1 nodes are available: 1 node(s) had taint {node.kubernetes.io/not-ready: }, that the pod didn't tolerate.",
		"Status: Running",
		"Status: Running",
	}
	fname := "testdata/PrintPodStatus.json"
	reader, err := os.Open(fname)
	if err != nil {
		t.Errorf("Unable to open test file %s: %s", fname, err)
	}
	defer reader.Close()

	podList := &corev1.PodList{}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(podList); err != nil {
		t.Errorf("Unable to decode to json the test file %s: %s", fname, err)
	}
	got := make([]string, len(podList.Items))
	if len(podList.Items) != len(expected) {
		t.Errorf("Expected %d pods, got %d instead", len(expected), len(podList.Items))
	}
	for idx, pod := range podList.Items {
		got[idx] = getPodStatus(pod)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %+v, got %+v instead", expected, got)
	}
}
