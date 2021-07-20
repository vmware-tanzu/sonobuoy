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

// Client is the interface for interacting with images.
type Client interface {
	PullImages(images []string, retries int) []error
	PushImages(images []TagPair, retries int) []error
	DownloadImages(images []string, version string) (string, error)
	DeleteImages(images []string, retries int) []error
	RunImage(entrypoint string, image string, args ...string) ([]string, error)
}
