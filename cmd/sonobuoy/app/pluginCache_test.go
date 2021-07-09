/*
Copyright the Sonobuoy contributors 2021

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
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestFilenameFromArg(t *testing.T) {
	tcs := []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:     "Adds ext",
			input:    "in",
			expected: "in.yaml",
		}, {
			desc:     "Accepts ext already present",
			input:    "in.yaml",
			expected: "in.yaml",
		}, {
			desc:     "Only looks at last extension",
			input:    "in.ext1.ext2.yaml",
			expected: "in.ext1.ext2.yaml",
		}, {
			desc:     "Only looks at last extension",
			input:    "in.ext1.ext2",
			expected: "in.ext1.ext2.yaml",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			o := filenameFromArg(tc.input, ".yaml")
			if o != tc.expected {
				t.Errorf("Expected %v but got %v", tc.expected, o)
			}
		})
	}
}

func TestGetPluginCacheLocation(t *testing.T) {
	defaultDir, err := expandPath(defaultSonobuoyDir)
	if err != nil {
		t.Fatal(err)
	}

	tcs := []struct {
		desc     string
		input    string
		expected string
		cleanup  bool
	}{
		{
			desc:     "Defaults to home directory and resolves",
			input:    "",
			expected: defaultDir,
		}, {
			desc:     "Override from env",
			input:    "./testdata/foo",
			expected: "./testdata/foo",
			cleanup:  true,
		}, {
			desc:     "Bad dir means empty result",
			input:    "~~~~/testdata",
			expected: "",
		},
	}

	for _, tc := range tcs {
		origval := os.Getenv(SonobuoyDirEnvKey)
		defer os.Setenv(SonobuoyDirEnvKey, origval)
		t.Run(tc.desc, func(t *testing.T) {
			if tc.cleanup {
				defer os.RemoveAll(tc.input)
			}
			cmd := &cobra.Command{}
			os.Setenv(SonobuoyDirEnvKey, tc.input)
			o := getPluginCacheLocation(cmd)
			if o != tc.expected {
				t.Errorf("Expected %v but got %v", tc.expected, o)
			}
		})
	}
}
