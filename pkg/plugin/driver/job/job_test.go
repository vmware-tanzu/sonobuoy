package job

import (
	"fmt"
	"testing"

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

	var pod corev1.Pod
	b, err := testJob.FillTemplate("")
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
}
