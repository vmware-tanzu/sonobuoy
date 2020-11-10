/*
Copyright the Sonobuoy contributors 2020

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

package app

import (
	"fmt"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/image"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func convertImagesToPairs(images []string, customRegistry, e2eRegistryConfig, k8sVersion string) ([]image.TagPair, error) {
	imageTagPairs := []image.TagPair{}
	customRegistryList, err := image.NewRegistryList(e2eRegistryConfig, k8sVersion)
	if err != nil {
		return imageTagPairs, err
	}
	for _, imageURL := range images {
		imageTagPairs = append(imageTagPairs, image.TagPair{
			Src: imageURL,
			Dst: translateRegistry(imageURL, customRegistry, customRegistryList),
		})
	}
	return imageTagPairs, nil
}

func translateRegistry(imageURL string, customRegistry string, customRegistryList *image.RegistryList) string {
	parts := strings.Split(imageURL, "/")
	countParts := len(parts)
	registryAndUser := strings.Join(parts[:countParts-1], "/")

	switch registryAndUser {
	case "gcr.io/e2e-test-images":
		registryAndUser = customRegistryList.PromoterE2eRegistry
	case "gcr.io/kubernetes-e2e-test-images":
		registryAndUser = customRegistryList.E2eRegistry
	case "gcr.io/kubernetes-e2e-test-images/volume":
		registryAndUser = customRegistryList.E2eVolumeRegistry
	case "k8s.gcr.io":
		registryAndUser = customRegistryList.GcRegistry
	case "k8s.gcr.io/sig-storage":
		registryAndUser = customRegistryList.SigStorageRegistry
	case "gcr.io/k8s-authenticated-test":
		registryAndUser = customRegistryList.PrivateRegistry
	case "gcr.io/google-samples":
		registryAndUser = customRegistryList.SampleRegistry
	case "gcr.io/gke-release":
		registryAndUser = customRegistryList.GcrReleaseRegistry
	case "docker.io/library":
		registryAndUser = customRegistryList.DockerLibraryRegistry
	case "sonobuoy":
		if customRegistry != "" {
			registryAndUser = customRegistry
		}
	default:
		if countParts != 1 {
			logrus.Warnf("unable to find internal registry map for image: %s, leaving unchanged", imageURL)
			return imageURL
		}

		// We assume we found an image from docker hub library
		// e.g. openjdk -> docker.io/library/openjdk
		registryAndUser = customRegistryList.DockerLibraryRegistry
	}

	return fmt.Sprintf("%s/%s", registryAndUser, parts[countParts-1])
}

func collectPluginsImages(plugins []string, k8sVersion string, client image.Client) ([]string, error) {
	images := []string{
		config.DefaultImage,
	}
	for _, plugin := range plugins {
		switch plugin {
		case systemdLogsPlugin:
			images = append(images, config.DefaultSystemdLogsImage)
		case e2ePlugin:
			conformanceImage := resolveConformanceImage(k8sVersion)
			images = append(images, conformanceImage)
			logrus.Info("conformance image to be used: ", conformanceImage)

			// pull before running to ensure stderr is empty, because...
			client.PullImages([]string{conformanceImage}, numDockerRetries)

			// this combines stdout and stderr
			e2eImages, err := client.RunImage(conformanceImage, "e2e.test", "--list-images")
			if err != nil {
				return images, errors.Wrap(err, "failed to gather e2e images from conformance image")
			}

			// in case there are empty newlines getting parsed as a slice element
			validE2eImages := []string{}
			for _, e2eImage := range e2eImages {
				if e2eImage != "" {
					validE2eImages = append(validE2eImages, e2eImage)
				}
			}
			images = append(images, validE2eImages...)
		default:
			return images, errors.Errorf("Unsupported plugin: %v", plugin)
		}
	}
	return images, nil
}
