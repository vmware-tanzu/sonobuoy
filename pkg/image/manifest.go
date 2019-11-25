/*
Copyright 2017 The Kubernetes Authors.
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

// NOTE: This is manually replicated from: https://github.com/kubernetes/kubernetes/blob/master/test/utils/image/manifest.go

package image

import (
	"fmt"
	"io/ioutil"

	version "github.com/hashicorp/go-version"
	yaml "gopkg.in/yaml.v2"
)

const (
	dockerLibraryRegistry   = "docker.io/library"
	e2eRegistry             = "gcr.io/kubernetes-e2e-test-images"
	etcdRegistry            = "quay.io/coreos"
	gcAuthenticatedRegistry = "gcr.io/authenticated-image-pulling"
	gcRegistry              = "k8s.gcr.io"
	gcrReleaseRegistry      = "gcr.io/gke-release"
	googleContainerRegistry = "gcr.io/google-containers"
	invalidRegistry         = "invalid.com/invalid"
	privateRegistry         = "gcr.io/k8s-authenticated-test"
	quayK8sCSI              = "quay.io/k8scsi"
	sampleRegistry          = "gcr.io/google-samples"
)

// RegistryList holds public and private image registries
type RegistryList struct {
	DockerLibraryRegistry   string `yaml:"dockerLibraryRegistry,omitempty"`
	E2eRegistry             string `yaml:"e2eRegistry,omitempty"`
	EtcdRegistry            string `yaml:"etcdRegistry,omitempty"`
	GcAuthenticatedRegistry string `yaml:"gcAuthenticatedRegistry,omitempty"`
	GcRegistry              string `yaml:"gcRegistry,omitempty"`
	GcrReleaseRegistry      string `yaml:"gcrReleaseRegistry,omitempty"`
	GoogleContainerRegistry string `yaml:"googleContainerRegistry,omitempty"`
	InvalidRegistry         string `yaml:"invalidRegistry,omitempty"`
	PrivateRegistry         string `yaml:"privateRegistry,omitempty"`
	QuayK8sCSI              string `yaml:"quayK8sCSI,omitempty"`
	SampleRegistry          string `yaml:"sampleRegistry,omitempty"`

	K8sVersion *version.Version `yaml:"-"`
	Images     map[int]Config   `yaml:"-"`
}

// Config holds an images registry, name, and version
type Config struct {
	registry string
	name     string
	version  string
}

// NewRegistryList returns a default registry or one that matches a config file passed
func NewRegistryList(repoConfig, k8sVersion string) (*RegistryList, error) {
	registry := &RegistryList{
		DockerLibraryRegistry:   dockerLibraryRegistry,
		E2eRegistry:             e2eRegistry,
		EtcdRegistry:            etcdRegistry,
		GcAuthenticatedRegistry: gcAuthenticatedRegistry,
		GcRegistry:              gcRegistry,
		GcrReleaseRegistry:      gcrReleaseRegistry,
		GoogleContainerRegistry: googleContainerRegistry,
		InvalidRegistry:         invalidRegistry,
		PrivateRegistry:         privateRegistry,
		QuayK8sCSI:              quayK8sCSI,
		SampleRegistry:          sampleRegistry,
	}

	// Load in a config file
	if repoConfig != "" {

		fileContent, err := ioutil.ReadFile(repoConfig)
		if err != nil {
			panic(fmt.Errorf("Error reading '%v' file contents: %v", repoConfig, err))
		}

		err = yaml.Unmarshal(fileContent, &registry)
		if err != nil {
			panic(fmt.Errorf("Error unmarshalling '%v' YAML file: %v", repoConfig, err))
		}
	}

	// Init images for k8s version & repos configured
	version, err := validateVersion(k8sVersion)
	if err != nil {
		return nil, err
	}

	registry.K8sVersion = version

	return registry, nil
}

// GetImageConfigs returns the map of imageConfigs
func (r *RegistryList) GetImageConfigs() (map[string]Config, error) {
	switch r.K8sVersion.Segments()[0] {
	case 1:
		switch r.K8sVersion.Segments()[1] {
		case 13:
			return r.v1_13(), nil
		case 14:
			return r.v1_14(), nil
		case 15:
			return r.v1_15(), nil
		case 16:
			return r.v1_16(), nil
		}
	}
	return map[string]Config{}, fmt.Errorf("No matching configuration for k8s version: %v", r.K8sVersion)
}

// GetDefaultImageRegistries returns the default default image registries used for
// a given version of the Kubernetes E2E tests
func GetDefaultImageRegistries(version string) (*RegistryList, error) {
	// Init images for k8s version & repos configured
	v, err := validateVersion(version)
	if err != nil {
		return nil, err
	}

	switch v.Segments()[0] {
	case 1:
		switch v.Segments()[1] {
		case 13, 14:
			return &RegistryList{
				DockerLibraryRegistry: dockerLibraryRegistry,
				E2eRegistry:           e2eRegistry,
				EtcdRegistry:          etcdRegistry,
				GcRegistry:            gcRegistry,
				SampleRegistry:        sampleRegistry,
			}, nil
		case 15:
			return &RegistryList{
				DockerLibraryRegistry: dockerLibraryRegistry,
				E2eRegistry:           e2eRegistry,
				GcRegistry:            gcRegistry,
				SampleRegistry:        sampleRegistry,
			}, nil
		case 16:
			return &RegistryList{
				DockerLibraryRegistry:   dockerLibraryRegistry,
				E2eRegistry:             e2eRegistry,
				GcRegistry:              gcRegistry,
				GoogleContainerRegistry: googleContainerRegistry,
				SampleRegistry:          sampleRegistry,

				// The following keys are used in the v1.16 registry list however their images
				// cannot be pulled as they are used as part of tests for checking image pull
				// behavior. They are omitted from the resulting config.
				// InvalidRegistry:         invalidRegistry,
				// GcAuthenticatedRegistry: gcAuthenticatedRegistry,
			}, nil
		}
	}
	return nil, fmt.Errorf("No matching configuration for k8s version: %v", v)
}

// GetE2EImage returns the fully qualified URI to an image (including version)
func (i *Config) GetE2EImage() string {
	return fmt.Sprintf("%s/%s:%s", i.registry, i.name, i.version)
}
