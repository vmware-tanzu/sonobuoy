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
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
	pluginutils "github.com/heptio/sonobuoy/pkg/plugin/driver/utils"
	"github.com/viniciuschiele/tarx"
)

func TestAggregation(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
	}
	// Happy path
	withAggregator(t, expected, func(agg *Aggregator) {
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs.json", []byte("foo"))
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
			t.Errorf("AggregationServer didn't record a result for node1. Got: %+v", agg.Results)
		}
	})
}

func TestAggregation_noExtension(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
	}

	withAggregator(t, expected, func(agg *Aggregator) {
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs", []byte("foo"))
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got (%v) response from server: %v", resp.StatusCode, string(body))
		}

		if result, ok := agg.Results["systemd_logs/node1"]; ok {
			bytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path()))
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
		plugin.ExpectedResult{ResultType: "e2e"},
	}

	fileBytes := []byte("foo")
	tarBytes := makeTarWithContents(t, "inside_tar.txt", fileBytes)

	withAggregator(t, expected, func(agg *Aggregator) {
		resp := doRequest(t, "PUT", "/api/v1/results/global/e2e.tar.gz", tarBytes)
		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			t.Errorf("Got (%v) response from server: %v", resp.StatusCode, string(body))
		}

		if result, ok := agg.Results["e2e"]; ok {
			realBytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path(), "inside_tar.txt"))
			if bytes.Compare(realBytes, fileBytes) != 0 || err != nil {
				t.Logf("results e2e tests incorrect (got %v, expected %v): %v", string(realBytes), string(fileBytes), err)
				output, _ := exec.Command("ls", "-lR", agg.OutputDir).CombinedOutput()
				t.Log(string(output))
				t.Fail()
			}
		} else {
			t.Errorf("AggregationServer didn't record a result for e2e tests. Got: %+v", agg.Results)
		}
	})
}

func TestAggregation_wrongnodes(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
	}

	withAggregator(t, expected, func(agg *Aggregator) {
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node10/systemd_logs.json", []byte("foo"))
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
		plugin.ExpectedResult{NodeName: "node1", ResultType: "systemd_logs"},
		plugin.ExpectedResult{NodeName: "node12", ResultType: "systemd_logs"},
	}
	withAggregator(t, expected, func(agg *Aggregator) {
		// Check in a node
		resp := doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs.json", []byte("foo"))
		if resp.StatusCode != 200 {
			t.Errorf("Got non-200 response from server: %v", resp.StatusCode)
		}

		// Check in the same node again, should conflict
		resp = doRequest(t, "PUT", "/api/v1/results/by-node/node1/systemd_logs.json", []byte("foo"))
		if resp.StatusCode != 409 {
			t.Errorf("Expected a 409 conflict for checking in a duplicate node, got %v", resp.StatusCode)
		}

		if _, ok := agg.Results["node10"]; ok {
			t.Fatal("Aggregator accepted a result from an unexpected host")
		}
	})
}

func TestAggregation_errors(t *testing.T) {
	expected := []plugin.ExpectedResult{
		plugin.ExpectedResult{ResultType: "e2e"},
	}

	withAggregator(t, expected, func(agg *Aggregator) {
		resultsCh := make(chan *plugin.Result)
		go agg.IngestResults(resultsCh)

		// Send an error
		resultsCh <- pluginutils.MakeErrorResult("e2e", map[string]interface{}{"error": "foo"}, "")
		agg.Wait(make(chan bool))

		if result, ok := agg.Results["e2e"]; ok {
			bytes, err := ioutil.ReadFile(path.Join(agg.OutputDir, result.Path()) + ".json")
			if err != nil || string(bytes) != `{"error":"foo"}` {
				t.Errorf("results for e2e plugin incorrect (got %v): %v", string(bytes), err)
			}
		} else {
			t.Errorf("Aggregator didn't record error result from e2e plugin, got %v", agg.Results)
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
		t.Fatal("Could not create results directory %v: %v", tardir, err)
		return
	}

	filepath := path.Join(tardir, filename)
	tarfile := path.Join(dir, "results.tar.gz")

	err = ioutil.WriteFile(filepath, fileContents, 0644)
	if err != nil {
		t.Fatalf("Could not write to temp file %v: %v", filepath, err)
		return
	}

	err = tarx.Compress(tarfile, tardir, &tarx.CompressOptions{Compression: tarx.Gzip})
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
