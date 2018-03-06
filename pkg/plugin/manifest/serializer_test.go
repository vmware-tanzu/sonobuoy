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

package manifest

import (
	"fmt"
	"testing"

	yaml "gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

func TestContainerToYAML(t *testing.T) {

	var (
		expectedName  = "test-container"
		expectedImage = "gcr.io/heptio/test-image:master"
		expectedCmd   = []string{"echo", "Hello world!"}
	)
	container := &Container{
		Container: v1.Container{
			Name:    expectedName,
			Image:   expectedImage,
			Command: expectedCmd,
		},
	}

	yamlDoc, err := kuberuntime.Encode(Encoder, container)

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
