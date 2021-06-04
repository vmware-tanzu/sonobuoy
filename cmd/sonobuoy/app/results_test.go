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
	"path/filepath"
	"testing"
)

func TestGetFileFromMeta(t *testing.T) {
	tcs := []struct {
		desc     string
		input    map[string]string
		expected string
	}{
		{
			desc:     "Nil map",
			input:    nil,
			expected: "",
		}, {
			desc:     "Empty",
			input:    map[string]string{},
			expected: "",
		}, {
			desc:     "File with slash",
			input:    map[string]string{"file": "a/b/c"},
			expected: "a/b/c",
		}, {
			desc:     "File with windows seperators",
			input:    map[string]string{"file": `a\b\c`},
			expected: "a/b/c",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out := getFileFromMeta(tc.input)
			if out != tc.expected {
				t.Errorf("Expected %v but got %v", tc.expected, out)
			}
		})
	}
}

func ExampleNewCmdResults() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin=e2e",
	})
	cmd.Execute()
	// Output:
	// Plugin: e2e
	// Status: failed
	// Total: 3
	// Passed: 1
	// Failed: 1
	// Skipped: 1
	//
	// Failed tests:
	// [sig-storage] CSI Volumes CSI Topology test using GCE PD driver [Serial] should fail to schedule a pod with a zone missing from AllowedTopologies; PD is provisioned with immediate volume binding
}

func ExampleNewCmdResults_custom() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin=custom-status",
	})
	cmd.Execute()
	// Output:
	// Plugin: custom-status
	// Status: custom-overall-status
	// Total: 7
	// Passed: 1
	// Failed: 1
	// Skipped: 1
	// complete: 2
	// custom: 2
	//
	// Failed tests:
	// [sig-storage] CSI Volumes CSI Topology test using GCE PD driver [Serial] should fail to schedule a pod with a zone missing from AllowedTopologies; PD is provisioned with immediate volume binding
}

func ExampleNewCmdResults_detailed() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--mode", "detailed", "--plugin=e2e",
	})
	cmd.Execute()
	// Output:
	// {"name":"[sig-storage] CSI Volumes CSI Topology test using GCE PD driver [Serial] should fail to schedule a pod with a zone missing from AllowedTopologies; PD is provisioned with immediate volume binding","status":"failed","meta":{"path":"e2e|junit_01.xml"}}
	// {"name":"[sig-storage] Subpath Atomic writer volumes should support subpaths with projected pod [LinuxOnly] [Conformance]","status":"passed","meta":{"path":"e2e|junit_01.xml"}}
	// {"name":"[sig-storage] In-tree Volumes [Driver: hostPath] [Testpattern: Inline-volume (default fs)] subPath should fail if non-existent subpath is outside the volume [Slow]","status":"skipped","meta":{"path":"e2e|junit_01.xml"}}
}

func ExampleNewCmdResults_plugin() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin", "tarresultsds",
	})
	cmd.Execute()
	// Output:
	// Plugin: tarresultsds
	// Status: passed
	// Total: 2
	// Passed: 2
	// Failed: 0
	// Skipped: 0
}

func ExampleNewCmdResults_pluginDetailed() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin", "tarresultsds",
		"--mode", "detailed",
	})
	cmd.Execute()
	// Output:
	// tarresultsds|kind-control-plane|out0 hello world
	// tarresultsds|kind-control-plane|out1 hello world pt2
}

func ExampleNewCmdResults_pluginDetailedNode() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin", "tarresultsds",
		"--mode", "detailed",
		"--node", "out1",
	})
	cmd.Execute()
	// Output:
	// out1 hello world pt2
}

func ExampleNewCmdResults_skipPrefix() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin", "tarresultsds",
		"--skip-prefix",
		"--mode=detailed",
		"--node", "out1",
	})
	cmd.Execute()
	// Output:
	// hello world pt2
}

func ExampleNewCmdResults_pluginDetailedArbitraryDetails() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--plugin", "arbitrary-details",
		"--mode", "detailed",
	})
	cmd.Execute()
	// Output:
	// {"name":"Item with arbitrary details","status":"complete","meta":{"path":"arbitrary-details|output-file"},"details":{"nested-details":{"key1":"value1","key2":"value2"},"string-array":["string 1","string 2","string 3"]}}
	// {"name":"Another item with arbitrary details","status":"complete","meta":{"path":"arbitrary-details|output-file"},"details":{"integer-array":[1,2,3],"nested-details":{"key1":"value1","key2":"value2","key3":{"nested-key1":"nested-value1","nested-key2":"nested-value2","nested-key3":{"another-nested-key":"another-nested-value"}}}}}
}
