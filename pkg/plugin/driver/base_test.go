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

package driver

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/vmware-tanzu/sonobuoy/pkg/backplane/ca"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
)

func TestMakeTLSSecret(t *testing.T) {
	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("unexpected error %v making authority", err)
	}
	expectedNamespace := "test-namespace"
	expectedName := "test-name"
	sessionID := "aaaaaa11111"

	cert, err := auth.ClientKeyPair("")
	if err != nil {
		t.Fatalf("unexpected error %v making client pair", err)
	}

	driver := &Base{
		Namespace: expectedNamespace,
		Definition: manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{PluginName: expectedName},
		},
		SessionID: sessionID,
	}

	ownerPodName := "ownerPodName"
	var ownerPodUID types.UID = "ownerPodUID"
	ownerPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: ownerPodName,
			UID:  ownerPodUID,
		},
	}

	secret, err := driver.MakeTLSSecret(cert, ownerPod)
	if err != nil {
		t.Fatalf("unexpected error %v making TLS Secret", err)
	}

	if secret.ObjectMeta.Name != driver.GetSecretName() {
		t.Errorf("expected name %v, got %v", expectedName, secret.ObjectMeta.Name)
	}
	if secret.ObjectMeta.Namespace != expectedNamespace {
		t.Errorf("expected namespace %v, got %v", expectedNamespace, secret.ObjectMeta.Namespace)
	}

	if len(secret.ObjectMeta.OwnerReferences) != 1 {
		t.Errorf("expected secret to have 1 owner reference, got %v", len(secret.ObjectMeta.OwnerReferences))
	} else {
		ownerReference := secret.ObjectMeta.OwnerReferences[0]
		if ownerReference.Name != ownerPodName {
			t.Errorf("expected owner reference to have name %v, got %v", ownerPodName, ownerReference.Name)
		}
		if ownerReference.UID != ownerPodUID {
			t.Errorf("expected owner reference to have UID %v, got %v", ownerPodUID, ownerReference.UID)
		}
	}

	expectedKeyBytes, err := x509.MarshalECPrivateKey(cert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		t.Fatalf("unexpected error %v marshalling EC private key", err)
	}
	keyPEM, _ := pem.Decode(secret.Data["tls.key"])
	if keyPEM == nil {
		t.Fatal("couldn't decode tls.key")
	}

	if sha1.Sum(expectedKeyBytes) != sha1.Sum(keyPEM.Bytes) {
		t.Error("key fingerprint didn't match")
	}

	certPEM, _ := pem.Decode(secret.Data["tls.crt"])
	if certPEM == nil {
		t.Fatal("couldn't decode tls.crt")
	}

	if sha1.Sum(cert.Leaf.Raw) != sha1.Sum(certPEM.Bytes) {
		t.Error("cert fingerprint didn't match")
	}
}

func TestSkipCleanup(t *testing.T) {
	b := &Base{
		Definition: manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{SkipCleanup: false},
		},
	}

	if b.SkipCleanup() {
		t.Error("Expected SkipCleanup to be false but was true")
	}

	b.Definition.SonobuoyConfig.SkipCleanup = true

	if !b.SkipCleanup() {
		t.Error("Expected SkipCleanup to be true but was false")
	}
}

func envContains(env []v1.EnvVar, item v1.EnvVar) bool {
	for _, v := range env {
		if v == item {
			return true
		}
	}
	return false
}

func TestCreateWorkerContainerDefinition(t *testing.T) {
	aggregatorURL := "aggregatorUrl"
	cert := &tls.Certificate{Certificate: [][]byte{}}
	command := []string{"sonobuoy"}
	args := []string{"worker", "--global"}

	b := &Base{
		Definition: manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{
				PluginName: "test-plugin",
			},
		},
		SonobuoyImage:   "sonobuoy:v1",
		ImagePullPolicy: "Never",
		SessionID:       "sessionID",
	}

	wc := b.CreateWorkerContainerDefintion(aggregatorURL, cert, command, args, "")

	checkFields := func(container v1.Container) error {
		if container.Name != "sonobuoy-worker" {
			return fmt.Errorf("expected worker container name to be %q, but got %q", "sonobuoy-worker", container.Name)
		}

		if container.Image != "sonobuoy:v1" {
			return fmt.Errorf("expected worker container image to be %q, but got %q", "sonobuoy:v1", container.Image)
		}

		if container.ImagePullPolicy != v1.PullNever {
			return fmt.Errorf("expected worker container pull policy to be %q, but got %q", v1.PullNever, container.ImagePullPolicy)
		}

		for i, c := range container.Command {
			if c != command[i] {
				return fmt.Errorf("expected command item %v to be %q, got %q", i, command[i], c)
			}
		}

		for i, arg := range container.Args {
			if arg != args[i] {
				return fmt.Errorf("expected args item %v to be %q, got %q", i, args[i], arg)
			}
		}
		return nil
	}

	checkEnvironment := func(container v1.Container) error {
		expectedEnvVars := []v1.EnvVar{
			{
				Name:  "RESULTS_DIR",
				Value: "/tmp/results",
			},
			{
				Name:  "RESULT_TYPE",
				Value: b.GetName(),
			},
			{
				Name:  "AGGREGATOR_URL",
				Value: aggregatorURL,
			},
			{
				Name:  "CA_CERT",
				Value: "",
			},
		}
		for _, e := range expectedEnvVars {
			if !envContains(container.Env, e) {
				return fmt.Errorf("expected container environment to contain %q", e)
			}
		}
		for _, e := range container.Env {
			switch e.Name {
			case "NODE_NAME":
				expected := "spec.nodeName"
				got := e.ValueFrom.FieldRef.FieldPath
				if got != expected {
					return fmt.Errorf("expected NODE_NAME to have FieldRef value %q, but got %q", expected, got)
				}
			case "CLIENT_CERT":
				expectedKey := "tls.crt"
				gotKey := e.ValueFrom.SecretKeyRef.Key
				if gotKey != expectedKey {
					return fmt.Errorf("expected CLIENT_CERT to have SecretKeyRef key %q, but got %q", expectedKey, gotKey)
				}
				expectedName := "sonobuoy-plugin-test-plugin-sessionID"
				gotName := e.ValueFrom.SecretKeyRef.LocalObjectReference.Name
				if gotKey != expectedKey {
					return fmt.Errorf("expected CLIENT_CERT to have SecretKeyRef name %q, but got %q", expectedName, gotName)
				}
			case "CLIENT_KEY":
				expectedKey := "tls.key"
				gotKey := e.ValueFrom.SecretKeyRef.Key
				if gotKey != expectedKey {
					return fmt.Errorf("expected CLIENT_CERT to have SecretKeyRef key %q, but got %q", expectedKey, gotKey)
				}
				expectedName := "sonobuoy-plugin-test-plugin-sessionID"
				gotName := e.ValueFrom.SecretKeyRef.LocalObjectReference.Name
				if gotKey != expectedKey {
					return fmt.Errorf("expected CLIENT_CERT to have SecretKeyRef name %q, but got %q", expectedName, gotName)
				}
			}
		}
		return nil
	}

	checkVolumes := func(container v1.Container) error {
		expectedVolume := v1.VolumeMount{
			Name:      "results",
			ReadOnly:  false,
			MountPath: "/tmp/results",
		}
		if len(container.VolumeMounts) != 1 {
			return fmt.Errorf("expected the container to have one volume mount, got %v", len(container.VolumeMounts))
		}
		if container.VolumeMounts[0] != expectedVolume {
			return fmt.Errorf("expected volume mount to equal %v, got %v", expectedVolume, container.VolumeMounts[0])
		}
		return nil

	}

	testCases := []struct {
		desc  string
		check func(v1.Container) error
	}{
		{
			"Fields set correctly",
			checkFields,
		},
		{
			"Environment set correctly",
			checkEnvironment,
		},
		{
			"Volume mounts set correctly",
			checkVolumes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if err := tc.check(wc); err != nil {
				t.Error(err)
			}
		})
	}
}
