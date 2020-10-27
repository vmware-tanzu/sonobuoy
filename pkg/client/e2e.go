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
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results/e2e"
)

// GetTests extracts the junit results from a sonobuoy archive and returns the requested tests.
func (*SonobuoyClient) GetTests(reader io.Reader, show string) ([]results.JUnitTestCase, error) {
	read := results.NewReaderWithVersion(reader, "irrelevant")
	junitResults := results.JUnitTestSuite{}
	e2eJUnitPath := path.Join(results.PluginsDir, e2e.ResultsSubdirectory, e2e.JUnitResultsFile)
	legacye2eJUnitPath := path.Join(results.PluginsDir, e2e.LegacyResultsSubdirectory, e2e.JUnitResultsFile)

	found := false
	err := read.WalkFiles(
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// TODO(chuckha) consider reusing this function for any generic e2e-esque plugin results.
			// TODO(chuckha) consider using path.Join()
			if path == e2eJUnitPath || path == legacye2eJUnitPath {
				found = true
				return results.ExtractFileIntoStruct(path, path, info, &junitResults)
			}
			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk results archive")
	}

	if !found {
		return nil, fmt.Errorf("failed to find results file %q in archive", e2eJUnitPath)
	}

	out := make([]results.JUnitTestCase, 0)
	if show == "passed" || show == "all" {
		out = append(out, results.JUnitFilter(results.JUnitPassed, junitResults)...)
	}
	if show == "failed" || show == "all" {
		out = append(out, results.JUnitFilter(results.JUnitFailed, junitResults)...)
	}
	if show == "skipped" || show == "all" {
		out = append(out, results.JUnitFilter(results.JUnitSkipped, junitResults)...)
	}
	sort.Sort(results.JUnitAlphabetizedTestCases(out))
	return out, nil
}

// Focus returns a value to be used in the E2E_FOCUS variable that is
// representative of the test cases in the struct.
func Focus(testCases []results.JUnitTestCase) string {
	testNames := make([]string, len(testCases))
	for i, tc := range testCases {
		testNames[i] = regexp.QuoteMeta(tc.Name)
	}
	return strings.Join(testNames, "|")
}

// PrintableTestCases nicely strings a []results.JUnitTestCase
type PrintableTestCases []results.JUnitTestCase

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
