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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	testhook "github.com/sirupsen/logrus/hooks/test"
)

func TestErrorRequestRetry(t *testing.T) {
	tests := []struct {
		name string
		f    func() (io.Reader, string, string, error)
	}{
		{
			name: "error request retry",
			f: func() (io.Reader, string, string, error) {
				return nil, "fakefile", "", errors.New("didn't succeed")
			},
		},
		{
			name: "success request retry",
			f: func() (io.Reader, string, string, error) {
				return bytes.NewBuffer([]byte("success!")), "fakefile", "success!", nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testServer := &testServer{
				responseCodes: []int{500, 200},
			}

			server := httptest.NewTLSServer(testServer)
			defer server.Close()

			err := DoRequest(server.URL, server.Client(), test.f)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if testServer.responseCount != 2 {
				t.Errorf("expected 2 requests, got %d", testServer.responseCount)
			}
		})
	}
}

func TestDoRequest_Headers(t *testing.T) {
	tests := []struct {
		name            string
		f               func() (io.Reader, string, string, error)
		expectedHeaders map[string]string
	}{
		{
			name: "filename and type",
			f: func() (io.Reader, string, string, error) {
				return nil, "myfile.xyz", "mytype", nil
			},
			expectedHeaders: map[string]string{
				"content-type":        "mytype",
				"content-disposition": "attachment;filename=myfile.xyz",
			},
		},
		{
			name: "type without filename",
			f: func() (io.Reader, string, string, error) {
				return nil, "", "mytype", nil
			},
			expectedHeaders: map[string]string{
				"content-type":        "mytype",
				"content-disposition": "attachment",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				for k, expect := range test.expectedHeaders {
					real := r.Header.Get(k)
					if real != expect {
						t.Errorf("Expected header %q to have value %q but got %q", k, expect, real)
					}
				}
			}

			server := httptest.NewTLSServer(http.HandlerFunc(handler))
			defer server.Close()

			err := DoRequest(server.URL, server.Client(), test.f)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

type testServer struct {
	sync.Mutex
	responseCodes []int
	responseCount int
}

func (t *testServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.Lock()
	defer t.Unlock()

	responseCode := 500

	if len(t.responseCodes) > 0 {
		responseCode, t.responseCodes = t.responseCodes[0], t.responseCodes[1:]
	}

	w.WriteHeader(responseCode)
	w.Write([]byte("ok!"))

	t.responseCount++
}

func TestDoRequestLogsMessagesAndRetries(t *testing.T) {
	testHook := &testhook.Hook{}
	logrus.AddHook(testHook)
	logrus.SetOutput(io.Discard)

	callback := func() (io.Reader, string, string, error) {
		return strings.NewReader("testReader"), "fakefilename", "testString", nil
	}

	passAfterN := func(i int) http.Handler {
		count := 0
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			count++
			if count > i {
				return
			}
			// Only retries certain types of responses/errors. Be careful if you change this code.
			w.WriteHeader(http.StatusInternalServerError)
		})
	}

	testCases := []struct {
		desc             string
		handler          http.Handler
		expectedLogs     int
		expectFinalError bool
	}{
		{
			desc:    "No errors is OK",
			handler: passAfterN(0),
		}, {
			desc:         "First err leads to one log",
			handler:      passAfterN(1),
			expectedLogs: 1,
		}, {
			desc:         "Multiple err leads to more logs",
			handler:      passAfterN(2),
			expectedLogs: 2,
		}, {
			desc:             "Retries stop after 3 failures",
			handler:          passAfterN(3),
			expectedLogs:     3,
			expectFinalError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testHook.Reset()
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()

			err := DoRequest(ts.URL, ts.Client(), callback)
			if err != nil && !tc.expectFinalError {
				t.Errorf("Expected no error to bubble up but got %v", err)
			} else if err == nil && tc.expectFinalError {
				t.Error("Expected an error to bubble up but got none")
			}

			if len(testHook.Entries) != tc.expectedLogs {
				t.Errorf("Expected %v logs entries but got %v. Logs:", tc.expectedLogs, len(testHook.Entries))
				for _, v := range testHook.Entries {
					t.Logf("%v %v %v", v.Time, v.Level, v.Message)
				}
			}
		})
	}
}
