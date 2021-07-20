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
	"testing"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/image/docker"
)

var imgs = []string{"test1/foo.io/sonobuoy:x.y"}

type FakeDockerClient struct {
	imageExists bool
	pushFails   bool
	pullFails   bool
	tagFails    bool
	saveFails   bool
	deleteFails bool
}

func (l FakeDockerClient) Run(entrypoint string, image string, args ...string) ([]string, error) {
	return l.Run(entrypoint, image, args...)
}

func (l FakeDockerClient) PullIfNotPresent(image string, retries int) error {
	if l.imageExists {
		return nil
	}
	return l.Pull(image, retries)
}

func (l FakeDockerClient) Pull(image string, retries int) error {
	if l.pullFails {
		return errors.New("pull failed")
	}
	return nil
}

func (l FakeDockerClient) Push(image string, retries int) error {
	if l.pushFails {
		return errors.New("push failed")
	}
	return nil
}

func (l FakeDockerClient) Tag(src, dest string, retries int) error {
	if l.tagFails {
		return errors.New("tag failed")
	}
	return nil
}

func (l FakeDockerClient) Rmi(image string, retries int) error {
	if l.deleteFails {
		return errors.New("delete failed")
	}
	return nil
}

func (l FakeDockerClient) Save(images []string, filename string) error {
	if l.saveFails {
		return errors.New("save failed")
	}
	return nil
}

func TestPushImages(t *testing.T) {
	imageTagPairs := []TagPair{
		{
			Src: imgs[0],
			Dst: "test1/private.io/sonobuoy:x.y",
		},
	}
	imageTagPairSame := []TagPair{
		{
			Src: imgs[0],
			Dst: imgs[0],
		},
	}

	tests := map[string]struct {
		client         docker.Docker
		imageTagPairs  []TagPair
		wantErrorCount int
	}{
		"simple": {
			client: FakeDockerClient{
				pushFails: false,
				tagFails:  false,
			},
			imageTagPairs:  imageTagPairs,
			wantErrorCount: 0,
		},
		"tag fails": {
			client: FakeDockerClient{
				pushFails: false,
				tagFails:  true,
			},
			imageTagPairs:  imageTagPairs,
			wantErrorCount: 1,
		},
		"push fails": {
			client: FakeDockerClient{
				pushFails: true,
				tagFails:  true,
			},
			imageTagPairs:  imageTagPairs,
			wantErrorCount: 2,
		},
		"source images equal destination images": {
			client: FakeDockerClient{
				pushFails: true,
				tagFails:  true,
			},
			imageTagPairs:  imageTagPairSame,
			wantErrorCount: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			imgClient := DockerClient{
				dockerClient: tc.client,
			}

			got := imgClient.PushImages(tc.imageTagPairs, 0)

			if len(got) != tc.wantErrorCount {
				t.Fatalf("Expected errors: %d but got %d", tc.wantErrorCount, len(got))
			}
		})
	}
}
func TestPullImages(t *testing.T) {
	tests := map[string]struct {
		client         docker.Docker
		wantErrorCount int
	}{
		"simple": {
			client: FakeDockerClient{
				imageExists: false,
				pullFails:   false,
			},

			wantErrorCount: 0,
		},
		"image exists": {
			client: FakeDockerClient{
				imageExists: true,
				pullFails:   false,
			},

			wantErrorCount: 0,
		},
		"error pulling image": {
			client: FakeDockerClient{
				imageExists: false,
				pullFails:   true,
			},
			wantErrorCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			imgClient := DockerClient{
				dockerClient: tc.client,
			}

			got := imgClient.PullImages(imgs, 0)

			if len(got) != tc.wantErrorCount {
				t.Fatalf("Expected errors: %d but got %d", tc.wantErrorCount, len(got))
			}
		})
	}
}
func TestDownloadImages(t *testing.T) {
	const k8sVersion = "99.YY.ZZ"
	images := []string{"foo.io/sonobuoy/test:1.0"}

	tests := map[string]struct {
		client       docker.Docker
		wantFileName string
		wantError    bool
	}{
		"simple": {
			client: FakeDockerClient{
				saveFails: false,
			},
			wantFileName: getTarFileName(k8sVersion),
			wantError:    false,
		},
		"fail": {
			client: FakeDockerClient{
				saveFails: true,
			},
			wantFileName: "",
			wantError:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			imgClient := DockerClient{
				dockerClient: tc.client,
			}

			gotFilename, gotErr := imgClient.DownloadImages(images, k8sVersion)

			if gotErr != nil && tc.wantError != true {
				t.Fatalf("Got unexpected error: %v", gotErr)
			}

			if gotFilename != tc.wantFileName {
				t.Fatalf("Expected filename: %s but got: %s", tc.wantFileName, gotFilename)
			}
		})
	}
}
func TestDeleteImages(t *testing.T) {
	tests := map[string]struct {
		client         docker.Docker
		wantErrorCount int
	}{
		"simple": {
			client: FakeDockerClient{
				deleteFails: false,
			},
			wantErrorCount: 0,
		},
		"fail": {
			client: FakeDockerClient{
				deleteFails: true,
			},
			wantErrorCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			imgClient := DockerClient{
				dockerClient: tc.client,
			}

			got := imgClient.DeleteImages(imgs, 0)

			if len(got) != tc.wantErrorCount {
				t.Fatalf("Expected errors: %d but got %d", tc.wantErrorCount, len(got))
			}
		})
	}
}
