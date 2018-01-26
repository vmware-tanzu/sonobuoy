package utils

import (
	"fmt"
	"testing"

	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
)

func TestContainerToYAML(t *testing.T) {

	var (
		expectedName  = "test-container"
		expectedImage = "gcr.io/heptio/test-image:master"
		expectedCmd   = []string{"echo", "Hello world!"}
	)
	container := &v1.Container{
		Name:    expectedName,
		Image:   expectedImage,
		Command: expectedCmd,
	}

	yamlDoc, err := ContainerToYAML(container)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlDoc), &parsed); err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if parsed["name"].(string) != expectedName {
		t.Errorf("expected name %v, got %v", expectedName, parsed["name"])
	}

	if parsed["image"].(string) != expectedImage {
		t.Errorf("expected image %v, got %v", expectedImage, parsed["image"])
	}

	// DeepEqual barfs on the []interface{}
	if fmt.Sprintf("%v", parsed["command"]) != fmt.Sprintf("%v", expectedCmd) {
		t.Errorf("expected command %v, got %v", expectedCmd, parsed["command"])
	}
}
