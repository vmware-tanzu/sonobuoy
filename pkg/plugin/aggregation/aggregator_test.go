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

package aggregation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/vmware-tanzu/sonobuoy/pkg/backplane/ca/authtest"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	pluginutils "github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/utils"
	"github.com/vmware-tanzu/sonobuoy/pkg/tarball"
)

func TestAggregation(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd_logs"},
	}

	withAggregator(t, expected, func(agg *Aggregator, srv *authtest.Server) {
		URL, err := NodeResultURL(srv.URL, "node1", "systemd_logs")
		if err != nil {
			t.Fatalf("couldn't get test server URL: %v", err)
		}

		resp := doRequest(t, srv.Client(), "PUT", URL, []byte("foo"))
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got (%v) response from server: %v", resp.StatusCode, string(body))
		}

		if result, ok := agg.Results["systemd_logs/node1"]; ok {
			bytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path(), defaultFilename))
			if string(bytes) != "foo" {
				t.Errorf("results for node1 incorrect (got %v): %v", string(bytes), err)
			}
		} else {
			t.Errorf("AggregationServer didn't record a result for node1. Got: %+v", agg.Results)
		}
	})
}

func TestAggregation_noExtension(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd_logs"},
	}

	withAggregator(t, expected, func(agg *Aggregator, srv *authtest.Server) {
		URL, err := NodeResultURL(srv.URL, "node1", "systemd_logs")
		if err != nil {
			t.Fatalf("couldn't get test server URL: %v", err)
		}
		resp := doRequest(t, srv.Client(), "PUT", URL, []byte("foo"))
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got (%v) response from server: %v", resp.StatusCode, string(body))
		}

		if result, ok := agg.Results["systemd_logs/node1"]; ok {
			bytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path(), defaultFilename))
			if string(bytes) != "foo" {
				t.Errorf("results for node1 incorrect (got %v): %v", string(bytes), err)
			}
		} else {
			t.Errorf("AggregationServer didn't record a result for node1. Got: %+v", agg.Results)
		}
	})
}

func TestAggregation_tarfile(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{ResultType: "e2e", NodeName: "global"},
	}

	fileBytes := []byte("foo")
	tarBytes := makeTarWithContents(t, "inside_tar.txt", fileBytes)

	withAggregator(t, expected, func(agg *Aggregator, srv *authtest.Server) {
		URL, err := GlobalResultURL(srv.URL, "e2e")
		if err != nil {
			t.Fatalf("couldn't get test server URL: %v", err)
		}

		headers := http.Header{}
		headers.Add("content-type", "application/gzip")

		resp := doRequestWithHeaders(t, srv.Client(), "PUT", URL, tarBytes, headers)
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got (%v) response from server: %v", resp.StatusCode, string(body))
		}

		if result, ok := agg.Results["e2e/global"]; ok {
			realBytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path(), "inside_tar.txt"))
			if err != nil || bytes.Compare(realBytes, fileBytes) != 0 {
				t.Logf("results e2e tests incorrect (got %v, expected %v): %v", string(realBytes), string(fileBytes), err)
				output, _ := exec.Command("ls", "-lR", agg.OutputDir).CombinedOutput()
				t.Log(string(output))
				t.Fail()
			}
		} else {
			t.Errorf("AggregationServer didn't record a result for e2e tests. Got: %+v", agg.Results)
			for k, v := range agg.Results {
				t.Logf("Result %q: %+v\n", k, v)
			}
		}
	})
}

func TestAggregation_wrongnodes(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd_logs"},
	}

	withAggregator(t, expected, func(agg *Aggregator, srv *authtest.Server) {
		URL, err := NodeResultURL(srv.URL, "randomnodename", "systemd_logs")
		if err != nil {
			t.Fatalf("couldn't get test server URL: %v", err)

		}
		resp := doRequest(t, srv.Client(), "PUT", URL, []byte("foo"))
		if resp.StatusCode != 403 {
			t.Errorf("Expected a 403 forbidden for checking in an unexpected node, got %v", resp.StatusCode)
		}

		if _, ok := agg.Results["systemd_logs/node10"]; ok {
			t.Fatal("Aggregator accepted a result from an unexpected host")
		}
	})
}

func TestAggregation_duplicates(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd_logs"},
		{NodeName: "node12", ResultType: "systemd_logs"},
	}
	withAggregator(t, expected, func(agg *Aggregator, srv *authtest.Server) {
		URL, err := NodeResultURL(srv.URL, "node1", "systemd_logs")
		if err != nil {
			t.Fatalf("couldn't get test server URL: %v", err)

		}
		// Check in a node
		resp := doRequest(t, srv.Client(), "PUT", URL, []byte("foo"))
		if resp.StatusCode != 200 {
			t.Errorf("Got non-200 response from server: %v", resp.StatusCode)
		}

		// Check in the same node again, should conflict
		resp = doRequest(t, srv.Client(), "PUT", URL, []byte("foo"))
		if resp.StatusCode != 409 {
			t.Errorf("Expected a 409 conflict for checking in a duplicate node, got %v", resp.StatusCode)
		}

		if _, ok := agg.Results["node10"]; ok {
			t.Fatal("Aggregator accepted a result from an unexpected host")
		}
	})
}

func TestAggregation_duplicatesWithErrors(t *testing.T) {
	// Setup aggregator with expected results and preload the test data/info
	// that we want to transmit/compare against.
	dir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatalf("Could not create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)
	outpath := filepath.Join(dir, "systemd_logs", "results", "node1", "fakeLogData.txt")
	testDataPath := "./testdata/fakeLogData.txt"
	testinfo, err := os.Stat(testDataPath)
	if err != nil {
		t.Fatalf("Could not stat test file: %v", err)
	}
	testDataReader, err := os.Open(testDataPath)
	if err != nil {
		t.Fatalf("Could not open test data file: %v", err)
	}
	defer testDataReader.Close()

	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd_logs"},
		{NodeName: "node12", ResultType: "systemd_logs"},
	}
	agg := NewAggregator(dir, expected)

	// Send first result and force an error in processing.
	errReader := iotest.TimeoutReader(testDataReader)
	err = agg.processResult(&plugin.Result{Body: errReader, NodeName: "node1", ResultType: "systemd_logs", Filename: "fakeLogData.txt"})
	if err == nil {
		t.Fatal("Expected error processing this due to reading error, instead got nil.")
	}

	// Confirm results are recorded but they are partial results.
	realinfo, err := os.Stat(outpath)
	if err != nil {
		t.Fatalf("Could not stat output file: %v", err)
	}
	if realinfo.Size() == testinfo.Size() {
		t.Fatal("Expected truncated results for first result (simulating error), but got all the data.")
	}

	// Retry the result without an error this time.
	_, err = testDataReader.Seek(0, 0)
	if err != nil {
		t.Fatalf("Could not rewind test data file: %v", err)
	}
	err = agg.processResult(&plugin.Result{Body: testDataReader, NodeName: "node1", ResultType: "systemd_logs", Filename: "fakeLogData.txt"})
	if err != nil {
		t.Errorf("Expected no error processing this result, got %v", err)
	}

	// Confirm the new results overwrite the old ones.
	realinfo, err = os.Stat(outpath)
	if err != nil {
		t.Fatalf("Could not stat output file: %v", err)
	}
	if realinfo.Size() != testinfo.Size() {
		t.Errorf("Expected all the data to be transmitted. Expected data size %v but got %v.", testinfo.Size(), realinfo.Size())
	}
}

// TestAggregation_RetryWindow ensures that the server Wait() method
// gives clients a chance to retry if their results were not processed correctly.
func TestAggregation_RetryWindow(t *testing.T) {
	// Setup aggregator with expected results and preload the test data/info
	// that we want to transmit/compare against.
	dir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatalf("Could not create temp directory: %v", err)
	}
	defer os.RemoveAll(dir)
	testRetryWindow := 1 * time.Second
	testBufferDuration := 200 * time.Millisecond
	expected := []plugin.ExpectedResult{
		{NodeName: "node1", ResultType: "systemd_logs"},
	}

	testCases := []struct {
		desc             string
		postProcessSleep time.Duration
		simulateErr      bool
		expectExtraWait  time.Duration
	}{
		{
			desc:            "Error causes us to wait at least the retry window",
			simulateErr:     true,
			expectExtraWait: testRetryWindow,
		}, {
			desc:             "Retry window is sliding",
			simulateErr:      true,
			postProcessSleep: 500 * time.Millisecond,
			expectExtraWait:  500 * time.Millisecond,
		}, {
			desc:             "Retry window can slide to 0",
			simulateErr:      true,
			postProcessSleep: testRetryWindow,
			expectExtraWait:  0,
		}, {
			desc:            "No retry window without error",
			simulateErr:     false,
			expectExtraWait: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			agg := NewAggregator(dir, expected)
			// Shorten retry window for testing.
			agg.retryWindow = testRetryWindow
			testDataPath := "./testdata/fakeLogData.txt"
			testDataReader, err := os.Open(testDataPath)
			if err != nil {
				t.Fatalf("Could not open test data file: %v", err)
			}
			defer testDataReader.Close()

			var r io.Reader
			if tc.simulateErr {
				r = iotest.TimeoutReader(testDataReader)
			} else {
				r = strings.NewReader("foo")
			}

			err = agg.processResult(&plugin.Result{Body: r, NodeName: "node1", ResultType: "systemd_logs"})
			if err == nil && tc.simulateErr {
				t.Fatal("Expected error processing this due to reading error, instead got nil.")
			}
			// check time before/after wait and ensure it is greater than the retryWindow.
			time.Sleep(tc.postProcessSleep)
			start := time.Now()
			agg.Wait(make(chan bool))
			waitTime := time.Now().Sub(start)

			// Add buffer to avoid raciness due to processing time.
			diffTime := waitTime - tc.expectExtraWait
			if diffTime > testBufferDuration || diffTime < -1*testBufferDuration {
				t.Errorf("Expected Wait() to wait the duration %v (+/- %v), instead waited %v", tc.expectExtraWait, testBufferDuration, waitTime)
			}
		})
	}
}

func TestAggregation_errors(t *testing.T) {
	expected := []plugin.ExpectedResult{
		{ResultType: "e2e", NodeName: "global"},
	}

	withAggregator(t, expected, func(agg *Aggregator, srv *authtest.Server) {
		resultsCh := make(chan *plugin.Result)
		go agg.IngestResults(context.TODO(), resultsCh)

		// Send an error
		resultsCh <- pluginutils.MakeErrorResult("e2e", map[string]interface{}{"error": "foo"}, "global")
		agg.Wait(make(chan bool))

		if result, ok := agg.Results["e2e/global"]; ok {
			bytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path(), "error.json"))
			if err != nil || string(bytes) != `{"error":"foo"}` {
				t.Errorf("results for e2e plugin incorrect (got %v): %v", string(bytes), err)
			}
		} else {
			t.Errorf("Aggregator didn't record error result from e2e plugin, got %v", agg.Results)
		}
	})
}

func TestProcessProgressUpdates(t *testing.T) {
	defaultExpectedResults := []plugin.ExpectedResult{
		{ResultType: "type1", NodeName: "global"},
		{ResultType: "type2", NodeName: "node1"},
		{ResultType: "type2", NodeName: "node2"},
	}

	type singleProgressEvent struct {
		progressUpdate          plugin.ProgressUpdate
		expectedErr             string
		expectedHTTPCode        int
		expectedProgressUpdates map[string]plugin.ProgressUpdate
	}

	testCases := []struct {
		desc            string
		expectedResults []plugin.ExpectedResult

		// each test run will loop through processing a series of updates
		events []singleProgressEvent
	}{
		{
			desc:            "Unexpected progress update results in Forbidden",
			expectedResults: defaultExpectedResults,
			events: []singleProgressEvent{
				{
					progressUpdate:   plugin.ProgressUpdate{PluginName: "foo", Node: "bar"},
					expectedHTTPCode: http.StatusForbidden,
					expectedErr:      "progress update for foo/bar unexpected",
				},
			},
		}, {
			desc:            "Latest updates get updated even with errors before and after",
			expectedResults: defaultExpectedResults,
			events: []singleProgressEvent{
				{
					progressUpdate:          plugin.ProgressUpdate{PluginName: "type1", Node: "global", Message: "hi"},
					expectedProgressUpdates: map[string]plugin.ProgressUpdate{"type1/global": {PluginName: "type1", Node: "global", Message: "hi"}},
				},
				{
					progressUpdate:          plugin.ProgressUpdate{PluginName: "foo", Node: "bar"},
					expectedHTTPCode:        http.StatusForbidden,
					expectedErr:             "progress update for foo/bar unexpected",
					expectedProgressUpdates: map[string]plugin.ProgressUpdate{"type1/global": {PluginName: "type1", Node: "global", Message: "hi"}},
				},
				{
					progressUpdate:          plugin.ProgressUpdate{PluginName: "type1", Node: "global", Message: "bye"},
					expectedProgressUpdates: map[string]plugin.ProgressUpdate{"type1/global": {PluginName: "type1", Node: "global", Message: "bye"}},
				},
				{
					progressUpdate:          plugin.ProgressUpdate{PluginName: "foo", Node: "bar"},
					expectedHTTPCode:        http.StatusForbidden,
					expectedErr:             "progress update for foo/bar unexpected",
					expectedProgressUpdates: map[string]plugin.ProgressUpdate{"type1/global": {PluginName: "type1", Node: "global", Message: "bye"}},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			withAggregator(t, tc.expectedResults, func(agg *Aggregator, srv *authtest.Server) {
				for _, v := range tc.events {
					err := agg.processProgressUpdate(v.progressUpdate)
					// Check error string
					if err != nil && len(v.expectedErr) == 0 {
						t.Fatalf("Expected nil error but got %q", err)
					}
					if err == nil && len(v.expectedErr) > 0 {
						t.Fatalf("Expected error %q but got nil", v.expectedErr)
					}
					if err != nil && fmt.Sprint(err) != v.expectedErr {
						t.Fatalf("Expected error to be %q but got %q", v.expectedErr, err)
					}

					// Check http status of the call
					if v.expectedHTTPCode > 0 {
						herr, ok := err.(*httpError)
						if !ok {
							t.Errorf("Expected HTTP error with status %v but got a type %T: %v", v.expectedHTTPCode, err, err)
						} else if herr.HttpCode() != v.expectedHTTPCode {
							t.Errorf("Expected error with HTTP code %v but got %v", v.expectedHTTPCode, herr.HttpCode())
						}
					}

					// Check the result of the update
					if diff := pretty.Compare(agg.LatestProgressUpdates, v.expectedProgressUpdates); diff != "" {
						t.Errorf("Unexpected difference in the LatestProgressUpdates:\n\n%s\n", diff)
					}
				}
			})
		})
	}
}

func withAggregator(t *testing.T, expected []plugin.ExpectedResult, callback func(*Aggregator, *authtest.Server)) {
	dir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatal("Could not create temp directory")
		return
	}
	defer os.RemoveAll(dir)

	agg := NewAggregator(dir, expected)
	handler := NewHandler(agg.HandleHTTPResult, agg.HandleHTTPProgressUpdate)
	srv := authtest.NewTLSServer(handler, t)
	defer srv.Close()

	// Run the server, ensuring it's fully stopped before returning

	callback(agg, srv)
}

// Create a gzipped tar file with the given filename (and contents) inside it,
// return the raw bytes for that tar file.
func makeTarWithContents(t *testing.T, filename string, fileContents []byte) (tarbytes []byte) {
	dir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatalf("Could not create temp directory: %v", err)
		return
	}
	defer os.RemoveAll(dir)

	tardir := path.Join(dir, "results")
	err = os.Mkdir(tardir, 0755)
	if err != nil {
		t.Fatalf("Could not create results directory %v: %v", tardir, err)
		return
	}

	filepath := path.Join(tardir, filename)
	tarfile := path.Join(dir, "results.tar.gz")

	err = ioutil.WriteFile(filepath, fileContents, 0644)
	if err != nil {
		t.Fatalf("Could not write to temp file %v: %v", filepath, err)
		return
	}

	err = tarball.DirToTarball(tardir, tarfile, true)
	if err != nil {
		t.Fatalf("Could not create tar file %v: %v", tarfile, err)
		return
	}

	tarbytes, err = ioutil.ReadFile(tarfile)
	if err != nil {
		t.Fatalf("Could not read created tar file %v: %v", tarfile, err)
		return
	}

	return tarbytes
}
