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

package docker

import (
	"bytes"
	"encoding/json"
	"time"

	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/image/exec"
)

type Docker interface {
	Inspect(image string, retries int) error
	PullIfNotPresent(image string, retries int) error
	Pull(image string, retries int) error
	Push(image string, retries int) error
	Tag(src, dest string, retries int) error
	Rmi(image string, retries int) error
	Save(images []string, filename string) error
	Run(image string, entryPoint string, env map[string]string, args ...string) ([]string, error)
}

type LocalDocker struct {
}

type inspectResponse struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"layers"`
	Manifests []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
		Platform  struct {
			Architecture string `json:"architecture"`
			Os           string `json:"os"`
		} `json:"platform,omitempty"`
	} `json:"manifests"`
}

func (l LocalDocker) Run(image string, entryPoint string, env map[string]string, args ...string) ([]string, error) {
	dockerArgs := []string{"run", "--rm"}
	if len(entryPoint) > 0 {
		dockerArgs = append(dockerArgs, fmt.Sprintf("--entrypoint=%v", entryPoint))
	}
	if len(env) > 0 {
		for k, v := range env {
			dockerArgs = append(dockerArgs, fmt.Sprintf("-e=%v=%v", k, v))
		}
	}
	dockerArgs = append(dockerArgs, image)
	dockerArgs = append(dockerArgs, args...)
	cmd := exec.Command("docker", dockerArgs...)
	return exec.CombinedOutputLines(cmd)
}

// Inspect
func (l LocalDocker) Inspect(image string, retries int) error {
	i := inspectResponse{}
	cmd := exec.Command("docker", "buildx", "imagetools", "inspect", "--raw", image)
	var buff bytes.Buffer
	cmd.SetStdout(&buff)
	cmd.SetStderr(&buff)

	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to run Docker command")
	}
	if err := json.Unmarshal(buff.Bytes(), &i); err != nil {
		for i := 1; i <= retries; i++ {
			log.Debug(buff.String())
			log.Debugf("Image inspection: %s retrying attempt: %v", image, i)
			buff.Reset()
			err := cmd.Run()
			if err != nil {
				log.Debug(err)
				time.Sleep(1 * time.Second)
			}
		}
		return errors.Wrapf(err, "Image: %s not found in registry", image)
	}

	if i.Config.Digest != "" {
		log.Debugf("Image: %s found in registry @%s", image, i.Config.Digest)
	}

	if len(i.Manifests) > 0 {
		log.Debugf("Image: %s found in registry @%s", image, i.Manifests[0].Digest)
	}
	defer buff.Reset()
	return nil
}

// PullIfNotPresent will pull an image if it is not present locally
// retrying up to "retries" times. Returns errors from pulling.
func (l LocalDocker) PullIfNotPresent(image string, retries int) error {
	cmd := exec.Command("docker", "inspect", "--type=image", image)
	if err := cmd.Run(); err == nil {
		log.Debugf("Image: %s present locally", image)
		return nil
	}
	// otherwise try to pull it
	return l.Pull(image, retries)
}

// Pull pulls an image, retrying up to retries times
func (l LocalDocker) Pull(image string, retries int) error {
	log.Infof("Pulling image: %s ...", image)
	return exec.RunLoggingOutputOnFail(exec.Command("docker", "pull", image), retries)
}

// Push pushes an image, retrying up to retries times
func (l LocalDocker) Push(image string, retries int) error {
	log.Infof("Pushing image: %s ...", image)
	return exec.RunLoggingOutputOnFail(exec.Command("docker", "push", image), retries)
}

// Tag tags an image, retrying up to retries times
func (l LocalDocker) Tag(src, dest string, retries int) error {
	log.Infof("Tagging image: %s as %s ...", src, dest)
	return exec.RunLoggingOutputOnFail(exec.Command("docker", "tag", src, dest), retries)
}

// Rmi removes an image, retrying up to retries times
func (l LocalDocker) Rmi(image string, retries int) error {
	log.Infof("Deleting image: %s ...", image)
	return exec.RunLoggingOutputOnFail(exec.Command("docker", "rmi", image), retries)
}

// Save exports a set of images to a tar file
func (l LocalDocker) Save(images []string, filename string) error {
	log.Info("Saving images: ...")

	//TODO(stevesloka) Check if all images exist on local client first

	// Build out docker command
	args := append([]string{"save"}, images...)
	args = append(args, "--output", filename)

	return exec.RunLoggingOutputOnFail(exec.Command("docker", args...), 0)
}
