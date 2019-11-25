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

package image

import (
	"strings"
	"testing"
)

func TestGetDefaultImageRegistryVersionValidation(t *testing.T) {
	tests := []struct {
		name    string
		version string
		error   bool
		expect  string
	}{
		{
			name:    "Non valid version results in error",
			version: "not-a-valid-version",
			error:   true,
			expect:  "\"not-a-valid-version\" is invalid",
		},
		{
			name:    "v1.13 is valid",
			version: "v1.13.0",
			error:   false,
		},
		{
			name:    "v1.14 is valid",
			version: "v1.14.0",
			error:   false,
		},
		{
			name:    "v1.15 is valid",
			version: "v1.15.0",
			error:   false,
		},
		{
			name:    "v1.16 is valid",
			version: "v1.16.0",
			error:   false,
		},
		{
			name:    "v1.12 is not valid",
			version: "v1.12.0",
			error:   true,
			expect:  "No matching configuration for k8s version: 1.12",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := GetDefaultImageRegistries(tc.version)
			if tc.error && err == nil {
				t.Fatal("expected error, got nil")
			} else if !tc.error && err != nil {
				t.Fatalf("expected no error, got %v", err)
			} else if tc.error && !strings.Contains(err.Error(), tc.expect) {
				t.Fatalf("expected error to contain %q, got %v", tc.expect, err.Error())
			}

		})
	}
}
