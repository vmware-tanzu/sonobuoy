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
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestStart(t *testing.T) {
	checkins := make(map[string]*plugin.Result, 0)

	expectedResult := "systemd_logs/results/testnode"
	expectedJSON := []byte(`{"some": "json"}`)

	tmpdir, err := ioutil.TempDir("", "sonobuoy_server_test")
	if err != nil {
		t.Fatal("Could not create temp directory")
		t.FailNow()
		return
	}
	defer os.RemoveAll(tmpdir)

	h := NewHandler(func(checkin *plugin.Result, w http.ResponseWriter) {
		// Just take note of what we've received
		checkins[checkin.Path()] = checkin
	})

	srv := httptest.NewServer(h)
	defer srv.Close()

	// Expect a 404 and no results
	response := doRequest(t, srv, "PUT", "/not/found", expectedJSON)
	if response.StatusCode != 404 {
		t.Fatalf("Expected a 404 response, got %v", response.StatusCode)
		t.Fail()
	}
	if len(checkins) > 0 {
		t.Fatalf("Request to a wrong URL should not have resulted in a node check-in")
		t.Fail()
	}

	// PUT is all that is accepted
	response = doRequest(t, srv, "POST", "/api/v1/results/by-node/testnode/systemd_logs", expectedJSON)
	if response.StatusCode != 405 {
		t.Fatalf("Expected a 405 response, got %v", response.StatusCode)
		t.Fail()
	}
	if len(checkins) > 0 {
		t.Fatalf("Request with wrong HTTP method should not have resulted in a node check-in")
		t.Fail()
	}

	// Happy path
	response = doRequest(t, srv, "PUT", "/api/v1/results/by-node/testnode/systemd_logs", expectedJSON)
	if response.StatusCode != 200 {
		t.Fatalf("Client got non-200 status from server: %v", response.StatusCode)
		t.Fail()
	}

	if _, ok := checkins[expectedResult]; !ok {
		t.Fatalf("Valid request for %v did not get recorded", expectedResult)
		t.Fail()
	}
}

func doRequest(t *testing.T, srv *httptest.Server, method, path string, body []byte) *http.Response {
	// Make a new HTTP transport for every request, this avoids issues where HTTP
	// connection keep-alive leaves connections running to old server instances.
	// (We can take the performance hit since it's just tests.)
	req, err := http.NewRequest(
		method,
		srv.URL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("error constructing request: %v", err)
		t.Fail()
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("error performing request: %v", err)
		t.Fail()
	}
	return resp
}
