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

package app

import (
	"testing"
)

func TestSubstituteRegistry(t *testing.T) {
	testCases := []struct {
		name           string
		image          string
		customRegistry string
		expected       string
	}{
		{
			name:           "Image with no registry has custom registry prepended",
			image:          "sonobuoy:v0.17.0",
			customRegistry: "my-custom-repo",
			expected:       "my-custom-repo/sonobuoy:v0.17.0",
		},
		{
			name:           "Registry is replaced with custom registry",
			image:          "sonobuoy/sonobuoy:v0.17.0",
			customRegistry: "my-custom-repo",
			expected:       "my-custom-repo/sonobuoy:v0.17.0",
		},
		{
			name:           "Custom registry ending with / does not result in // in result",
			image:          "sonobuoy/sonobuoy:v0.17.0",
			customRegistry: "my-custom-repo/",
			expected:       "my-custom-repo/sonobuoy:v0.17.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := substituteRegistry(tc.image, tc.customRegistry)
			if tc.expected != actual {
				t.Errorf("Unexpected registry substition, expected %q, got %q", tc.expected, actual)
			}
		})
	}

}
