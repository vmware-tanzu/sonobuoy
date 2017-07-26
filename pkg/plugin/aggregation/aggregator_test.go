/*
Copyright 2017 Heptio Inc.

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
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestAggregation(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
	}
	// Happy path
	withAggregator(t, expected, func(agg *Aggregator) {
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs", "foo")
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got (%v) response from server: %v", resp.StatusCode, string(body))
		}

		if result, ok := agg.Results["systemd_logs/node1"]; ok {
			bytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path()) + ".json")
			if string(bytes) != "foo" {
				t.Errorf("results for node1 incorrect (got %v): %v", string(bytes), err)
			}
		} else {
			t.Errorf("AggregationServer didn't record a result for node1")
		}
	})
}

func TestAggregation_wrongnodes(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
	}

	withAggregator(t, expected, func(agg *Aggregator) {
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node10/systemd_logs", "foo")
		if resp.StatusCode != 403 {
			t.Errorf("Expected a 403 forbidden for checking in an unexpected node, got %v", resp.StatusCode)
		}

		if _, ok := agg.Results["node10"]; ok {
			t.Fatal("Aggregator accepted a result from an unexpected host")
			t.Fail()
		}
	})
}

func TestAggregation_duplicates(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
		plugin.ExpectedResult{NodeName: "node12", ResultType: "systemd_logs"},
	}
	withAggregator(t, expected, func(agg *Aggregator) {
		// Check in a node
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs", "foo")
		if resp.StatusCode != 200 {
			t.Errorf("Got non-200 response from server: %v", resp.StatusCode)
		}

		// Check in the same node again, should conflict
		resp = doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs", "foo")
		if resp.StatusCode != 409 {
			t.Errorf("Expected a 409 conflict for checking in a duplicate node, got %v", resp.StatusCode)
		}

		if _, ok := agg.Results["node10"]; ok {
			t.Fatal("Aggregator accepted a result from an unexpected host")
			t.Fail()
		}
	})
}

func withAggregator(t *testing.T, expected []plugin.ExpectedResult, callback func(*Aggregator)) {
	dir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatal("Could not create temp directory")
		t.FailNow()
		return
	}
	defer os.RemoveAll(dir)

	agg := NewAggregator(dir, expected)
	srv := NewServer(":"+strconv.Itoa(testPort), agg.HandleHTTPResult)

	// Run the server, ensuring it's fully stopped before returning
	done := make(chan error)
	go func() {
		done <- srv.Start()
	}()
	defer func() {
		srv.Stop()
		<-done
	}()

	srv.WaitUntilReady()
	callback(agg)
}
