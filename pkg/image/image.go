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

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/image/docker"
)

type ImageClient struct {
	dockerClient docker.Docker
}

func NewImageClient() ImageClient {
	return ImageClient{
		dockerClient: docker.LocalDocker{},
	}
}

func (i ImageClient) PullImages(images []string, retries int) []error {
	errs := []error{}
	for _, image := range images {
		err := i.dockerClient.PullIfNotPresent(image, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't pull image: %v", image))
		}
	}
	return errs
}

func (i ImageClient) PushImages(upstreamImages, privateImages []string, retries int) []error {
	errs := []error{}
	for k, upstreamImg := range upstreamImages {
		privateImg := privateImages[k]

		// Skip if the source/dest are equal
		if privateImg == upstreamImg {
			fmt.Printf("Skipping public image: %s\n", upstreamImg)
			continue
		}

		err := i.dockerClient.Tag(upstreamImg, privateImg, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't tag image %q as %q", upstreamImg, privateImg))
		}

		err = i.dockerClient.Push(privateImg, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't push image: %v", privateImg))
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

func (i ImageClient) DeleteImages(images []string, retries int) []error {
	errs := []error{}

	for _, image := range images {
		err := i.dockerClient.Rmi(image, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't delete image: %v", image))
		}
	}

	return errs
}

// GetE2EImages gets a list of E2E image names
func GetE2EImages(e2eRegistryConfig, version string) ([]string, error) {
	// Get list of upstream images that match the version
	reg, err := NewRegistryList(e2eRegistryConfig, version)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create image registry list")
	}

	imgs, err := reg.GetImageNames()
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get images for version")
	}
	return imgs, nil
}

// getTarFileName returns a filename matching the version of Kubernetes images are exported
func getTarFileName(version string) string {
	return fmt.Sprintf("kubernetes_e2e_images_%s.tar", version)
}
