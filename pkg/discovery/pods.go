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

package discovery

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/config"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// PodLogsLocation is the location within the results tarball where pod
	// information is stored.
	PodLogsLocation = "podlogs"
)

// mapping to k8s.io/api/core/v1/PodLogOptions.
func getPodLogOptions(cfg *config.Config) *v1.PodLogOptions {
	podLogLimits := &cfg.Limits.PodLogs

	options := &v1.PodLogOptions{
		Previous:     podLogLimits.Previous,
		SinceSeconds: podLogLimits.SinceSeconds,
		SinceTime:    podLogLimits.SinceTime,
		Timestamps:   podLogLimits.Timestamps,
		TailLines:    podLogLimits.TailLines,
		LimitBytes:   podLogLimits.LimitBytes,
	}

	// TODO: investigate using LimitBytes (SizeLimitBytes is deprecated)
	//       and SinceSeconds (TimeLimitDuration is deprecated)
	// Only set values if they have values greater than 0 (as in they user specified).
	limitBytes := podLogLimits.SizeLimitBytes(0)                           //nolint:staticcheck
	sinceSeconds := int64(podLogLimits.TimeLimitDuration(0) / time.Second) //nolint:staticcheck
	if limitBytes > 0 {
		options.LimitBytes = &limitBytes
	}
	if sinceSeconds > 0 {
		options.SinceSeconds = &sinceSeconds
	}

	return options
}

// gatherPodLogs will loop through collecting pod logs and placing them into a directory tree
// If ns is not provided,  meaning candidate pods can come from all namespaces
// visitedPods will eliminate duplicate pods when execute overlapping queries,
// e.g. query by namespaces and query by fieldSelectors
func gatherPodLogs(kubeClient kubernetes.Interface, ns string, opts metav1.ListOptions, cfg *config.Config,
	visitedPods map[string]struct{}) error {

	// 1 - Collect the list of pods
	podlist, err := kubeClient.CoreV1().Pods(ns).List(context.TODO(), opts)
	if err != nil {
		return errors.WithStack(err)
	}

	podLogOptions := getPodLogOptions(cfg)

	// 2 - Foreach pod, dump each of its containers' logs in a tree in the following location:
	//   pods/:podname/logs/:containername.txt
	for _, pod := range podlist.Items {
		if _, ok := visitedPods[pod.SelfLink]; ok {
			continue // skip visited pods
		}
		visitedPods[pod.SelfLink] = struct{}{}

		if pod.Status.Phase == v1.PodFailed && pod.Status.Reason == "Evicted" {
			logrus.WithField("podName", pod.Name).Info("Skipping evicted pod.")
			continue
		}
		for _, container := range pod.Spec.Containers {
			podLogOptions.Container = container.Name
			body, err := kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions).DoRaw(context.TODO())
			if err != nil {
				return errors.WithStack(err)
			}
			outdir := path.Join(cfg.OutputDir(), PodLogsLocation, pod.Namespace, pod.Name, "logs")
			if err = os.MkdirAll(outdir, 0755); err != nil {
				return errors.WithStack(err)
			}

			outfile := path.Join(outdir, container.Name) + ".txt"
			if err = ioutil.WriteFile(outfile, body, 0644); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}
