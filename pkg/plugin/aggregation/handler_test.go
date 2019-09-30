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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/backplane/ca/authtest"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"

	"github.com/kylelemons/godebug/pretty"
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
	}, func(status plugin.ProgressUpdate, w http.ResponseWriter) {
		return
	})

	srv := authtest.NewTLSServer(h, t)
	defer srv.Close()

	// Expect a 404 and no results
	response := doRequest(t, srv.Client(), "PUT", srv.URL+"/not/found", expectedJSON)
	if response.StatusCode != 404 {
		t.Fatalf("Expected a 404 response, got %v", response.StatusCode)
		t.Fail()
	}
	if len(checkins) > 0 {
		t.Fatalf("Request to a wrong URL should not have resulted in a node check-in")
		t.Fail()
	}

	URL, err := NodeResultURL(srv.URL, "testnode", "systemd_logs")
	if err != nil {
		t.Fatalf("error getting global result URL %v", err)
	}

	// PUT is all that is accepted
	response = doRequest(t, srv.Client(), "POST", URL, expectedJSON)
	if response.StatusCode != 405 {
		t.Fatalf("Expected a 405 response, got %v", response.StatusCode)
		t.Fail()
	}
	if len(checkins) > 0 {
		t.Fatalf("Request with wrong HTTP method should not have resulted in a node check-in")
		t.Fail()
	}

	// Happy path
	response = doRequest(t, srv.Client(), "PUT", URL, expectedJSON)
	if response.StatusCode != 200 {
		t.Fatalf("Client got non-200 status from server: %v", response.StatusCode)
		t.Fail()
	}

	if _, ok := checkins[expectedResult]; !ok {
		t.Fatalf("Valid request for %v did not get recorded", expectedResult)
		t.Fail()
	}

	URL, err = GlobalResultURL(srv.URL, "gztest")
	if err != nil {
		t.Fatalf("error getting global result URL %v", err)
	}

	headers := http.Header{}
	headers.Set("content-type", gzipMimeType)

	response = doRequestWithHeaders(t, srv.Client(), "PUT", URL, expectedJSON, headers)
	if response.StatusCode != 200 {
		t.Fatalf("Client got non-200 status from server: %v", response.StatusCode)
	}
	if _, ok := checkins[expectedResult]; !ok {
		t.Fatalf("Valid request for %v did not get recorded", expectedResult)
	}

	expectedResult = "gztest/results/global"

	// Happy path with gzip
	res, ok := checkins[expectedResult]
	if !ok {
		t.Fatalf("Valid request for %v did not get recorded; have %v", expectedResult, checkins)
	}
	if res.MimeType != gzipMimeType {
		t.Fatalf("expected mime type %s, got %s", gzipMimeType, res.MimeType)
	}
}

func doRequestWithHeaders(t *testing.T, client *http.Client, method, reqURL string, body []byte, headers http.Header) *http.Response {
	req, err := http.NewRequest(
		method,
		reqURL,
		bytes.NewReader(body),
	)
	req.Header = headers
	if err != nil {
		t.Fatalf("error constructing request: %v", err)
		t.Fail()
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("error performing request: %v", err)
		t.Fail()
	}
	return resp
}

func doRequest(t *testing.T, client *http.Client, method, reqURL string, body []byte) *http.Response {
	return doRequestWithHeaders(t, client, method, reqURL, body, http.Header{})
}

func TestResultFromRequest(t *testing.T) {
	reqWithHeaders := func(headerMap map[string]string) *http.Request {
		r, err := http.NewRequest(http.MethodPost, "url", nil)
		if err != nil {
			t.Fatalf("Failed to make test request object: %v", err)
		}
		for k, v := range headerMap {
			r.Header.Set(k, v)
		}
		return r
	}

	tcs := []struct {
		desc     string
		req      *http.Request
		vars     map[string]string
		expected *plugin.Result
	}{
		{
			desc: "Reads vars as expected",
			vars: map[string]string{
				"plugin": "pluginvar",
				"node":   "nodevar",
			},
			req: reqWithHeaders(map[string]string{
				"content-type":        "test-type",
				"content-disposition": "attachment;filename=foo.txt",
			}),
			expected: &plugin.Result{
				ResultType: "pluginvar",
				NodeName:   "nodevar",
				Filename:   "foo.txt",
				MimeType:   "test-type",
			},
		}, {
			desc: "Defaults node name if not given",
			vars: map[string]string{
				"plugin": "pluginvar",
			},
			req: reqWithHeaders(map[string]string{
				"content-type":        "test-type",
				"content-disposition": "attachment;filename=foo.txt",
			}),
			expected: &plugin.Result{
				ResultType: "pluginvar",
				NodeName:   "global",
				Filename:   "foo.txt",
				MimeType:   "test-type",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out := resultFromRequest(tc.req, tc.vars)
			if !reflect.DeepEqual(out, tc.expected) {
				t.Errorf("Expected %+v but got %+v", tc.expected, out)
			}
		})
	}
}

func TestFilenameFromHeader(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		expect string
	}{
		{
			desc:   "No filename leads to default",
			input:  "attachment",
			expect: "result",
		}, {
			desc:   "Filename in content-disposition is used",
			input:  "attachment;filename=foo",
			expect: "foo",
		}, {
			desc:   "No header leads to default",
			input:  "",
			expect: "result",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out := filenameFromHeader(tc.input)
			if out != tc.expect {
				t.Errorf("Expected %q but got %q", tc.expect, out)
			}
		})
	}
}

func TestProgressFromRequest(t *testing.T) {
	tcs := []struct {
		desc        string
		input       io.Reader
		vars        map[string]string
		expected    *plugin.ProgressUpdate
		expectedErr string
	}{
		{
			desc:  "Handles body and muxvars",
			input: strings.NewReader(`{"msg":"foo"}`),
			vars: map[string]string{
				"node":   "nodename",
				"plugin": "pluginname",
			},
			expected: &plugin.ProgressUpdate{
				Message:    "foo",
				PluginName: "pluginname",
				Node:       "nodename",
			},
		}, {
			desc:  "Missing node treated as global",
			input: strings.NewReader(`{"msg":"foo"}`),
			vars: map[string]string{
				"plugin": "pluginname",
			},
			expected: &plugin.ProgressUpdate{
				Message:    "foo",
				PluginName: "pluginname",
				Node:       "global",
			},
		}, {
			desc:  "Failure to decode value reported",
			input: strings.NewReader(``),
			vars: map[string]string{
				"plugin": "pluginname",
			},
			expected: &plugin.ProgressUpdate{
				PluginName: "pluginname",
				Node:       "global",
			},
			expectedErr: "unable to decode body: EOF",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "www.example.com", tc.input)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			update, err := progressFromRequest(req, tc.vars)
			if len(tc.expectedErr) == 0 && err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if len(tc.expectedErr) > 0 {
				if err == nil {
					t.Errorf("Expected error %v but got no error", tc.expectedErr)
				} else if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("Expected error to contain '%v', got '%v'", tc.expectedErr, err.Error())
				}
			}

			// Can't predict timestamps; just make sure its non-zero and empty it.
			if update.Timestamp.IsZero() {
				t.Errorf("Expected non-zero time to be set but got %v", update.Timestamp)
			}
			update.Timestamp = time.Time{}

			if diff := pretty.Compare(tc.expected, update); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
