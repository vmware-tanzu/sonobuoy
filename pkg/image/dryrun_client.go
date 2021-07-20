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
	"strings"

	"github.com/sirupsen/logrus"
)

// DryRunClient is an implementation of Client that logs the image operations that would
// be performed rather than performing them.
type DryRunClient struct{}

const v1_19_images = `docker.io/library/httpd:2.4.38-alpine
docker.io/library/httpd:2.4.39-alpine
gcr.io/kubernetes-e2e-test-images/nautilus:1.0
docker.io/library/nginx:1.14-alpine
gcr.io/kubernetes-e2e-test-images/volume/gluster:1.0
gcr.io/kubernetes-e2e-test-images/cuda-vector-add:2.0
k8s.gcr.io/build-image/debian-iptables:v12.1.2
docker.io/gluster/glusterdynamic-provisioner:v1.0
docker.io/library/nginx:1.15-alpine
k8s.gcr.io/prometheus-dummy-exporter:v0.1.0
k8s.gcr.io/prometheus-to-sd:v0.5.0
gcr.io/kubernetes-e2e-test-images/resource-consumer:1.5
k8s.gcr.io/sd-dummy-exporter:v0.2.0
gcr.io/k8s-authenticated-test/agnhost:2.6
gcr.io/authenticated-image-pulling/alpine:3.7
gcr.io/kubernetes-e2e-test-images/jessie-dnsutils:1.0
gcr.io/kubernetes-e2e-test-images/volume/nfs:1.0
gcr.io/kubernetes-e2e-test-images/volume/rbd:1.0.1
gcr.io/kubernetes-e2e-test-images/apparmor-loader:1.0
gcr.io/kubernetes-e2e-test-images/cuda-vector-add:1.0
gcr.io/kubernetes-e2e-test-images/nonewprivs:1.0
gcr.io/kubernetes-e2e-test-images/sample-apiserver:1.17
invalid.com/invalid/alpine:3.1
gcr.io/kubernetes-e2e-test-images/ipc-utils:1.0
docker.io/library/perl:5.26
gcr.io/kubernetes-e2e-test-images/regression-issue-74839-amd64:1.0
k8s.gcr.io/e2e-test-images/agnhost:2.20
gcr.io/authenticated-image-pulling/windows-nanoserver:v1
k8s.gcr.io/etcd:3.4.13-0
gcr.io/kubernetes-e2e-test-images/echoserver:2.2
docker.io/library/redis:5.0.5-alpine
gcr.io/kubernetes-e2e-test-images/metadata-concealment:1.2
k8s.gcr.io/pause:3.2
gcr.io/kubernetes-e2e-test-images/nonroot:1.0
gcr.io/kubernetes-e2e-test-images/volume/iscsi:2.0
docker.io/library/busybox:1.29
gcr.io/kubernetes-e2e-test-images/kitten:1.0
k8s.gcr.io/sig-storage/nfs-provisioner:v2.2.2
`

func (i DryRunClient) RunImage(entrypoint string, image string, args ...string) ([]string, error) {
	return strings.Split(v1_19_images, "\n"), nil
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
