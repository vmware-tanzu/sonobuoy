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
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	localProgressURLPath = "/progress"
)

func init() {
	err := mime.AddExtensionType(".gz", "application/gzip")
	if err != nil {
		logrus.Error(err)
	}
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
	// TODO(chuckha) evaluate wait.Until [https://github.com/kubernetes/apimachinery/blob/e9ff529c66f83aeac6dff90f11ea0c5b7c4d626a/pkg/util/wait/wait.go]
	for {
		select {
		case <-ticker.C:
			if resultFile, err := ioutil.ReadFile(waitfile); err == nil {
				resultFile = bytes.TrimSpace(resultFile)
				logrus.WithField("resultFile", string(resultFile)).Info("Detected done file, transmitting result file")
				return handleWaitFile(string(resultFile), url, client)
			}
		case <-stopc:
			logrus.Info("Did not receive plugin results in time. Shutting down worker.")
			return nil
		}
	}
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
