/*
Copyright 2018 the Sonobuoy project contributors

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
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	testhook "github.com/sirupsen/logrus/hooks/test"
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
			name:    "version with addendum",
			version: "v1.13.0-beta.2.78+e0b33dbc2bde88",
			expect:  "v1.13.0-beta.2.78+e0b33dbc2bde88",
			error:   false,
		},
		{
			name:    "version with plus",
			version: "v1.11+",
			error:   true,
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
	testHook := &testhook.Hook{}
	logrus.AddHook(testHook)
	logrus.SetOutput(ioutil.Discard)

	workingServerVersion := &fakeServerVersionInterface{
		version: version.Info{
			Major:      "1",
			Minor:      "14",
			GitVersion: "v1.14.1",
		},
	}

	betaServerVersion := &fakeServerVersionInterface{
		version: version.Info{
			Major:      "1",
			Minor:      "14",
			GitVersion: "v1.14.1-beta.2.78+e0b33dbc2bde88",
		},
	}

	gkeServerVersion := &fakeServerVersionInterface{
		version: version.Info{
			Major:      "1",
			Minor:      "11+",
			GitVersion: "v1.11.5-gke.3",
		},
	}

	brokenServerVersion := &fakeServerVersionInterface{
		err: errors.New("can't connect"),
	}

	tests := []struct {
		name          string
		version       ConformanceImageVersion
		serverVersion discovery.ServerVersionInterface
		expected      string
		error         bool
		warning       bool
	}{
		{
			name:          "auto retrieves server version",
			version:       "auto",
			serverVersion: workingServerVersion,
			expected:      "v1.14.1",
		},
		{
			name:          "auto returns error if upstream fails",
			version:       "auto",
			serverVersion: brokenServerVersion,
			error:         true,
		},
		{
			name:          "beta server version throws warning",
			version:       "auto",
			serverVersion: betaServerVersion,
			warning:       true,
			expected:      "v1.14.1",
		},
		{
			name:          "gke server strips plus sign",
			version:       "auto",
			serverVersion: gkeServerVersion,
			expected:      "v1.11",
		},
		{
			name:          "set version ignores server version",
			version:       "v1.11.2",
			serverVersion: workingServerVersion,
			expected:      "v1.11.2",
		},
		{
			name:          "set version ignores server version and can be anything",
			version:       "foo",
			serverVersion: workingServerVersion,
			expected:      "foo",
		},
		{
			name:          "set version doesn't call server so ignores errors",
			version:       "v1.11.2",
			serverVersion: brokenServerVersion,
			expected:      "v1.11.2",
		},
		{
			name:          "latest ignores server version",
			version:       "latest",
			serverVersion: workingServerVersion,
			expected:      "latest",
		},
		{
			name:          "latest doesn't call server so ignores errors",
			version:       "latest",
			serverVersion: brokenServerVersion,
			expected:      "latest",
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
			testHook.Reset()
			v, err := test.version.Get(test.serverVersion)
			if test.error && err == nil {
				t.Fatalf("expected error, got nil")
			} else if !test.error && err != nil {
				t.Fatalf("unexpecter error %v", err)
			}

			if test.warning {
				last := testHook.LastEntry()
				if last == nil {
					t.Errorf("expected warning entry, got nothing")
				} else if last.Level != logrus.WarnLevel {
					t.Errorf("expected level %v, got %v", logrus.WarnLevel, last.Level)
				}
			}

			if v != test.expected {
				t.Errorf("expected version %q, got %q", test.expected, v)
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
