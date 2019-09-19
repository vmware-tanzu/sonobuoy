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

// RegistryList holds public and private image registries
type RegistryList struct {
	GcAuthenticatedRegistry string `yaml:"gcAuthenticatedRegistry"`
	DockerLibraryRegistry   string `yaml:"dockerLibraryRegistry"`
	E2eRegistry             string `yaml:"e2eRegistry"`
	InvalidRegistry         string `yaml:"invalidRegistry"`
	GcRegistry              string `yaml:"gcRegistry"`
	GcrReleaseRegistry      string `yaml:"gcrReleaseRegistry"`
	GoogleContainerRegistry string `yaml:"googleContainerRegistry"`
	EtcdRegistry            string `yaml:"etcdRegistry"`
	PrivateRegistry         string `yaml:"privateRegistry"`
	SampleRegistry          string `yaml:"sampleRegistry"`
	QuayK8sCSI              string `yaml:"quayK8sCSI"`

	K8sVersion *version.Version
	Images     map[int]Config
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
		DockerLibraryRegistry:   "docker.io/library",
		E2eRegistry:             "gcr.io/kubernetes-e2e-test-images",
		EtcdRegistry:            "quay.io/coreos",
		GcRegistry:              "k8s.gcr.io",
		PrivateRegistry:         "gcr.io/k8s-authenticated-test",
		SampleRegistry:          "gcr.io/google-samples",
		GcAuthenticatedRegistry: "gcr.io/authenticated-image-pulling",
		InvalidRegistry:         "invalid.com/invalid",
		GcrReleaseRegistry:      "gcr.io/gke-release",
		GoogleContainerRegistry: "gcr.io/google-containers",
		QuayK8sCSI:              "quay.io/k8scsi",
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

// GetE2EImage returns the fully qualified URI to an image (including version)
func (i *Config) GetE2EImage() string {
	return fmt.Sprintf("%s/%s:%s", i.registry, i.name, i.version)
}
