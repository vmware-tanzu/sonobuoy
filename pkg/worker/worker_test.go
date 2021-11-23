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
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/backplane/ca/authtest"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
)

func TestRun(t *testing.T) {
	hosts := []string{"node1", "node2", "node3", "node4", "node5"}

	// Create an expectedResults array
	expectedResults := make([]plugin.ExpectedResult, 0, len(hosts))
	for _, node := range hosts {
		expectedResults = append(expectedResults, plugin.ExpectedResult{
			NodeName:   node,
			ResultType: "systemd_logs",
		})
	}

	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator, srv *authtest.Server) {
		for _, h := range hosts {
			URL, err := aggregation.NodeResultURL(srv.URL, h, "systemd_logs")
			if err != nil {
				t.Fatalf("unexpected error getting node result url %v", err)
			}

			withTempDir(t, func(tmpdir string) {
				ioutil.WriteFile(filepath.Join(tmpdir, "systemd_logs"), []byte("{}"), 0755)
				ioutil.WriteFile(filepath.Join(tmpdir, "done"), []byte(filepath.Join(tmpdir, "systemd_logs")), 0755)
				err := GatherResults(filepath.Join(tmpdir, "done"), URL, srv.Client(), nil)
				if err != nil {
					t.Fatalf("Got error running agent: %v", err)
				}

				ensureExists(t, filepath.Join(aggr.OutputDir, "systemd_logs", "results", "node1"))
			})
		}
	})
}

func TestRunGlobal(t *testing.T) {

	// Create an expectedResults array
	expectedResults := []plugin.ExpectedResult{
		{ResultType: "systemd_logs", NodeName: "global"},
	}

	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator, srv *authtest.Server) {
		url, err := aggregation.GlobalResultURL(srv.URL, "systemd_logs")
		if err != nil {
			t.Fatalf("unexpected error getting global result url %v", err)
		}

		withTempDir(t, func(tmpdir string) {
			ioutil.WriteFile(filepath.Join(tmpdir, "systemd_logs.json"), []byte("{}"), 0755)
			ioutil.WriteFile(filepath.Join(tmpdir, "done"), []byte(filepath.Join(tmpdir, "systemd_logs.json")), 0755)
			err := GatherResults(filepath.Join(tmpdir, "done"), url, srv.Client(), nil)
			if err != nil {
				t.Fatalf("Got error running agent: %v", err)
			}

			ensureExists(t, filepath.Join(aggr.OutputDir, "systemd_logs", "results"))
		})
	})
}

func TestRunGlobal_noExtension(t *testing.T) {

	// Create an expectedResults array
	expectedResults := []plugin.ExpectedResult{
		{ResultType: "systemd_logs", NodeName: "global"},
	}

	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator, srv *authtest.Server) {
		url, err := aggregation.GlobalResultURL(srv.URL, "systemd_logs")
		if err != nil {
			t.Fatalf("unexpected error getting global result url %v", err)
		}
		withTempDir(t, func(tmpdir string) {
			ioutil.WriteFile(filepath.Join(tmpdir, "systemd_logs"), []byte("{}"), 0755)
			ioutil.WriteFile(filepath.Join(tmpdir, "done"), []byte(filepath.Join(tmpdir, "systemd_logs")), 0755)
			err := GatherResults(filepath.Join(tmpdir, "done"), url, srv.Client(), nil)
			if err != nil {
				t.Fatalf("Got error running agent: %v", err)
			}

			ensureExists(t, filepath.Join(aggr.OutputDir, "systemd_logs", "results"))
		})
	})
}

func TestRunGlobalCleanup(t *testing.T) {

	// Create an expectedResults array
	expectedResults := []plugin.ExpectedResult{
		{ResultType: "systemd_logs"},
	}
	stopc := make(chan struct{}, 1)
	stopc <- struct{}{}
	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator, srv *authtest.Server) {
		url, err := aggregation.GlobalResultURL(srv.URL, "systemd_logs")
		if err != nil {
			t.Fatalf("unexpected error getting global result url %v", err)
		}

		withTempDir(t, func(tmpdir string) {
			err := GatherResults(filepath.Join(tmpdir, "done"), url, srv.Client(), stopc)
			if err != nil {
				t.Fatalf("Got error running agent: %v", err)
			}
		})
	})
}

func TestRunCustomDoneFile(t *testing.T) {
	expectedResults := []plugin.ExpectedResult{
		{ResultType: "systemd_logs"},
	}
	stopc := make(chan struct{}, 1)
	stopc <- struct{}{}
	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator, srv *authtest.Server) {
		url, err := aggregation.GlobalResultURL(srv.URL, "systemd_logs")
		if err != nil {
			t.Fatalf("unexpected error getting global result url %v", err)
		}

		withTempDir(t, func(tmpdir string) {
			err := GatherResults(filepath.Join(tmpdir, "customDone"), url, srv.Client(), stopc)
			if err != nil {
				t.Fatalf("Got error running agent: %v", err)
			}
		})
	})
}

func TestRelayProgress(t *testing.T) {
	tcs := []struct {
		desc           string
		input          io.Reader
		expected       string
		expectedStatus int

		serverStatus int
	}{
		{
			desc:           "Copies body to new request",
			input:          strings.NewReader("Hello"),
			expected:       "Hello",
			expectedStatus: http.StatusOK,
		}, {
			desc:           "Copies HTTP status to response",
			serverStatus:   http.StatusTeapot,
			expectedStatus: http.StatusTeapot,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				if tc.serverStatus != 0 {
					w.WriteHeader(tc.serverStatus)
				}
				b, err := ioutil.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("Failed to ready body: %v", err)
				}
				if string(b) != tc.expected {
					t.Errorf("Expected %q but got %q on the server", tc.expected, string(b))
				}
			}
			testingServer := httptest.NewServer(http.HandlerFunc(handler))
			relayer := relayProgress(testingServer.URL, testingServer.Client())
			relayServer := httptest.NewServer(http.HandlerFunc(relayer))
			resp, err := http.Post(relayServer.URL, "application/text", tc.input)
			if err != nil {
				t.Fatalf("Failed to post to relay server: %v", err)
			}
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %v but got %v", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func ensureExists(t *testing.T, checkPath string) {
	if _, err := os.Stat(checkPath); err != nil && os.IsNotExist(err) {
		t.Logf("Plugin agent ran, but couldn't find expected results at %v:", checkPath)
		output, _ := exec.Command("ls", "-l", filepath.Dir(checkPath)).CombinedOutput()
		t.Log(string(output))
		t.Fail()
	}
}

func withTempDir(t *testing.T, callback func(tmpdir string)) {
	// Create a temporary directory for results gathering
	tmpdir, err := ioutil.TempDir("", "sonobuoy_test")
	defer os.RemoveAll(tmpdir)
	if err != nil {
		t.Fatalf("Could not create temporary directory %v: %v", tmpdir, err)
	}

	callback(tmpdir)
}

func withAggregator(t *testing.T, expectedResults []plugin.ExpectedResult, callback func(*aggregation.Aggregator, *authtest.Server)) {
	withTempDir(t, func(tmpdir string) {
		// Reset the default transport to clear any connection pooling
		http.DefaultTransport = &http.Transport{}

		// Configure the aggregator
		aggr := aggregation.NewAggregator(tmpdir, expectedResults)
		handler := aggregation.NewHandler(aggr.HandleHTTPResult, aggr.HandleHTTPProgressUpdate)
		srv := authtest.NewTLSServer(handler, t)
		defer srv.Close()

		callback(aggr, srv)
	})
}
