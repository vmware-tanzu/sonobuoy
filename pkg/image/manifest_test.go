/*
Copyright the Sonobuoy contributors 2019

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

package image

import (
	"os"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

func TestGetDefaultImageRegistryVersionValidation(t *testing.T) {
	tests := []struct {
		name    string
		version string
		error   bool
		expect  string
	}{
		{
			name:    "Non valid version results in error",
			version: "not-a-valid-version",
			error:   true,
			expect:  "\"not-a-valid-version\" is invalid",
		},
		{
			name:    "v1.17 is valid",
			version: "v1.17.0",
			error:   false,
		},
		{
			name:    "v1.18 is valid",
			version: "v1.18.0",
			error:   false,
		},
		{
			name:    "v1.19 is valid",
			version: "v1.19.0",
			error:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetDefaultImageRegistries(tc.version)
			if tc.error && err == nil {
				t.Fatal("expected error, got nil")
			} else if !tc.error && err != nil {
				t.Fatalf("expected no error, got %v", err)
			} else if tc.error && !strings.Contains(err.Error(), tc.expect) {
				t.Fatalf("expected error to contain %q, got %v", tc.expect, err.Error())
			}
		})
	}
}

func TestFullQualifiedImageName(t *testing.T) {
	img := Config{
		registry: "docker.io/sonobuoy",
		name:     "testimage",
		tag:      "latest",
	}
	expected := "docker.io/sonobuoy/testimage:latest"
	actual := img.GetFullyQualifiedImageName()
	if actual != expected {
		t.Errorf("expected image name to be %q, got %q", expected, actual)
	}
}

func createTestRegistryConfig(customRegistry, version string) (string, error) {
	registries, err := GetDefaultImageRegistries(version)
	if err != nil {
		return "", err
	}

	registries.E2eRegistry = customRegistry
	registries.DockerLibraryRegistry = customRegistry
	registries.GcRegistry = customRegistry
	registries.SampleRegistry = customRegistry

	tmpfile, err := os.CreateTemp("", "config.*.yaml")
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()

	d, err := yaml.Marshal(&registries)
	if err != nil {
		return "", err
	}
	if _, err := tmpfile.Write(d); err != nil {
		return "", err
	}
	return tmpfile.Name(), nil
}

func contains(set []string, val string) bool {
	for _, v := range set {
		if v == val {
			return true
		}
	}
	return false
}
