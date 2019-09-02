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

// This code runs the aggregation server and sends 1000 simultaneous requests
// to it to ensure that it can handle load from large clusters.

package stress

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/heptio/sonobuoy/pkg/backplane/ca/authtest"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
	"github.com/heptio/sonobuoy/pkg/worker"
)

var numResults = 1000
var timeoutSeconds = 10
var bindAddr = ":8080"

func TestStress(t *testing.T) {
	// Create temp dir for results
	dir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatal("Could not create temp directory")
	}
	defer os.RemoveAll(dir)

	// Create expected results for the aggregator to use
	expected := make([]plugin.ExpectedResult, numResults)
	for i := 0; i < numResults; i++ {
		expected[i] = plugin.ExpectedResult{
			NodeName:   "node" + strconv.Itoa(i),
			ResultType: "fake",
		}
	}

	// Launch the aggregator and server
	aggr := aggregation.NewAggregator(dir+"/results", expected)
	handler := aggregation.NewHandler(aggr.HandleHTTPResult, aggr.HandleHTTPProgressUpdate)
	srv := authtest.NewTLSServer(handler, t)

	stopCh := make(chan bool)
	timeoutCh := make(chan bool, 1)
	doneCh := make(chan bool, 1)
	srvDoneCh := make(chan error, 1)
	go func() {
		time.Sleep(time.Duration(timeoutSeconds) * time.Second)
		timeoutCh <- true
	}()
	go func() {
		aggr.Wait(stopCh)
		doneCh <- true
	}()

	sendResults(t, srv.URL, srv.Client(), numResults)

	// Wait for the results to be finished.
	select {
	case err := <-srvDoneCh:
		t.Fatalf("Error running server: %v", err)
	case <-timeoutCh:
		t.Fatalf("Timed out.")
	case <-doneCh:
	}
}

func sendResults(t *testing.T, baseURL string, client *http.Client, n int) {
	// Put <numResults> requests in a channel
	for i := 0; i < n; i++ {
		go func(i int) {
			url := baseURL + "/api/v1/results/by-node/node" + strconv.Itoa(i) + "/fake"
			err := worker.DoRequest(url, client, func() (io.Reader, string, string, error) {
				return bytes.NewReader([]byte("hello")), "fakefile", "", nil
			})
			if err != nil {
				t.Errorf("Error doing request to %v: %v\n", url, err)
			}
		}(i)
	}
}
