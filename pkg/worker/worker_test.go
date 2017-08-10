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

package worker

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
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

	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator) {
		for _, h := range hosts {
			url := "http://:" + strconv.Itoa(aggregatorPort) + "/api/v1/results/by-node/" + h + "/systemd_logs.json"

			withTempDir(t, func(tmpdir string) {
				ioutil.WriteFile(tmpdir+"/systemd_logs", []byte("{}"), 0755)
				ioutil.WriteFile(tmpdir+"/done", []byte(tmpdir+"/systemd_logs"), 0755)
				err := GatherResults(tmpdir+"/done", url)
				if err != nil {
					t.Fatalf("Got error running agent: %v", err)
				}

				ensureExists(t, path.Join(aggr.OutputDir, "systemd_logs", "results", "node1.json"))
			})
		}
	})
}

func TestRunGlobal(t *testing.T) {
	url := "http://:" + strconv.Itoa(aggregatorPort) + "/api/v1/results/global/systemd_logs"

	// Create an expectedResults array
	expectedResults := []plugin.ExpectedResult{
		plugin.ExpectedResult{ResultType: "systemd_logs"},
	}

	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator) {
		withTempDir(t, func(tmpdir string) {
			ioutil.WriteFile(tmpdir+"/systemd_logs.json", []byte("{}"), 0755)
			ioutil.WriteFile(tmpdir+"/done", []byte(tmpdir+"/systemd_logs.json"), 0755)
			err := GatherResults(tmpdir+"/done", url)
			if err != nil {
				t.Fatalf("Got error running agent: %v", err)
			}

			ensureExists(t, path.Join(aggr.OutputDir, "systemd_logs", "results.json"))
		})
	})
}

func TestRunGlobal_noExtension(t *testing.T) {
	url := "http://:" + strconv.Itoa(aggregatorPort) + "/api/v1/results/global/systemd_logs"

	// Create an expectedResults array
	expectedResults := []plugin.ExpectedResult{
		plugin.ExpectedResult{ResultType: "systemd_logs"},
	}

	withAggregator(t, expectedResults, func(aggr *aggregation.Aggregator) {
		withTempDir(t, func(tmpdir string) {
			ioutil.WriteFile(tmpdir+"/systemd_logs", []byte("{}"), 0755)
			ioutil.WriteFile(tmpdir+"/done", []byte(tmpdir+"/systemd_logs"), 0755)
			err := GatherResults(tmpdir+"/done", url)
			if err != nil {
				t.Fatalf("Got error running agent: %v", err)
			}

			ensureExists(t, path.Join(aggr.OutputDir, "systemd_logs", "results"))
		})
	})
}

const aggregatorPort = 8090

func ensureExists(t *testing.T, filepath string) {
	if _, err := os.Stat(filepath); err != nil && os.IsNotExist(err) {
		t.Logf("Plugin agent ran, but couldn't find expected results at %v:", filepath)
		output, _ := exec.Command("ls", "-l", path.Dir(filepath)).CombinedOutput()
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

func withAggregator(t *testing.T, expectedResults []plugin.ExpectedResult, callback func(*aggregation.Aggregator)) {
	withTempDir(t, func(tmpdir string) {
		// Reset the default transport to clear any connection pooling
		http.DefaultTransport = &http.Transport{}

		// Configure the aggregator
		aggr := aggregation.NewAggregator(tmpdir, expectedResults)
		srv := aggregation.NewServer(":"+strconv.Itoa(aggregatorPort), aggr.HandleHTTPResult)

		// Run the aggregation server
		done := make(chan error)
		go func() {
			done <- srv.Start()
		}()
		defer func() {
			srv.Stop()
			<-done
		}()
		srv.WaitUntilReady()

		callback(aggr)
	})
}
