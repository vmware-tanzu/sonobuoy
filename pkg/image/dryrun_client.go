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

package image

import (
	"github.com/sirupsen/logrus"
)

// DryRunClient is an implementation of Client that logs the image operations that would
// be performed rather than performing them.
type DryRunClient struct{}

func (i DryRunClient) RunImage(image string, entryPoint string, env map[string]string, args ...string) ([]string, error) {
	// Called from collectPluginsImages, retrieve e2e images
	// Return empty list instead of outdated info
	return []string{}, nil
}

// PullImages logs the images that would be pulled.
func (i DryRunClient) PullImages(images []string, retries int) []error {
	for _, image := range images {
		logrus.Infof("Pulling image: %s", image)
	}
	return []error{}
}

// PushImages logs what the images would be tagged and pushed as.
func (i DryRunClient) PushImages(images []TagPair, retries int) []error {
	for _, image := range images {
		logrus.Infof("Tagging image: %s as %s", image.Src, image.Dst)
		logrus.Infof("Pushing image: %s", image.Dst)
	}
	return []error{}
}

// DownloadImages logs that the images would be saved and returns the tarball name.
func (i DryRunClient) DownloadImages(images []string, version string) (string, error) {
	logrus.Info("Saving images")
	return getTarFileName(version), nil
}

// DeleteImages logs which images would be deleted.
func (i DryRunClient) DeleteImages(images []string, retries int) []error {
	for _, image := range images {
		logrus.Infof("Deleting image: %s\n", image)
	}
	return []error{}
}
