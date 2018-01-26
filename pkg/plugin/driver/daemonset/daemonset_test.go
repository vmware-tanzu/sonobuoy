package daemonset

import (
	"fmt"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestFillTemplate(t *testing.T) {
	testDaemonSet := NewPlugin(plugin.Definition{
		Name:       "test-plugin",
		ResultType: "test-plugin-result",
		Spec: corev1.Container{
			Name: "producer-container",
		},
	}, "test-namespace")

	var daemonSet v1beta1.DaemonSet
	b, err := testDaemonSet.FillTemplate("")
	if err != nil {
		t.Fatalf("Failed to fill template: %v", err)
	}

	t.Logf("%s", b.Bytes())

	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), b.Bytes(), &daemonSet); err != nil {
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
}
