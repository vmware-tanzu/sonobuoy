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

package job

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/backplane/ca"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	sonotime "github.com/vmware-tanzu/sonobuoy/pkg/time/timetest"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	expectedImageName = "gcr.io/heptio-image/sonobuoy:master"
	expectedNamespace = "test-namespace"
)

func createClientCertificate(name string) (*tls.Certificate, error) {
	auth, err := ca.NewAuthority()
	if err != nil {
		return nil, fmt.Errorf("couldn't make CA Authority %v", err)
	}

	clientCert, err := auth.ClientKeyPair("test-job")
	if err != nil {
		return nil, fmt.Errorf("couldn't make client certificate %v", err)
	}
	return clientCert, nil
}

func TestCreatePodDefinition(t *testing.T) {
	pluginName := "test-job"
	testPlugin := NewPlugin(
		manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{
				PluginName: pluginName,
			},
			Spec: manifest.Container{
				Container: corev1.Container{
					Name: "producer-container",
				},
			},
			ExtraVolumes: []manifest.Volume{
				{
					Volume: corev1.Volume{
						Name: "test1",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/test",
							},
						},
					},
				},
				{
					Volume: corev1.Volume{
						Name: "test2",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/test2",
							},
						},
					},
				},
			},
		}, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{"key1": "val1", "key2": "val2"})

	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("couldn't make CA Authority %v", err)
	}

	clientCert, err := auth.ClientKeyPair(pluginName)
	if err != nil {
		t.Fatalf("couldn't make client certificate %v", err)
	}

	pod := testPlugin.createPodDefinition("", clientCert, &corev1.Pod{}, "")

	expectedName := fmt.Sprintf("sonobuoy-%v-job-%v", pluginName, testPlugin.SessionID)
	if pod.Name != expectedName {
		t.Errorf("Expected pod name %v, got %v", expectedName, pod.Name)
	}

	if pod.Namespace != expectedNamespace {
		t.Errorf("Expected pod namespace %v, got %v", expectedNamespace, pod.Namespace)
	}

	pluginLabel := "sonobuoy-plugin"
	if pod.Labels[pluginLabel] != pluginName {
		t.Errorf("Expected pod to have label %q with value %q, but had value %q", pluginLabel, pluginName, pod.Labels[pluginLabel])
	}

	expectedContainers := 2
	if len(pod.Spec.Containers) != expectedContainers {
		t.Errorf("Expected to have %v containers, got %v", expectedContainers, len(pod.Spec.Containers))
	} else {
		// Don't segfault if the count is incorrect
		expectedProducerName := "producer-container"
		if pod.Spec.Containers[0].Name != expectedProducerName {
			t.Errorf(
				"Expected producer pod to have name %v, got %v",
				expectedProducerName,
				pod.Spec.Containers[0].Name,
			)
		}

		if pod.Spec.Containers[1].Image != expectedImageName {
			t.Errorf(
				"Expected consumer pod to have image %v, got %v",
				expectedImageName,
				pod.Spec.Containers[1].Image,
			)
		}
	}

	env := make(map[string]string)
	for _, envVar := range pod.Spec.Containers[1].Env {
		env[envVar.Name] = envVar.Value
	}

	caCertPEM, ok := env["CA_CERT"]
	if !ok {
		t.Fatal("no env var CA_CERT")
	}
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		t.Fatal("No PEM block found.")
	}

	caCertFingerprint := sha1.Sum(caCertBlock.Bytes)

	if caCertFingerprint != sha1.Sum(auth.CACert().Raw) {
		t.Errorf("CA_CERT fingerprint didn't match")
	}

	if len(pod.Spec.Volumes) != 3 {
		t.Errorf("Expected 2 volumes on pod, got %d", len(pod.Spec.Volumes))
	}

	if len(pod.Spec.ImagePullSecrets) != 1 {
		t.Errorf("Expected 1 imagePullSecrets but got %v", len(pod.Spec.ImagePullSecrets))
	} else {
		if pod.Spec.ImagePullSecrets[0].Name != "image-pull-secret" {
			t.Errorf("Expected imagePullSecrets with name %v but got %v", "image-pull-secret", pod.Spec.ImagePullSecrets)
		}
	}

	if pod.Annotations["key1"] != "val1" ||
		pod.Annotations["key2"] != "val2" {
		t.Errorf("Expected annotations key1:val1 and key2:val2 to be set, but got %v", pod.Annotations)
	}
}

func TestCreatePodDefinitionUsesDefaultPodSpec(t *testing.T) {
	m := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName: "test-job",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name: "producer-container",
			},
		},
	}
	testPlugin := NewPlugin(m, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	pod := testPlugin.createPodDefinition("", clientCert, &corev1.Pod{}, "")

	expectedServiceAccount := "sonobuoy-serviceaccount"
	if pod.Spec.ServiceAccountName != expectedServiceAccount {
		t.Errorf("expected pod spec to have default service account name %q, got %q", expectedServiceAccount, pod.Spec.ServiceAccountName)

	}

	// Check something specific to the job default pod spec
	expectedNumTolerations := 3
	actualNumTolerations := len(pod.Spec.Tolerations)
	if actualNumTolerations != expectedNumTolerations {
		t.Errorf("expected pod spec to %v tolerations, got %v", expectedNumTolerations, actualNumTolerations)
	}

	if len(pod.Spec.NodeSelector) != 1 || pod.Spec.NodeSelector["kubernetes.io/os"] != "linux" {
		t.Errorf("Expected node selector kubernetes.io/os:linux but got %v", pod.Spec.NodeSelector)
	}
}

func TestCreatePodDefinitionUsesProvidedPodSpec(t *testing.T) {
	expectedServiceAccountName := "test-serviceaccount"
	m := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName: "test-job",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name: "producer-container",
			},
		},
		PodSpec: &manifest.PodSpec{
			PodSpec: corev1.PodSpec{
				ServiceAccountName: expectedServiceAccountName,
				NodeSelector:       map[string]string{"foo": "bar"},
			},
		},
	}
	testPlugin := NewPlugin(m, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	pod := testPlugin.createPodDefinition("", clientCert, &corev1.Pod{}, "")

	if pod.Spec.ServiceAccountName != expectedServiceAccountName {
		t.Errorf("expected pod spec to have provided service account name %q, got %q", expectedServiceAccountName, pod.Spec.ServiceAccountName)
	}

	if len(pod.Spec.NodeSelector) != 1 || pod.Spec.NodeSelector["foo"] != "bar" {
		t.Errorf("Expected node selector foo:bar but got %v", pod.Spec.NodeSelector)
	}
}

func TestCreatePodDefinitionAddsToExistingResourcesInPodSpec(t *testing.T) {
	m := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName: "test-job",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name: "producer-container",
			},
		},
		ExtraVolumes: []manifest.Volume{{Volume: corev1.Volume{Name: "test1"}}},
		PodSpec: &manifest.PodSpec{
			PodSpec: corev1.PodSpec{
				Containers:       []corev1.Container{{}},
				ImagePullSecrets: []corev1.LocalObjectReference{{}},
				Volumes:          []corev1.Volume{{}},
			},
		},
	}
	testPlugin := NewPlugin(m, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	pod := testPlugin.createPodDefinition("", clientCert, &corev1.Pod{}, "")

	// Existing container in pod spec, plus 2 added by Sonobuoy
	expectedNumContainers := 3
	actualNumContainers := len(pod.Spec.Containers)
	if expectedNumContainers != actualNumContainers {
		t.Errorf("expected pod spec to have %v containers, got %v", expectedNumContainers, actualNumContainers)
	}

	// Existing image pull secret in pod spec, plus 1 added by Sonobuoy
	expectedNumImagePullSecrets := 2
	actualNumImagePullSecrets := len(pod.Spec.ImagePullSecrets)
	if expectedNumImagePullSecrets != actualNumImagePullSecrets {
		t.Errorf("expected pod spec to have %v image pull secrets, got %v", expectedNumImagePullSecrets, actualNumImagePullSecrets)
	}

	// Existing volume in pod spec, plus 1 extra volume, plus 1 added by Sonobuoy
	expectedNumVolumes := 3
	actualNumVolumes := len(pod.Spec.Volumes)
	if expectedNumVolumes != actualNumVolumes {
		t.Errorf("expected pod spec to have %v volumes, got %v", expectedNumVolumes, actualNumVolumes)
	}
}

func TestCreatePodDefinitionSetsOwnerReference(t *testing.T) {
	m := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName: "test-job",
		},
		Spec: manifest.Container{Container: corev1.Container{}},
	}
	testPlugin := NewPlugin(m, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	aggregatorPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sonobuoy-aggregator",
			UID:  "123456-abcdef",
		},
	}

	pod := testPlugin.createPodDefinition("", clientCert, &aggregatorPod, "")
	ownerReferences := pod.ObjectMeta.OwnerReferences

	if len(ownerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference, got %v", len(ownerReferences))
	}

	testCases := []struct {
		field string
		want  string
		got   string
	}{
		{
			field: "APIVersion",
			want:  "v1",
			got:   ownerReferences[0].APIVersion,
		},
		{
			field: "Kind",
			want:  "Pod",
			got:   ownerReferences[0].Kind,
		},
		{
			field: "Name",
			want:  aggregatorPod.ObjectMeta.Name,
			got:   ownerReferences[0].Name,
		},
		{
			field: "UID",
			want:  string(aggregatorPod.ObjectMeta.UID),
			got:   string(ownerReferences[0].UID),
		},
	}

	for _, tc := range testCases {
		if tc.got != tc.want {
			t.Errorf("Expected ownerReference %v to be %q, got %q", tc.field, tc.want, tc.got)
		}
	}
}

func TestMonitorOnce(t *testing.T) {
	// Note: the pods must be marked with the label "sonobuoy-run" or else our labelSelector
	// logic will filter them out even though the fake server returns them.

	testCases := []struct {
		desc       string
		expectDone bool

		// Ensure we are getting the err result we expect; lots of ways to get errors
		// that may not be clear.
		expectErrResultMsg string

		job           *Plugin
		podOnServer   *corev1.Pod
		errFromServer error
	}{
		{
			desc:       "Cleaned up indicates exit without error",
			expectDone: true,
			job:        &Plugin{driver.Base{CleanedUp: true}},
		}, {
			desc:          "Server error results in error being ignored",
			job:           &Plugin{},
			podOnServer:   nil,
			errFromServer: errors.New("forcedError"),
			expectDone:    false,
		}, {
			desc: "Failing pod results in error",
			job:  &Plugin{},
			podOnServer: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{
						{Reason: "Unschedulable", Message: "conditionMsg"},
					},
				},
			},
			expectErrResultMsg: "Can't schedule pod: conditionMsg",
			expectDone:         true,
		}, {
			desc: "Healthy pod results in no error and continued monitoring",
			job:  &Plugin{},
			podOnServer: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, ret kuberuntime.Object, err error) {
				if tc.podOnServer == nil {
					return true, &corev1.PodList{}, tc.errFromServer
				}
				podList := &corev1.PodList{
					Items: []corev1.Pod{
						*(tc.podOnServer),
					},
				}
				return true, podList, tc.errFromServer
			})

			done, errResult := tc.job.monitorOnce(fclient, nil)
			if done != tc.expectDone {
				t.Errorf("Expected %v but got %v", tc.expectDone, done)
			}
			switch {
			case errResult != nil && tc.expectErrResultMsg == "":
				t.Errorf("Expected no error but got %v", errResult)
			case errResult == nil && tc.expectErrResultMsg != "":
				t.Errorf("Expected error %v but got nil", tc.expectErrResultMsg)
			case errResult != nil && tc.expectErrResultMsg != errResult.Error:
				t.Errorf("Expected error %q but got %q", tc.expectErrResultMsg, errResult.Error)
			}
		})
	}
}

func TestMonitor(t *testing.T) {
	// For these tests ensure sleeping is fast; choosing non-zero we know which
	// branch of select may be chosen first.
	sonotime.UseShortAfter()
	defer sonotime.ResetAfter()

	failingPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Reason: "Unschedulable"},
			},
		},
	}
	healthyPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
	}

	testCases := []struct {
		desc       string
		expectDone bool

		// Ensure we are getting the err result we expect; lots of ways to get errors
		// that may not be clear.
		expectErrResultMsg string

		podList               *corev1.PodList
		expectNumResults      int
		expectStillMonitoring bool
		cancelContext         bool
	}{
		{
			desc:                  "Errored pod should cause error on channel and exit",
			expectNumResults:      1,
			expectStillMonitoring: false,
			podList: &corev1.PodList{
				Items: []corev1.Pod{failingPod},
			},
		}, {
			desc:                  "Continues to poll with healthy pod",
			expectNumResults:      0,
			expectStillMonitoring: true,
			podList: &corev1.PodList{
				Items: []corev1.Pod{healthyPod},
			},
		}, {
			desc:                  "Can be cancelled via context",
			cancelContext:         true,
			expectNumResults:      0,
			expectStillMonitoring: false,
			podList: &corev1.PodList{
				Items: []corev1.Pod{healthyPod},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret kuberuntime.Object, err error) {
				return true, tc.podList, nil
			})

			p := &Plugin{}
			ctx, cancel := context.WithCancel(context.Background())
			ch := make(chan (*plugin.Result), 1)

			wasStillMonitoring := false
			if tc.cancelContext {
				cancel()
			} else {
				// Max timeout for test to unblock.
				go func() {
					time.Sleep(2 * time.Second)
					wasStillMonitoring = true
					cancel()
				}()
			}
			go p.Monitor(ctx, fclient, nil, ch)

			count := 0
			for range ch {
				count++
			}

			if count != tc.expectNumResults {
				t.Errorf("Expected %v results but found %v", tc.expectNumResults, count)
			}
			if wasStillMonitoring != tc.expectStillMonitoring {
				t.Errorf("Expected wasStillMonitoring %v but found %v", tc.expectStillMonitoring, wasStillMonitoring)
			}
		})
	}
}
