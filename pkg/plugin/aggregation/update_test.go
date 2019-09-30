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
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"

	"github.com/kylelemons/godebug/pretty"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestCreateUpdater(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd"},
		{NodeName: "node2", ResultType: "systemd"},
		{NodeName: "", ResultType: "e2e"},
	}

	updater := newUpdater(
		expected,
		"sonobuoy-test",
		nil,
	)

	if err := updater.Receive(&PluginStatus{
		Status: FailedStatus,
		Node:   "node1",
		Plugin: "systemd",
	}); err != nil {
		t.Errorf("unexpected error receiving update %v", err)
	}

	if updater.status.Status != FailedStatus {
		t.Errorf("expected status to be failed, got %v", updater.status.Status)
	}
}

func TestGetAggregatorPod(t *testing.T) {
	createPodWithRunLabel := func(name string) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"run": "sonobuoy-master"},
			},
		}
	}

	testPods := []corev1.Pod{
		createPodWithRunLabel("sonobuoy-run-pod-1"),
		createPodWithRunLabel("sonobuoy-run-pod-2"),
	}

	checkNoError := func(err error) error {
		if err != nil {
			return errors.Wrap(err, "expected no error")
		}
		return nil
	}

	checkErrorFromServer := func(err error) error {
		if err == nil {
			return errors.New("expected error but got nil")
		}
		return nil
	}

	checkNoPodWithLabelError := func(err error) error {
		if _, ok := err.(NoPodWithLabelError); !ok {
			return errors.Wrap(err, "expected error to have type NoPodWithLabelError")
		}
		return nil
	}

	testCases := []struct {
		desc          string
		podsOnServer  corev1.PodList
		errFromServer error
		checkError    func(err error) error
		expectedPod   *corev1.Pod
	}{
		{
			desc:          "Error retrieving pods from server results in no pod and an error being returned",
			podsOnServer:  corev1.PodList{},
			errFromServer: errors.New("could not retrieve pods"),
			checkError:    checkErrorFromServer,
			expectedPod:   nil,
		},
		{
			desc:         "No pods results in no pod and no error",
			podsOnServer: corev1.PodList{},
			checkError:   checkNoPodWithLabelError,
			expectedPod:  nil,
		},
		{
			desc:         "Only one pod results in that pod being returned",
			podsOnServer: corev1.PodList{Items: []corev1.Pod{testPods[0]}},
			checkError:   checkNoError,
			expectedPod:  &testPods[0],
		},
		{
			desc:         "More that one pod results in the first pod being returned",
			podsOnServer: corev1.PodList{Items: testPods},
			checkError:   checkNoError,
			expectedPod:  &testPods[0],
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret kuberuntime.Object, err error) {
				return true, &tc.podsOnServer, tc.errFromServer
			})

			pod, err := GetAggregatorPod(fclient, "sonobuoy")
			if checkErr := tc.checkError(err); checkErr != nil {
				t.Errorf("error check failed: %v", checkErr)
			}
			if tc.expectedPod == nil {
				if pod != nil {
					t.Errorf("Expected no pod to be found but found pod %q", pod.GetName())
				}
			} else {
				if pod.GetName() != tc.expectedPod.GetName() {
					t.Errorf("Incorrect pod returned, expected %q but got %q", tc.expectedPod.GetName(), pod.GetName())
				}
			}

		})
	}
}

func TestGetAggregatorPodName(t *testing.T) {
	createPodWithRunLabel := func(name string) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"run": "sonobuoy-master"},
			},
		}
	}

	testCases := []struct {
		desc            string
		podsOnServer    corev1.PodList
		errFromServer   error
		expectedPodName string
	}{
		{
			desc:            "Error retrieving pods from server results in no pod name and an error being returned",
			podsOnServer:    corev1.PodList{},
			errFromServer:   errors.New("could not retrieve pods"),
			expectedPodName: "",
		},
		{
			desc:            "No pods results in default pod name being used",
			podsOnServer:    corev1.PodList{},
			errFromServer:   nil,
			expectedPodName: "sonobuoy",
		},
		{
			desc: "A returned pod results in that pod name being used",
			podsOnServer: corev1.PodList{
				Items: []corev1.Pod{
					createPodWithRunLabel("sonobuoy-run-pod"),
				},
			},
			errFromServer:   nil,
			expectedPodName: "sonobuoy-run-pod",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret kuberuntime.Object, err error) {
				return true, &tc.podsOnServer, tc.errFromServer
			})

			podName, err := GetAggregatorPodName(fclient, "sonobuoy")
			if tc.errFromServer == nil && err != nil {
				t.Errorf("Unexpected error returned, expected nil but got %q", err)
			}
			if tc.errFromServer != nil && err == nil {
				t.Errorf("Error not returned, expected %q but was nil", tc.errFromServer)
			}
			if podName != tc.expectedPodName {
				t.Errorf("Incorrect pod name returned, expected %q but got %q", tc.expectedPodName, podName)
			}
		})
	}
}

func TestReceive(t *testing.T) {
	expectedResults := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "type1"},
		{NodeName: "node2", ResultType: "type1"},
		{NodeName: "global", ResultType: "type2"},
	}
	tcs := []struct {
		desc     string
		results  map[string]*plugin.Result
		updates  map[string]*plugin.ProgressUpdate
		expected *Status
	}{
		{
			desc: "No results or updates",
			expected: &Status{
				Plugins: []PluginStatus{
					{Node: "node1", Plugin: "type1", Status: "running"},
					{Node: "node2", Plugin: "type1", Status: "running"},
					{Node: "global", Plugin: "type2", Status: "running"},
				},
				Status: "running",
			},
		}, {
			desc: "Status updates without results",
			expected: &Status{
				Plugins: []PluginStatus{
					{
						Node: "node1", Plugin: "type1", Status: "running",
						Progress: &plugin.ProgressUpdate{
							PluginName: "type1", Node: "node1", Message: "new status",
							Failures: []string{"f1", "f2"},
							Errors:   []string{"err1", "err2"},
						},
					},
					{Node: "node2", Plugin: "type1", Status: "running"},
					{Node: "global", Plugin: "type2", Status: "running"},
				},
				Status: "running",
			},
			updates: map[string]*plugin.ProgressUpdate{
				"type1/node1": {
					PluginName: "type1", Node: "node1", Message: "new status",
					Failures: []string{"f1", "f2"},
					Errors:   []string{"err1", "err2"},
				},
			},
		}, {
			desc: "Results (passing and failing) without status updates",
			expected: &Status{
				Plugins: []PluginStatus{
					{
						Node: "node1", Plugin: "type1", Status: "complete",
					},
					{Node: "node2", Plugin: "type1", Status: "failed"},
					{Node: "global", Plugin: "type2", Status: "running"},
				},
				Status: "failed",
			},
			results: map[string]*plugin.Result{
				"type1/node1": {NodeName: "node1", ResultType: "type1"},
				"type1/node2": {NodeName: "node2", ResultType: "type1", Error: "errmsg"},
			},
		}, {
			desc: "Overlapping status updates and results",
			expected: &Status{
				Plugins: []PluginStatus{
					{
						Node: "node1", Plugin: "type1", Status: "complete",
						Progress: &plugin.ProgressUpdate{
							PluginName: "type1", Node: "node1", Message: "new status",
						},
					},
					{Node: "node2", Plugin: "type1", Status: "running"},
					{Node: "global", Plugin: "type2", Status: "running"},
				},
				Status: "running",
			},
			results: map[string]*plugin.Result{
				"type1/node1": {NodeName: "node1", ResultType: "type1"},
			},
			updates: map[string]*plugin.ProgressUpdate{
				"type1/node1": {PluginName: "type1", Node: "node1", Message: "new status"},
			},
		}, {
			desc: "Non-overlapping updates and results",
			expected: &Status{
				Plugins: []PluginStatus{
					{Node: "node1", Plugin: "type1", Status: "complete"},
					{Node: "node2", Plugin: "type1", Status: "running"},
					{
						Node: "global", Plugin: "type2", Status: "running",
						Progress: &plugin.ProgressUpdate{
							PluginName: "type2", Node: "global", Message: "new status",
						},
					},
				},
				Status: "running",
			},
			results: map[string]*plugin.Result{
				"type1/node1": {NodeName: "node1", ResultType: "type1"},
			},
			updates: map[string]*plugin.ProgressUpdate{
				"type2/global": {PluginName: "type2", Node: "global", Message: "new status"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			u := newUpdater(expectedResults, "testns", nil)
			u.ReceiveAll(tc.results, tc.updates)
			if diff := pretty.Compare(tc.expected, u.status); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
