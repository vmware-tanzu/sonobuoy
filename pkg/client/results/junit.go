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

package results

import (
	"encoding/xml"
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo/reporters"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Filter keeps only the tests that match the predicate function.
func Filter(predicate func(testCase reporters.JUnitTestCase) bool, testSuite reporters.JUnitTestSuite) []reporters.JUnitTestCase {
	out := make([]reporters.JUnitTestCase, 0)
	for _, tc := range testSuite.TestCases {
		if predicate(tc) {
			out = append(out, tc)
		}
	}
	return out
}

// AlphabetizedTestCases implements Sort over the list of testCases.
type AlphabetizedTestCases []reporters.JUnitTestCase

func (a AlphabetizedTestCases) Len() int           { return len(a) }
func (a AlphabetizedTestCases) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a AlphabetizedTestCases) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// predicate functions

// Skipped returns true if the test was skipped.
func Skipped(testCase reporters.JUnitTestCase) bool { return testCase.Skipped != nil }

// Passed returns true if the test passed.
func Passed(testCase reporters.JUnitTestCase) bool {
	return testCase.Skipped == nil && testCase.FailureMessage == nil
}

// Failed returns true if the test failed.
func Failed(testCase reporters.JUnitTestCase) bool {
	return testCase.Skipped == nil && testCase.FailureMessage != nil
}

func processJunitFile(pluginDir, currentFile string) (Item, error) {
	junitResults := reporters.JUnitTestSuite{}
	relPath, err := filepath.Rel(pluginDir, currentFile)
	if err != nil {
		logrus.Errorf("Error making path %q relative to %q: %v", pluginDir, currentFile, err)
		relPath = currentFile
	}

	// Passed unless a failure is encountered for aggregate status.
	resultObj := Item{
		Name:     filepath.Base(currentFile),
		Status:   StatusPassed,
		Metadata: map[string]string{"file": relPath},
	}

	infile, err := os.Open(currentFile)
	if err != nil {
		resultObj.Status = StatusUnknown
		resultObj.Metadata["error"] = err.Error()
		return resultObj, errors.Wrapf(err, "opening file %v", currentFile)
	}
	defer infile.Close()

	decoder := xml.NewDecoder(infile)
	if err := decoder.Decode(&junitResults); err != nil {
		resultObj.Status = StatusUnknown
		resultObj.Metadata["error"] = err.Error()
		return resultObj, errors.Wrap(err, "error decoding json into object")
	}

	for _, t := range junitResults.TestCases {
		status := StatusUnknown
		switch {
		case Passed(t):
			status = StatusPassed
		case Failed(t):
			resultObj.Status = StatusFailed
			status = StatusFailed
		case Skipped(t):
			status = StatusSkipped
		}

		resultObj.Items = append(resultObj.Items, Item{Name: t.Name, Status: status})
	}

	return resultObj, nil
}
