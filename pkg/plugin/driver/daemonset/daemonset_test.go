package daemonset

import (
	"crypto/sha1"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/heptio/sonobuoy/pkg/backplane/ca"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestFillTemplate(t *testing.T) {
	testDaemonSet := NewPlugin(plugin.Definition{
		Name:       "test-plugin",
		ResultType: "test-plugin-result",
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
	clientCert, err := auth.ClientKeyPair("test-job")
	if err != nil {
		t.Fatalf("couldn't make client certificate %v", err)
	}

	var daemonSet v1beta1.DaemonSet
	b, err := testDaemonSet.FillTemplate("", clientCert)
	if err != nil {
		t.Fatalf("Failed to fill template: %v", err)
	}

	t.Logf("%s", b)

	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), b, &daemonSet); err != nil {
		t.Fatalf("Failed to decode template to daemonSet: %v", err)
	}

	expectedName := fmt.Sprintf("sonobuoy-test-plugin-daemon-set-%v", testDaemonSet.SessionID)
	if daemonSet.Name != expectedName {
		t.Errorf("Expected daemonSet name %v, got %v", expectedName, daemonSet.Name)
	}

	expectedNamespace := "test-namespace"
	if daemonSet.Namespace != expectedNamespace {
		t.Errorf("Expected daemonSet namespace %v, got %v", expectedNamespace, daemonSet.Namespace)
	}

	containers := daemonSet.Spec.Template.Spec.Containers

	expectedContainers := 2
	if len(containers) != expectedContainers {
		t.Errorf("Expected to have %v containers, got %v", expectedContainers, len(containers))
	} else {
		// Don't segfault if the count is incorrect
		expectedProducerName := "producer-container"
		if containers[0].Name != expectedProducerName {
			t.Errorf("Expected producer daemonSet to have name %v, got %v", expectedProducerName, containers[0].Name)
		}
	}

	env := make(map[string]string)
	for _, envVar := range daemonSet.Spec.Template.Spec.Containers[1].Env {
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
