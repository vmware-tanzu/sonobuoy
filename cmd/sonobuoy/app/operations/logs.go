/*
Copyright 2018 Heptio Inc.

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

package operations

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/utils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	podLogSeparator = strings.Repeat("-", 79)
)

// Logs gathers the logs for the containers in the sonobuoy namespace and prints them

// GetLogs streams logs from the sonobuoy pod by default to stdout.
func GetLogs(namespace string, follow bool) error {
	client, err := utils.OutOfClusterClient()
	if err != nil {
		return errors.Wrap(err, "could not load kubernetes client")
	}

	// Follow the main logs if -f is provided
	if follow {
		return streamLogs(client, namespace, utils.SonobuoyPod, &v1.PodLogOptions{Follow: true})
	}

	// Otherwise print every log we can
	pods, err := client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "could not list pods")
	}
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			logrus.WithFields(logrus.Fields{"pod": pod.Name, "container": container.Name}).Info("Printing container logs")
			err := streamLogs(client, namespace, pod.Name, &v1.PodLogOptions{
				Container: container.Name,
			})
			if err != nil {
				return errors.Wrap(err, "failed to stream logs")
			}
			fmt.Println(podLogSeparator)
		}
	}
	return nil
}

// TODO(chuckha) the output is a little confusing because our containers already produce structured logs.

func streamLogs(client kubernetes.Interface, namespace, podName string, logOptions *v1.PodLogOptions) error {
	req := client.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	readCloser, err := req.Stream()
	if err != nil {
		return errors.Wrap(err, "could not stream the request")
	}
	defer readCloser.Close()
	_, err = io.Copy(os.Stdout, readCloser)
	return errors.Wrap(err, "could not copy request body")
}
