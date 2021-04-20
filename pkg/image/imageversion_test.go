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

package image

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
)

func TestSetConformanceImageVersion(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)

	tests := []struct {
		name    string
		version string
		expect  string
		error   bool
	}{
		{
			name:    "version detect",
			version: "auto",
			expect:  "auto",
			error:   false,
		},
		{
			name:    "use latest",
			version: "latest",
			expect:  "latest",
			error:   false,
		},
		{
			name:    "random string",
			version: "test",
			error:   true,
		},
		{
			name:    "stable version",
			version: "v1.13.0",
			expect:  "v1.13.0",
			error:   false,
		},
		{
			name:    "version without v",
			version: "1.13.0",
			error:   true,
		},
		{
			name:    "version without patch",
			version: "v1.13",
			expect:  "v1.13.0",
			error:   false,
		},
		{
			name:    "version without minor/patch",
			version: "v1",
			expect:  "v1.0.0",
			error:   false,
		},
		{
			name:    "empty string",
			version: "",
			error:   true,
		},
		{
			name:    "version with prerelease and metadata",
			version: "v1.13.0-beta.2.78+e0b33dbc2bde88",
			expect:  "v1.13.0-beta.2.78+e0b33dbc2bde88",
			error:   false,
		},
		{
			name:    "version with empty metadata",
			version: "v1.11+",
			error:   true,
		},
		{
			name:    "version without patch but with metadata",
			version: "v1.11+vendor.1",
			expect:  "v1.11.0+vendor.1",
			error:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v ConformanceImageVersion
			err := v.Set(tc.version)
			if tc.error && err == nil {
				t.Fatal("expected error, got nil")
			} else if !tc.error && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if v.String() != tc.expect {
				t.Errorf("Expected %q but got %q", tc.expect, v.String())
			}
		})
	}
}

func TestGetConformanceImageVersion(t *testing.T) {
	workingServerVersion := &fakeServerVersionInterface{
		version: version.Info{
			Major:      "1",
			Minor:      "14",
			GitVersion: "v1.14.1",
		},
	}

	brokenServerVersion := &fakeServerVersionInterface{
		err: errors.New("can't connect"),
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "latest-dev-version+meta")
	}))
	defer ts.Close()

	tests := []struct {
		name          string
		version       ConformanceImageVersion
		serverVersion discovery.ServerVersionInterface
		expected      string
		expectedReg   string
		error         bool
	}{
		{
			name:          "auto retrieves server version",
			version:       "auto",
			serverVersion: workingServerVersion,
			expected:      "v1.14.1",
			expectedReg:   config.UpstreamKubeConformanceImageURL,
		},
		{
			name:          "auto returns error if upstream fails",
			version:       "auto",
			serverVersion: brokenServerVersion,
			error:         true,
		},
		{
			name:          "set version ignores server version",
			version:       "v1.11.2",
			serverVersion: workingServerVersion,
			expected:      "v1.11.2",
			expectedReg:   config.UpstreamKubeConformanceImageURL,
		},
		{
			name:          "set version ignores server version and can be anything",
			version:       "foo",
			serverVersion: workingServerVersion,
			expected:      "foo",
			expectedReg:   config.UpstreamKubeConformanceImageURL,
		},
		{
			name:          "set version doesn't call server so ignores errors",
			version:       "v1.11.2",
			serverVersion: brokenServerVersion,
			expected:      "v1.11.2",
			expectedReg:   config.UpstreamKubeConformanceImageURL,
		},
		{
			name:          "latest ignores server version and prefixes metadata with underscore",
			version:       "latest",
			serverVersion: workingServerVersion,
			expected:      "latest-dev-version_meta",
			expectedReg:   DevVersionImageURL,
		},
		{
			name:          "latest doesn't call server so ignores errors",
			version:       "latest",
			serverVersion: brokenServerVersion,
			expected:      "latest-dev-version_meta",
			expectedReg:   DevVersionImageURL,
		},
		{
			name:          "nil serverVersion",
			version:       "auto",
			serverVersion: nil,
			error:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reg, v, err := test.version.Get(test.serverVersion, ts.URL)
			if test.error && err == nil {
				t.Fatalf("expected error, got nil")
			} else if !test.error && err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if v != test.expected {
				t.Errorf("expected version %q, got %q", test.expected, v)
			}
			if reg != test.expectedReg {
				t.Errorf("expected registry %q, got %q", test.expectedReg, reg)
			}
		})
	}
}

func TestConformanceTagFromSemver(t *testing.T) {
	tcs := []struct {
		desc     string
		input    string
		expected string
		error    bool
	}{
		{
			desc:     "Alpha releases supported",
			input:    "v1.14.1-alpha.2.78+e0b33dbc2bde88",
			expected: "v1.14.1-alpha.2.78",
		}, {
			desc:     "Beta releases supported",
			input:    "v1.14.1-beta.2.78+e0b33dbc2bde88",
			expected: "v1.14.1-beta.2.78",
		}, {
			desc:     "Release candidates supported",
			input:    "v1.14.1-rc.2.78+e0b33dbc2bde88",
			expected: "v1.14.1-rc.2.78",
		}, {
			desc:     "Misc release ignored",
			input:    "v1.14.1-34.2.78+e0b33dbc2bde88",
			expected: "v1.14.1",
		}, {
			desc:     "providers version ignored",
			input:    "v1.14.1-gke.2.78+e0b33dbc2bde88",
			expected: "v1.14.1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out, err := conformanceTagFromSemver(tc.input)
			if tc.error && err == nil {
				t.Fatalf("expected error, got nil")
			} else if !tc.error && err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if out != tc.expected {
				t.Errorf("expected version %q, got %q", tc.expected, out)
			}
		})
	}
}

// fakeServerVersionInterface is used as a test implementation as
// discovery.ServerVersionInterface.
type fakeServerVersionInterface struct {
	err     error
	version version.Info
}

func (f *fakeServerVersionInterface) ServerVersion() (*version.Info, error) {
	return &f.version, f.err
}
