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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"testing"
)

func TestDeleteInvalidConfig(t *testing.T) {
	testcases := []struct {
		desc             string
		config           *DeleteConfig
		expectedErrorMsg string
	}{
		{
			desc:             "Passing a nil config results in an error",
			config:           nil,
			expectedErrorMsg: "nil DeleteConfig provided",
		},
		{
			desc:             "Passing an invalid config with an empty namespace results in an error",
			config:           &DeleteConfig{},
			expectedErrorMsg: "config validation failed",
		},
	}

	c, err := NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			err = c.Delete(tc.config)
			expectedError := len(tc.expectedErrorMsg) > 0
			if !expectedError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if expectedError {
				if err == nil {
					t.Errorf("Expected provided config to be invalid but got no error")
				} else if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
					t.Errorf("Expected error to contain '%v', got '%v'", tc.expectedErrorMsg, err.Error())
				}
			}
		})
	}
}

func TestIsE2ENamespace(t *testing.T) {
	testcases := []struct {
		desc           string
		ns             v1.Namespace
		expectedResult bool
	}{
		{
			desc:           "Namespace with no labels is not classified as E2E namespace",
			ns:             v1.Namespace{},
			expectedResult: false,
		},
		{
			desc: "Namespace with non E2E labels is not classified as E2E namespace",
			ns: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"non-e2e-label": "some value",
					},
				},
			},
			expectedResult: false,
		},
		{
			desc: "Namespace with only e2e-framework label is not classified as E2E namespace",
			ns: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"non-e2e-label": "some value",
						"e2e-framework": "statefulset",
					},
				},
			},
			expectedResult: false,
		},
		{
			desc: "Namespace with only e2e-run label is not classified as E2E namespace",
			ns: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"non-e2e-label": "some value",
						"e2e-run":       "32c0b368-bacb-447e-8a66-03ea70a1be72",
					},
				},
			},
			expectedResult: false,
		},
		{
			desc: "Namespace with e2e-framework and e2e-run labels is classified as E2E namespace",
			ns: v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"non-e2e-label": "some value",
						"e2e-framework": "statefulset",
						"e2e-run":       "32c0b368-bacb-447e-8a66-03ea70a1be72",
					},
				},
			},
			expectedResult: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			result := isE2ENamespace(tc.ns)
			if result != tc.expectedResult {
				t.Errorf("expected e2e namespace classification for namespace with labels %v to be %v, got %v", tc.ns.Labels, tc.expectedResult, result)
			}
		})
	}
}
