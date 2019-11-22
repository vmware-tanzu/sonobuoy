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

func TestE2EImageSupportsProgress(t *testing.T) {
	tcs := []struct {
		desc     string
		input    string
		expected bool
	}{
		{
			desc:  "tag not semver wont support it",
			input: "someimage:sometag",
		}, {
			desc:  "v1.16 not supported",
			input: "someimage:v1.16.99",
		}, {
			desc:     "tag with v prefix is OK",
			input:    "someimage:v1.17.0",
			expected: true,
		}, {
			desc:     "tag without v prefix is OK",
			input:    "someimage:1.17.0",
			expected: true,
		}, {
			desc:     "tag with metadata is ok",
			input:    "someimage:v1.17.0+meta",
			expected: true,
		}, {
			desc:     "future versions ok",
			input:    "someimage:v1.19.1",
			expected: true,
		}, {
			desc:     "prerelease of v1.17 is ok",
			input:    "someimage:v1.17.0-beta",
			expected: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := e2eImageSupportsProgress(tc.input)
			if got != tc.expected {
				t.Errorf("Expected %v but got %v", tc.expected, got)
			}
		})
	}
}
