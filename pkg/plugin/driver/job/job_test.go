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

func TestFillTemplate(t *testing.T) {
	testJob := NewPlugin(plugin.Definition{
		Name:       "test-job",
		ResultType: "test-job-result",
		Spec: manifest.Container{
			Container: corev1.Container{
				Name: "producer-container",
			},
		},
	}, "test-namespace")

	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("couldn't make CA Authority %v", err)
	}
	clientCert, err := auth.ClientKey("test-job")
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

	expectedNamespace := "test-namespace"
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
			t.Errorf("Expected producer pod to have name %v, got %v", expectedProducerName, pod.Spec.Containers[0].Name)
		}
	}

	env := make(map[string]string)
	for _, envVar := range pod.Spec.Containers[1].Env {
		env[envVar.Name] = envVar.Value
	}

	for _, testCase := range []struct {
		EnvVar      string
		Fingerprint [sha1.Size]byte
	}{
		{
			EnvVar:      "CA_CERT",
			Fingerprint: sha1.Sum(auth.CACert().Raw),
		},
		{
			EnvVar:      "CLIENT_CERT",
			Fingerprint: sha1.Sum(clientCert.Leaf.Raw),
		},
	} {
		caCertPEM, ok := env[testCase.EnvVar]
		if !ok {
			t.Fatalf("no env var %v", testCase.EnvVar)
		}

		caCertBlock, _ := pem.Decode([]byte(caCertPEM))
		if caCertBlock == nil {
			t.Fatal("No PEM block found.")
		}

		caCertFingerprint := sha1.Sum(caCertBlock.Bytes)

		if caCertFingerprint != testCase.Fingerprint {
			t.Errorf("%v fingerprint didn't match", testCase.EnvVar)
		}
	}
}
