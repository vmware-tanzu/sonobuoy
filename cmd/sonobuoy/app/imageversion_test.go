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

package app

import (
	"testing"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
)

func TestSetConformanceImageVersion(t *testing.T) {

	tests := []struct {
		name    string
		version string
		error   bool
	}{
		{
			name:    "version detect",
			version: "auto",
			error:   false,
		},
		{
			name:    "use latest",
			version: "latest",
			error:   false,
		},
		{
			name:    "random string",
			version: "test",
			error:   true,
		},
		{
			name:    "stable version",
			version: "v1.11.0",
			error:   false,
		},
		{
			name:    "version without v",
			version: "1.11.0",
			error:   true,
		},
		{
			name:    "empty string",
			version: "",
			error:   true,
		},
		{
			name:    "version with addendum",
			version: "v1.11.0-beta.2.78+e0b33dbc2bde88",
			error:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var v ConformanceImageVersion
			err := v.Set(test.version)
			if test.error && err == nil {
				t.Error("expected error, got nil")
			} else if !test.error && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestGetConformanceImageVersion(t *testing.T) {
	workingServerVersion := &fakeServerVersionInterface{
		version: version.Info{
			GitVersion: "v1.11.0",
		},
	}

	betaServerVersion := &fakeServerVersionInterface{
		version: version.Info{
			GitVersion: "v1.11.0-beta.2.78+e0b33dbc2bde88",
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
	}{
		{
			name:          "auto retrieves server version",
			version:       "auto",
			serverVersion: workingServerVersion,
			expected:      "v1.11.0",
		},
		{
			name:          "auto returns error if upstream fails",
			version:       "auto",
			serverVersion: brokenServerVersion,
			error:         true,
		},
		{
			name:          "beta server version throws error",
			version:       "auto",
			serverVersion: betaServerVersion,
			error:         true,
		},
		{
			name:          "set version ignores server version",
			version:       "v1.10.2",
			serverVersion: workingServerVersion,
			expected:      "v1.10.2",
		},
		{
			name:          "set version doesn't call server so ignores errors",
			version:       "v1.10.2",
			serverVersion: brokenServerVersion,
			expected:      "v1.10.2",
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
			v, err := test.version.Get(test.serverVersion)
			if test.error && err == nil {
				t.Fatalf("expected error, got nil")
			} else if !test.error && err != nil {
				t.Fatalf("unexpecter error %v", err)
			}

			if v != test.expected {
				t.Errorf("expected version %q, got %q", test.expected, v)
			}
		})
	}
}

type fakeServerVersionInterface struct {
	err     error
	version version.Info
}

func (f *fakeServerVersionInterface) ServerVersion() (*version.Info, error) {
	return &f.version, f.err
}
