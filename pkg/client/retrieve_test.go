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
	"strings"
	"testing"
)

func TestRetrieveInvalidConfig(t *testing.T) {
	testcases := []struct {
		config           *RetrieveConfig
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			config:           nil,
			expectedError:    true,
			expectedErrorMsg: "nil RetrieveConfig provided",
		},
		{
			config:           &RetrieveConfig{},
			expectedError:    true,
			expectedErrorMsg: "config validation failed",
		},
	}

	c, err := NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		_, _, err = c.RetrieveResults(tc.config)
		if !tc.expectedError && err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if tc.expectedError {
			if err == nil {
				t.Errorf("Expected provided config to be invalid but got no error")
			} else if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
				t.Errorf("Expected error to contain '%v', got '%v'", tc.expectedErrorMsg, err.Error())
			}
		}
	}
}
