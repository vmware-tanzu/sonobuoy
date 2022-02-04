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

package worker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/tarball"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	localProgressURLPath = "/progress"

	SonobuoyNSEnvKey              = "SONOBUOY_NS"
	SonobuoyPluginPodEnvKey       = "SONOBUOY_PLUGIN_POD"
	SonobuoyWorkerContainerEnvKey = "SONOBUOY_WORKER_CONTAINER"

	// defaultWaitFileConsumptionDelay provides a minimum time window where sidecars may intercept results and remove
	// the done file before Sonobuoy decides to process it.
	defaultWaitFileConsumptionDelay = 5 * time.Second

	// SonobuoyDoneFileDelayKey is the env key specifying where this value may be overwritten. Adding this in as a mitigation for problems
	// caused by adding the delay. I anticipate we will not have a problem though since the default delay is reasonably short.
	SonobuoyDoneFileDelayKey = "SONOBUOY_DONE_FILE_DELAY"
)

var (
	// apiClient is used to check the status of this pod and other containers in it. If it is
	// nil, those checks will be skipped.
	apiClient kubernetes.Interface

	// waitFileConsumptionDelay provides a minimum time window where sidecars may intercept results and remove
	// the done file before Sonobuoy decides to process it.
	waitFileConsumptionDelay = defaultWaitFileConsumptionDelay
)

func init() {
	err := mime.AddExtensionType(".gz", "application/gzip")
	if err != nil {
		logrus.Error(err)
	}

	if v := os.Getenv(SonobuoyDoneFileDelayKey); len(v) > 0 {
		if d, err := time.ParseDuration(v); err != nil {
			logrus.Errorf("Failed to parse duration value (%v) %v: %v", SonobuoyDoneFileDelayKey, v, err)
			logrus.Infof("Using default %v of %v", SonobuoyDoneFileDelayKey, defaultWaitFileConsumptionDelay)
		} else {
			waitFileConsumptionDelay = d
		}
	}
}

func initAPIClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate in-cluster client config, unable to tar results without donefile: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate in-cluster api-client, unable to tar results without donefile: %w", err)
	}
	return clientset, nil
}

// RelayProgressUpdates start listening to the given port and will use the client to post progressUpdates
// to the aggregatorURL.
func RelayProgressUpdates(port string, aggregatorURL string, client *http.Client) {
	http.HandleFunc(localProgressURLPath, relayProgress(aggregatorURL, client))
	logrus.Infof("Starting to listen on port %v for progress updates and will relay them to %v", port, aggregatorURL)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		logrus.Errorf("Error listening on port %q: %v", port, err)
	}
}

// relayProgress returns a closure which is an http.Handler which is capable of relaying the
// progress updates it gets to the aggregatorURL.
func relayProgress(aggregatorURL string, client *http.Client) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest(http.MethodPost, aggregatorURL, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logrus.Errorf("Failed to create progress update request for the aggregator: %v", err)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logrus.Errorf("Failed to send progress update to aggregator: %v", err)
			return
		}
		w.WriteHeader(resp.StatusCode)
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			logrus.Errorf("Failed to copy aggregator response to plugin progress: %v", err)
			return
		}
	}
}

// GatherResults is the consumer of a co-scheduled container that agrees on the following
// contract:
//
// 1. Output data will be placed into an agreed upon results directory.
// 2. The Job will wait for a done file
// 3. The done file contains a single string of the results to be sent to the aggregator
func GatherResults(waitfile string, url string, client *http.Client, stopc <-chan struct{}) error {
	logrus.WithField("waitfile", waitfile).Info("Waiting for waitfile")
	ticker := time.NewTicker(time.Duration(1) * time.Second)
	containerTicker, assumeResults := time.NewTicker(time.Duration(5)*time.Second), false

	var err error
	apiClient, err = initAPIClient()
	if err != nil {
		logrus.Errorln(err)
		containerTicker.Stop()
	}

	// TODO(chuckha) evaluate wait.Until [https://github.com/kubernetes/apimachinery/blob/e9ff529c66f83aeac6dff90f11ea0c5b7c4d626a/pkg/util/wait/wait.go]
	for {
		select {
		case <-ticker.C:
			if resultFile, err := ioutil.ReadFile(waitfile); err == nil {
				logrus.Tracef("Detected done file but sleeping for %v then checking again for file. This allows other containers to intervene if desired.", waitFileConsumptionDelay)
				time.Sleep(waitFileConsumptionDelay)
				if _, err := os.Stat(waitfile); err != nil {
					logrus.Trace("Done file has been removed, potentially by a sidecar for postprocessing reasons. Resuming wait routine.")
					continue
				}
				resultFile = bytes.TrimSpace(resultFile)
				logrus.WithField("resultFile", string(resultFile)).Info("Detected done file, transmitting result file")
				return handleWaitFile(string(resultFile), url, client)
			}
		case <-stopc:
			logrus.Info("Did not receive plugin results in time. Shutting down worker.")
			return nil
		case <-containerTicker.C:
			// Check if worker is the only container left running (and the others stopped as completed).
			// If so, just report the whole results directory as results.
			if assumeResults {
				dir := filepath.Dir(waitfile)
				tarballPath := filepath.Join(dir, "out.tar.gz")
				logrus.Infof("All other containers have completed but not results have been submitted. Tarring up results directory (%v) as submitted automatically.", dir)
				if err := tarball.DirToTarball(dir, tarballPath, true); err != nil {
					logrus.Errorf("Failed to tar directory %v: %v", dir, err)
				}
				if err := os.WriteFile(waitfile, []byte(tarballPath), 0644); err != nil {
					logrus.Errorf("Failed to write donefile (%v) with contents %q: %v", waitfile, tarballPath, err)
				}
				containerTicker.Stop()
			}
			if !assumeResults && allOtherContainersCompleted(os.Getenv(SonobuoyNSEnvKey), os.Getenv(SonobuoyPluginPodEnvKey), os.Getenv(SonobuoyWorkerContainerEnvKey)) {
				// Don't be too quick to return everything; give a few ticks to avoid a race.
				assumeResults = true
			}
		}
	}
}

func allOtherContainersCompleted(myNS, myPod, myContainer string) bool {
	if apiClient == nil {
		return false
	}
	pod, err := apiClient.CoreV1().Pods(myNS).Get(context.TODO(), myPod, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("Failed to get pod that worker is running within: %v", err)
		return false
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == myContainer {
			continue
		}

		if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.ExitCode == 0 {
			continue
		} else if cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0 {
			continue
		} else {
			return false
		}
	}

	return true
}

func handleWaitFile(resultFile, url string, client *http.Client) error {
	var outfile *os.File
	var err error

	// Set content type
	extension := filepath.Ext(resultFile)
	mimeType := mime.TypeByExtension(extension)

	defer func() {
		if outfile != nil {
			outfile.Close()
		}
	}()

	// transmit back the results file.
	return DoRequest(url, client, func() (io.Reader, string, string, error) {
		outfile, err = os.Open(resultFile)
		return outfile, filepath.Base(resultFile), mimeType, errors.WithStack(err)
	})
}
