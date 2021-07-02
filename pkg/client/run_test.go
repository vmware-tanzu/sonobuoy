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

func TestRunInvalidConfig(t *testing.T) {
	testcases := []struct {
		desc             string
		config           *RunConfig
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			desc:             "Passing a nil config results in an error",
			config:           nil,
			expectedErrorMsg: "nil RunConfig provided",
		},
		{
			desc: "Passing a file takes priority over config flags",
			config: &RunConfig{
				GenFile: "foo.yaml",
			},
			expectedErrorMsg: "no such file or directory",
		},
		// NOTE: Running non-failing logic here is not supported at this time due to the fact
		// that it tries to actually start executing logic with the dynamic client which
		// is nil.
	}

	c, err := NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			expectedError := len(tc.expectedErrorMsg) > 0
			err = c.Run(tc.config)
			if !expectedError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if expectedError {
				if err == nil {
					t.Errorf("Expected provided config to be invalid but got no error")
				} else if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tc.expectedErrorMsg, err.Error())
				}
			}
		})
	}
}
