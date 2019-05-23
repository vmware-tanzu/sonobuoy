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
	"crypto/sha1"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/heptio/sonobuoy/pkg/backplane/ca"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"

	corev1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	expectedImageName = "gcr.io/heptio-image/sonobuoy:master"
	expectedNamespace = "test-namespace"
)

func TestFillTemplate(t *testing.T) {
	testJob := NewPlugin(plugin.Definition{
		Name:       "test-job",
		ResultType: "test-job-result",
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
	clientCert, err := auth.ClientKeyPair("test-job")
	if err != nil {
		t.Fatalf("couldn't make client certificate %v", err)
	}

	var pod corev1.Pod
	b, err := testJob.FillTemplate("", clientCert)
	if err != nil {
		t.Fatalf("Failed to fill template: %v", err)
	}

	t.Logf("%s", b)

	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), b, &pod); err != nil {
		t.Fatalf("Failed to decode template to pod: %v", err)
	}

	expectedName := fmt.Sprintf("sonobuoy-test-job-job-%v", testJob.SessionID)
	if pod.Name != expectedName {
		t.Errorf("Expected pod name %v, got %v", expectedName, pod.Name)
	}

	if pod.Namespace != expectedNamespace {
		t.Errorf("Expected pod namespace %v, got %v", expectedNamespace, pod.Namespace)
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
