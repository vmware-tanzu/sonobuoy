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
)

func ExampleNewCmdResults() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{filepath.Join("testdata", "testResultsOutput.tar.gz")})
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

func ExampleNewCmdResults_detailed() {
	cmd := NewCmdResults()
	cmd.SetArgs([]string{
		filepath.Join("testdata", "testResultsOutput.tar.gz"),
		"--mode", "detailed",
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
