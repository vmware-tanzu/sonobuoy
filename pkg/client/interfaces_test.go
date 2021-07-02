/*
Copyright the Sonobuoy contributors 2019

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

package client

import (
	"testing"
)

// ConfigValidator allows the command configurations to be validated.
type validator interface {
	Validate() error
}

func TestConfigValidation(t *testing.T) {
	testcases := []struct {
		desc          string
		config        validator
		valid         bool
		expectedError string
	}{
		{
			desc:          "log config with no namespace is not valid",
			config:        &LogConfig{},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:          "log config with empty namespace is not valid",
			config:        &LogConfig{Namespace: "", Follow: true},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:   "log config with namespace is valid",
			config: &LogConfig{Namespace: "valid-namespace"},
			valid:  true,
		},
		{
			desc:          "delete config with no namespace is not valid",
			config:        &DeleteConfig{},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:          "delete config with empty namespace is not valid",
			config:        &DeleteConfig{Namespace: "", EnableRBAC: true},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:   "delete config with namespace is valid",
			config: &DeleteConfig{Namespace: "valid-namespace", DeleteAll: true},
			valid:  true,
		},
		{
			desc:          "retrieve config with no namespace is not valid",
			config:        &RetrieveConfig{},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:          "retrieve config with empty namespace is not valid",
			config:        &RetrieveConfig{Namespace: ""},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:   "retrieve config with namespace is valid",
			config: &RetrieveConfig{Namespace: "valid-namespace"},
			valid:  true,
		},
		{
			desc:          "status config with no namespace is not valid",
			config:        &StatusConfig{},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:          "status config with empty namespace is not valid",
			config:        &StatusConfig{Namespace: ""},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:   "status config with namespace is valid",
			config: &StatusConfig{Namespace: "valid-namespace"},
			valid:  true,
		},
		{
			desc:          "preflight config with no namespace is not valid",
			config:        &PreflightConfig{},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:          "preflight config with no namespace is not valid",
			config:        &PreflightConfig{Namespace: ""},
			valid:         false,
			expectedError: "namespace cannot be empty",
		},
		{
			desc:   "preflight config with namespace is valid",
			config: &PreflightConfig{Namespace: "valid-namespace"},
			valid:  true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.valid && err != nil {
				t.Errorf("Expected config to be valid, got error: %v", err)
			}

			if !tc.valid {
				if err == nil {
					t.Errorf("Expected config to not be valid but got no error")
				} else if err.Error() != tc.expectedError {
					t.Errorf("Expected error to be '%v', got '%v'", tc.expectedError, err.Error())
				}
			}
		})
	}
}
