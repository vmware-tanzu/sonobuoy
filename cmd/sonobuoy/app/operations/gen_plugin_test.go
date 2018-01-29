package operations

import (
	"testing"

	"k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestLoadNonexistantPlugin(t *testing.T) {
	_, err := GeneratePluginManifest(GenPluginConfig{
		Paths:      []string{"./plugins.d"},
		PluginName: "non-existant-plugin",
	})

	if err.Error() != "expected 1 plugin, got 0" {
		t.Errorf("unexpected or no error %q", err)
	}
}

func TestLoadRealPlugin(t *testing.T) {
	bytes, err := GeneratePluginManifest(GenPluginConfig{
		// Tests are executed with cwd set to their containing directory
		Paths:      []string{"../../../../plugins.d"},
		PluginName: "e2e",
	})

	if err != nil {
		t.Fatalf("%v", err)
	}

	var job v1.Pod

	if err = kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), bytes, &job); err != nil {
		t.Errorf("failed to decode job: %v", err)
	}
}
