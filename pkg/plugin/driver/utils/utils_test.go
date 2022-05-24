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

package utils

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodFailing(t *testing.T) {
	// fromGoodPod is a helper function to simplify the specification of test cases.
	fromGoodPod := func(f func(*corev1.Pod) *corev1.Pod) *corev1.Pod {
		goodPod := &corev1.Pod{}
		return f(goodPod)
	}

	testCases := []struct {
		desc          string
		pod           *corev1.Pod
		expectFailing bool
		expectMsg     string
	}{
		{
			desc: "OK pod not failing",
			pod:  fromGoodPod(func(p *corev1.Pod) *corev1.Pod { return p }),
		}, {
			desc:          "Terminated container reported failing if old enough",
			expectFailing: true,
			expectMsg:     "Container container1 is in a terminated state (exit code 1) due to reason: myReason: myMsg",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "container1",
						State: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:     "myReason",
								Message:    "myMsg",
								ExitCode:   1,
								FinishedAt: metav1.Time{Time: time.Now().Add(-3 * terminatedContainerWindow)},
							},
						}},
				}
				return p
			}),
		}, {
			desc:          "Terminated container not reported failing if too recent",
			expectFailing: false,
			expectMsg:     "",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							Reason:     "reason",
							Message:    "msg",
							ExitCode:   1,
							FinishedAt: metav1.Time{Time: time.Now().Add(terminatedContainerWindow / -2)},
						},
					}},
				}
				return p
			}),
		}, {
			desc:          "Does not panic if StartTime is nil when container status is waiting",
			expectFailing: false,
			expectMsg:     "",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.StartTime = nil
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					}},
				}
				return p
			}),
		}, {
			desc:          "ImagePullBackOff is not considered a failure if elapsed time within wait window",
			expectFailing: false,
			expectMsg:     "",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.StartTime = &metav1.Time{Time: time.Now().Add(maxWaitForImageTime / -2)}
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					}},
				}
				return p
			}),
		}, {
			desc:          "ImagePullBackOff is considered a failure if elapsed time greater than wait window",
			expectFailing: true,
			expectMsg:     "Failed to pull image random-image for container error-container within 5m0s. Container is in state ImagePullBackOff",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.StartTime = &metav1.Time{Time: time.Now().Add(-maxWaitForImageTime)}
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name:  "error-container",
						Image: "random-image",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ImagePullBackOff",
							},
						}},
				}
				return p
			}),
		}, {
			desc:          "ErrImagePull is considered a failure if elapsed time greater than wait window",
			expectFailing: true,
			expectMsg:     "Failed to pull image random-image for container error-container within 5m0s. Container is in state ErrImagePull",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.StartTime = &metav1.Time{Time: time.Now().Add(-maxWaitForImageTime)}
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name:  "error-container",
						Image: "random-image",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ErrImagePull",
							},
						}},
				}
				return p
			}),
		}, {
			desc:          "Other wait reason not considered a failure if elapsed time greater than wait window",
			expectFailing: false,
			expectMsg:     "",
			pod: fromGoodPod(func(p *corev1.Pod) *corev1.Pod {
				p.Status.StartTime = &metav1.Time{Time: time.Now().Add(-maxWaitForImageTime)}
				p.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ContainerCreating",
							},
						}},
				}
				return p
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			failing, msg := IsPodFailing(tc.pod)
			if failing != tc.expectFailing {
				t.Errorf("Expected %v but got %v", tc.expectFailing, failing)
			}
			if msg != tc.expectMsg {
				t.Errorf("Expected %q but got %q", tc.expectMsg, msg)
			}
		})
	}
}
