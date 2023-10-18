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

func TestVerifyKubernetesVersion(t *testing.T) {
	testCases := []struct {
		desc       string
		k8sVersion string
		expectErr  string
	}{
		{
			desc:       "Usage of e2e-docker-config-file flag with Kubernetes versions below 1.27 should throw error",
			k8sVersion: "1.26-alpha-3",
			expectErr:  "e2e-docker-config-file is only supported for Kubernetes 1.27 or later",
		},
		{
			desc:       "Usage of e2e-docker-config-file flag along with Kubernetes versions 1.27 or later should work as expected",
			k8sVersion: "1.27",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			err := verifyKubernetesVersion(testCase.k8sVersion)
			if err != nil {
				if len(testCase.expectErr) == 0 {
					t.Fatalf("Expected nil error but got %v", err)
				}
				if err.Error() != testCase.expectErr {
					t.Fatalf("Expected error %q but got %q", err, testCase.expectErr)
				}
			}
		})

	}
}
