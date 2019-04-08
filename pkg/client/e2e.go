/*
Copyright 2018 Heptio Inc.

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
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/heptio/sonobuoy/pkg/client/results"
	"github.com/heptio/sonobuoy/pkg/client/results/e2e"
	"github.com/onsi/ginkgo/reporters"
	"github.com/pkg/errors"
)

// GetTests extracts the junit results from a sonobuoy archive and returns the requested tests.
func (*SonobuoyClient) GetTests(reader io.Reader, show string) ([]reporters.JUnitTestCase, error) {
	read := results.NewReaderWithVersion(reader, "irrelevant")
	junitResults := reporters.JUnitTestSuite{}
	e2eJunitPath := path.Join(results.PluginsDir, e2e.ResultsSubdirectory, e2e.JUnitResultsFile)

	found := false
	err := read.WalkFiles(
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// TODO(chuckha) consider reusing this function for any generic e2e-esque plugin results.
			// TODO(chuckha) consider using path.Join()
			if path == e2eJunitPath {
				found = true
				return results.ExtractFileIntoStruct(e2eJunitPath, path, info, &junitResults)
			}
			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk results archive")
	}

	if !found {
		return nil, fmt.Errorf("failed to find results file %q in archive", e2eJunitPath)
	}

	out := make([]reporters.JUnitTestCase, 0)
	if show == "passed" || show == "all" {
		out = append(out, results.Filter(results.Passed, junitResults)...)
	}
	if show == "failed" || show == "all" {
		out = append(out, results.Filter(results.Failed, junitResults)...)
	}
	if show == "skipped" || show == "all" {
		out = append(out, results.Filter(results.Skipped, junitResults)...)
	}
	sort.Sort(results.AlphabetizedTestCases(out))
	return out, nil
}

// Focus returns a value to be used in the E2E_FOCUS variable that is
// representative of the test cases in the struct.
func Focus(testCases []reporters.JUnitTestCase) string {
	// YAML doesn't like escaped characters and regex needs escaped characters. Therefore a double escape is necessary.
	r := strings.NewReplacer("[", `\\[`, "]", `\\]`)
	testNames := make([]string, len(testCases))
	for i, tc := range testCases {
		testNames[i] = r.Replace(tc.Name)
	}
	return strings.Join(testNames, "|")
}

// PrintableTestCases nicely strings a []reporters.JunitTestCase
type PrintableTestCases []reporters.JUnitTestCase

func (p PrintableTestCases) String() string {
	if len(p) == 0 {
		return ""
	}

	out := make([]string, len(p))
	for i, tc := range p {
		out[i] = tc.Name
	}
	return strings.Join(out, "\n")
}
