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
	"fmt"

	"github.com/heptio/sonobuoy/pkg/image/docker"
	"github.com/pkg/errors"
)

type ImageClient struct {
	dockerClient docker.Docker
}

func NewImageClient() ImageClient {
	return ImageClient{
		dockerClient: docker.LocalDocker{},
	}
}

func (i ImageClient) PullImages(images map[string]Config, retries int) []error {
	errs := []error{}
	for _, v := range images {
		err := i.dockerClient.PullIfNotPresent(v.GetE2EImage(), retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't pull image: %v", v.GetE2EImage()))
		}
	}
	return errs
}

func (i ImageClient) PushImages(upstreamImages, privateImages map[string]Config, retries int) []error {
	errs := []error{}
	for k, v := range upstreamImages {
		privateImg := privateImages[k]

		// Skip if the source/dest are equal
		if privateImg.GetE2EImage() == v.GetE2EImage() {
			fmt.Printf("Skipping public image: %s\n", v.GetE2EImage())
			continue
		}

		err := i.dockerClient.Tag(v.GetE2EImage(), privateImg.GetE2EImage(), retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't tag image: %v", v.GetE2EImage()))
		}

		err = i.dockerClient.Push(privateImg.GetE2EImage(), retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't push image: %v", v.GetE2EImage()))
		}
	}
	return errs
}

func (i ImageClient) DownloadImages(images []string, version string) (string, error) {
	fileName := getTarFileName(version)

	err := i.dockerClient.Save(images, fileName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't save images to tar")
	}

	return fileName, nil
}

func (i ImageClient) DeleteImages(images map[string]Config, retries int) []error {
	errs := []error{}

	for _, v := range images {
		err := i.dockerClient.Rmi(v.GetE2EImage(), retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't delete image: %v", v.GetE2EImage()))
		}
	}

	return errs
}

// GetImages gets a map of image Configs
func GetImages(e2eRegistryConfig, version string) (map[string]Config, error) {
	// Get list of upstream images that match the version
	reg, err := NewRegistryList(e2eRegistryConfig, version)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't init Registry List")
	}

	imgs, err := reg.GetImageConfigs()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get images for version")
	}
	return imgs, nil
}

// getTarFileName returns a filename matching the version of Kubernetes images are exported
func getTarFileName(version string) string {
	return fmt.Sprintf("kubernetes_e2e_images_%s.tar", version)
}
