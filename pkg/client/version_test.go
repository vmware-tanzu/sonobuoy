/*
Copyright the Sonobuoy contributors 2025

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
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

// TestSonobuoyClientVersion tests the Version method
func TestSonobuoyClientVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		versionErr error
		expected   string
		expectErr  bool
	}{
		{
			name:       "Successfully retrieves server version",
			version:    "v1.20.0",
			versionErr: nil,
			expected:   "v1.20.0",
			expectErr:  false,
		},
		{
			name:       "Returns error when server version fails",
			version:    "",
			versionErr: errors.New("server version error"),
			expected:   "",
			expectErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create the mock client
			mockClient := &mockKubeClient{
				version:    tc.version,
				versionErr: tc.versionErr,
			}

			// Create and set up our SonobuoyClient
			sonobuoyClient := &SonobuoyClient{}

			// Set the client directly
			sonobuoyClient.client = mockClient

			// Call the method under test
			result, err := sonobuoyClient.Version()

			// Verify results
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}

				if result != tc.expected {
					t.Errorf("Expected version %q, got %q", tc.expected, result)
				}
			}
		})
	}
}

// Mock Kubernetes client implementation
type mockKubeClient struct {
	kubernetes.Interface
	version    string
	versionErr error
}

// Mock the Discovery() method to return our mock discovery client
func (c *mockKubeClient) Discovery() discovery.DiscoveryInterface {
	return &mockDiscoveryClient{
		version: c.version,
		err:     c.versionErr,
	}
}

// mockDiscoveryClient implements just enough of discovery.DiscoveryInterface
type mockDiscoveryClient struct {
	discovery.DiscoveryInterface
	version string
	err     error
}

// Implement ServerVersion method for the test
func (m *mockDiscoveryClient) ServerVersion() (*version.Info, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &version.Info{
		GitVersion: m.version,
	}, nil
}
