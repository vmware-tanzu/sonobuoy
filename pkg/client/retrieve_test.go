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
	"os"
	"path/filepath"
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

func TestGetFilename(t *testing.T) {
	testCases := []struct {
		desc   string
		input  string
		count  int
		expect string
	}{
		{
			desc:   "0 leads to no suffix",
			input:  "foo",
			expect: "foo",
		}, {
			desc:   "<0 leads to no suffix",
			input:  "foo",
			expect: "foo",
			count:  -1,
		}, {
			desc:   ">0 leads to suffix",
			input:  "foo",
			expect: "foo-01",
			count:  1,
		}, {
			desc:   "Suffix is before ext",
			input:  "foo.ext",
			expect: "foo-01.ext",
			count:  1,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			o := getFilename(tc.input, tc.count)
			if o != tc.expect {
				t.Errorf("Expected %v but got %v", tc.expect, o)
			}
		})
	}
}

func TestTarCmd(t *testing.T) {
	testCases := []struct {
		path     string
		expected []string
	}{
		{
			path:     "/results",
			expected: []string{"/sonobuoy", "splat", "/results"},
		},
		{
			path:     "",
			expected: []string{"/sonobuoy", "splat", ""},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			cmd := tarCmd(tc.path)
			if len(cmd) != len(tc.expected) {
				t.Fatalf("Expected length %d, got %d", len(tc.expected), len(cmd))
			}

			for i, v := range tc.expected {
				if cmd[i] != v {
					t.Errorf("Expected command[%d] to be %s, got %s", i, v, cmd[i])
				}
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sonobuoy-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file in the temp directory
	tmpFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testCases := []struct {
		name           string
		path           string
		expectedExists bool
		expectError    bool
	}{
		{
			name:           "Directory exists",
			path:           tmpDir,
			expectedExists: true,
			expectError:    false,
		},
		{
			name:           "File is not a directory",
			path:           tmpFile,
			expectedExists: false,
			expectError:    false,
		},
		{
			name:           "Path does not exist",
			path:           filepath.Join(tmpDir, "non-existent"),
			expectedExists: false,
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exists, err := dirExists(tc.path)

			if tc.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if exists != tc.expectedExists {
				t.Errorf("Expected exists to be %v, got %v", tc.expectedExists, exists)
			}
		})
	}
}
