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

// DockerClient is an implementation of Client that uses the local docker installation
// to interact with images.
type DockerClient struct {
	dockerClient docker.Docker
}

// TagPair represents a source image and a destination image that it will be tagged and
// pushed as.
type TagPair struct {
	Src string
	Dst string
}

// NewDockerClient returns a DockerClient that can interact with the local docker installation.
func NewDockerClient() Client {
	return DockerClient{
		dockerClient: docker.LocalDocker{},
	}
}

// PullImages pulls the given list of images, skipping if they are already present on the machine.
// It will retry for the provided number of retries on failure.
func (i DockerClient) PullImages(images []string, retries int) []error {
	errs := []error{}
	for _, image := range images {
		err := i.dockerClient.PullIfNotPresent(image, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't pull image: %v", image))
		}
	}
	return errs
}

// PushImages will tag each of the source images as the destination image and push.
// It will skip the operation if the image source and destination are equal.
// It will retry for the provided number of retries on failure.
func (i DockerClient) PushImages(images []TagPair, retries int) []error {
	errs := []error{}
	for _, image := range images {
		// Skip if the source/dest are equal
		if image.Src == image.Dst {
			fmt.Printf("Skipping public image: %s\n", image.Src)
			continue
		}

		err := i.dockerClient.Tag(image.Src, image.Dst, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't tag image %q as %q", image.Src, image.Dst))
		}

		err = i.dockerClient.Push(image.Dst, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't push image: %v", image.Dst))
		}
	}
	return errs
}

// DownloadImages exports the list of images to a tar file. The provided version will be included in the
// resulting file name.
func (i DockerClient) DownloadImages(images []string, version string) (string, error) {
	fileName := getTarFileName(version)

	err := i.dockerClient.Save(images, fileName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't save images to tar")
	}

	return fileName, nil
}

// DeleteImages deletes the given list of images from the local machine.
// It will retry for the provided number of retries on failure.
func (i DockerClient) DeleteImages(images []string, retries int) []error {
	errs := []error{}

	for _, image := range images {
		err := i.dockerClient.Rmi(image, retries)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "couldn't delete image: %v", image))
		}
	}

	return errs
}

func (i DockerClient) RunImage(entrypoint string, image string, args ...string) ([]string, error) {
	output, err := i.dockerClient.Run(entrypoint, image, args...)
	if err != nil {
		return []string{}, err
	}
	return output, nil
}

// getTarFileName returns a filename matching the version of Kubernetes images are exported
func getTarFileName(version string) string {
	return fmt.Sprintf("kubernetes_e2e_images_%s.tar", version)
}
