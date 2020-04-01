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
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

const (
	dockerGluster           = "docker.io/gluster"
	dockerLibraryRegistry   = "docker.io/library"
	e2eRegistry             = "gcr.io/kubernetes-e2e-test-images"
	etcdRegistry            = "quay.io/coreos"
	gcAuthenticatedRegistry = "gcr.io/authenticated-image-pulling"
	gcRegistry              = "k8s.gcr.io"
	gcrReleaseRegistry      = "gcr.io/gke-release"
	googleContainerRegistry = "gcr.io/google-containers"
	invalidRegistry         = "invalid.com/invalid"
	privateRegistry         = "gcr.io/k8s-authenticated-test"
	promoterE2eRegistry     = "us.gcr.io/k8s-artifacts-prod/e2e-test-images"
	quayIncubator           = "quay.io/kubernetes_incubator"
	quayK8sCSI              = "quay.io/k8scsi"
	sampleRegistry          = "gcr.io/google-samples"
)

// RegistryList holds public and private image registries
type RegistryList struct {
	DockerGluster           string `yaml:"dockerGluster,omitempty"`
	DockerLibraryRegistry   string `yaml:"dockerLibraryRegistry,omitempty"`
	E2eRegistry             string `yaml:"e2eRegistry,omitempty"`
	EtcdRegistry            string `yaml:"etcdRegistry,omitempty"`
	GcAuthenticatedRegistry string `yaml:"gcAuthenticatedRegistry,omitempty"`
	GcRegistry              string `yaml:"gcRegistry,omitempty"`
	GcrReleaseRegistry      string `yaml:"gcrReleaseRegistry,omitempty"`
	GoogleContainerRegistry string `yaml:"googleContainerRegistry,omitempty"`
	InvalidRegistry         string `yaml:"invalidRegistry,omitempty"`
	PrivateRegistry         string `yaml:"privateRegistry,omitempty"`
	PromoterE2eRegistry     string `yaml:"promoterE2eRegistry"`
	QuayIncubator           string `yaml:"quayIncubator,omitempty"`
	QuayK8sCSI              string `yaml:"quayK8sCSI,omitempty"`
	SampleRegistry          string `yaml:"sampleRegistry,omitempty"`

	K8sVersion *version.Version `yaml:"-"`
	Images     map[int]Config   `yaml:"-"`
}

// Config holds an image's fully qualified name components registry, name, and tag
type Config struct {
	registry string
	name     string
	tag      string
}

// NewRegistryList returns a default registry or one that matches a config file passed
func NewRegistryList(repoConfig, k8sVersion string) (*RegistryList, error) {
	registry := &RegistryList{
		DockerGluster:           dockerGluster,
		DockerLibraryRegistry:   dockerLibraryRegistry,
		E2eRegistry:             e2eRegistry,
		EtcdRegistry:            etcdRegistry,
		GcAuthenticatedRegistry: gcAuthenticatedRegistry,
		GcRegistry:              gcRegistry,
		GcrReleaseRegistry:      gcrReleaseRegistry,
		GoogleContainerRegistry: googleContainerRegistry,
		InvalidRegistry:         invalidRegistry,
		PrivateRegistry:         privateRegistry,
		QuayIncubator:           quayIncubator,
		QuayK8sCSI:              quayK8sCSI,
		SampleRegistry:          sampleRegistry,
	}

	// Load in a config file
	if repoConfig != "" {

		fileContent, err := ioutil.ReadFile(repoConfig)
		if err != nil {
			return nil, fmt.Errorf("Error reading '%v' file contents: %v", repoConfig, err)
		}

		err = yaml.Unmarshal(fileContent, &registry)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshalling '%v' YAML file: %v", repoConfig, err)
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

// getImageConfigs returns the map of image Config for the registry version
func (r *RegistryList) getImageConfigs() (map[string]Config, error) {
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
		case 17:
			return r.v1_17(), nil
		case 18:
			return r.v1_18(), nil
		}
	}
	return map[string]Config{}, fmt.Errorf("No matching configuration for k8s version: %v", r.K8sVersion)

}

// GetE2EImages gets a list of E2E image names
func GetE2EImages(e2eRegistryConfig, version string) ([]string, error) {
	// Get list of upstream images that match the version
	reg, err := NewRegistryList(e2eRegistryConfig, version)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create image registry list")
	}

	imgConfigs, err := reg.getImageConfigs()
	if err != nil {
		return []string{}, errors.Wrap(err, "couldn't get images for version")
	}

	imageNames := []string{}
	for _, imageConfig := range imgConfigs {
		imageNames = append(imageNames, imageConfig.GetFullyQualifiedImageName())

	}
	return imageNames, nil
}

// GetE2EImagePairs gets a list of E2E image tag pairs from the default src to custom destination
func GetE2EImageTagPairs(e2eRegistryConfig, version string) ([]TagPair, error) {
	defaultImageRegistry, err := NewRegistryList("", version)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create image registry list")
	}
	defaultImageConfigs, err := defaultImageRegistry.getImageConfigs()
	if err != nil {
		return []TagPair{}, errors.Wrap(err, "couldn't get images for version")
	}

	customImageRegistry, err := NewRegistryList(e2eRegistryConfig, version)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create image registry list")
	}
	customImageConfigs, err := customImageRegistry.getImageConfigs()
	if err != nil {
		return []TagPair{}, errors.Wrap(err, "couldn't get images for version")
	}

	var imageTagPairs []TagPair
	for name, cfg := range defaultImageConfigs {
		imageTagPairs = append(imageTagPairs, TagPair{
			Src: cfg.GetFullyQualifiedImageName(),
			Dst: customImageConfigs[name].GetFullyQualifiedImageName(),
		})
	}
	return imageTagPairs, nil
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
		case 17:
			return &RegistryList{
				E2eRegistry:             e2eRegistry,
				DockerLibraryRegistry:   dockerLibraryRegistry,
				GcRegistry:              gcRegistry,
				GoogleContainerRegistry: googleContainerRegistry,
				DockerGluster:           dockerGluster,
				QuayIncubator:           quayIncubator,

				// The following keys are used in the v1.17 registry list however their images
				// cannot be pulled as they are used as part of tests for checking image pull
				// behavior. They are omitted from the resulting config.
				// InvalidRegistry:         invalidRegistry,
				// GcAuthenticatedRegistry: gcAuthenticatedRegistry,
				// PrivateRegistry:         privateRegistry,
			}, nil
		case 18:
			return &RegistryList{
				E2eRegistry:             e2eRegistry,
				DockerLibraryRegistry:   dockerLibraryRegistry,
				GcRegistry:              gcRegistry,
				GoogleContainerRegistry: googleContainerRegistry,
				DockerGluster:           dockerGluster,
				QuayIncubator:           quayIncubator,
				PromoterE2eRegistry:     promoterE2eRegistry,

				// The following keys are used in the v1.18 registry list however their images
				// cannot be pulled as they are used as part of tests for checking image pull
				// behavior. They are omitted from the resulting config.
				// InvalidRegistry:         invalidRegistry,
				// GcAuthenticatedRegistry: gcAuthenticatedRegistry,
				// PrivateRegistry:         privateRegistry,
			}, nil
		}
	}
	return nil, fmt.Errorf("No matching configuration for k8s version: %v", v)
}

// GetFullyQualifiedImageName returns the fully qualified URI to an image (including tag)
func (i Config) GetFullyQualifiedImageName() string {
	return fmt.Sprintf("%s/%s:%s", i.registry, i.name, i.tag)
}
