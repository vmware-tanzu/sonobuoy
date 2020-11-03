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
	"io/ioutil"
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
			name:    "v1.16 is not valid",
			version: "v1.16.0",
			error:   true,
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
		{
			name:    "v1.12 is not valid",
			version: "v1.12.0",
			error:   true,
			expect:  "No matching configuration for k8s version: 1.12",
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

func TestGetE2EImages(t *testing.T) {
	version := "v1.17.0"
	registry, err := NewRegistryList("", version)
	if err != nil {
		t.Fatalf("unexpected error from NewRegistryList: %q", err)
	}

	imageNames, err := GetE2EImages("", version)
	if err != nil {
		t.Fatalf("unexpected error from GetE2EImages: %q", err)
	}

	expectedRegistry := registry.v1_17()
	if len(imageNames) != len(expectedRegistry) {
		t.Fatalf("Unexpected number of images returned, expected %v, got %v", len(expectedRegistry), len(imageNames))
	}

	// Check one of the returned image names to ensure correct format
	registryImage := expectedRegistry[Agnhost]
	registryImageName := registryImage.GetFullyQualifiedImageName()
	if !contains(imageNames, registryImageName) {
		t.Errorf("Expected result of GetImageNames to contain registry image %q", registryImageName)
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

	tmpfile, err := ioutil.TempFile("", "config.*.yaml")
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

func TestGetE2EImageTagPairs(t *testing.T) {
	version := "v1.17.0"
	customRegistry := "my-custom/registry"
	customRegistries, err := createTestRegistryConfig(customRegistry, version)
	if err != nil {
		t.Fatalf("unexpected error creating temp registry config: %q", err)
	}

	imageTagPairs, err := GetE2EImageTagPairs(customRegistries, version)
	if err != nil {
		t.Fatalf("unexpected error from GetE2ETagPairs: %q", err)
	}

	defaultRegistry, err := NewRegistryList("", version)
	if err != nil {
		t.Fatalf("unexpected error from NewRegistryList: %q", err)
	}
	expectedDefaultRegistry := defaultRegistry.v1_17()
	if len(imageTagPairs) != len(expectedDefaultRegistry) {
		t.Fatalf("unexpected number of image tag pairs returned, expected %v, got %v", len(expectedDefaultRegistry), len(imageTagPairs))
	}

	// As a sample, check one of the images for E2E and assert their mapping
	var e2eImageTagPair TagPair
	for _, imageTagPair := range imageTagPairs {
		if strings.Contains(imageTagPair.Src, "e2e") {
			e2eImageTagPair = imageTagPair
			break
		}
	}

	if e2eImageTagPair == (TagPair{}) {
		t.Errorf("no e2eImageTagPair is found")
	}
	if strings.HasPrefix(e2eImageTagPair.Src, customRegistry) {
		t.Errorf("src image should not have custom registry prefix: %q", e2eImageTagPair.Src)
	}

	imageComponents := strings.SplitAfter(e2eImageTagPair.Src, "/")
	if !strings.HasPrefix(e2eImageTagPair.Dst, customRegistry) {
		t.Errorf("expected Dst image to have prefix %q, got %q", customRegistry, e2eImageTagPair.Dst)
	}
	if !strings.HasSuffix(e2eImageTagPair.Dst, imageComponents[len(imageComponents)-1]) {
		t.Errorf("expected Dst image to have suffix %q, got %q", imageComponents[len(imageComponents)-1], e2eImageTagPair.Dst)
	}
}

func contains(set []string, val string) bool {
	for _, v := range set {
		if v == val {
			return true
		}
	}
	return false
}
