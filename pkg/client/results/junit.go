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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// JUnitStdoutKey is the key in the Items.Details map for the system-out output.
	JUnitStdoutKey = "system-out"

	// JUnitStderrKey is the key in the Items.Details map for the system-out output.
	JUnitStderrKey = "system-err"

	// JUnitFailureKey is the key in the Items.Details map for the failure output.
	JUnitFailureKey = "failure"

	// JUnitErrorKey is the key in the Items.Details map for the error output.
	JUnitErrorKey = "error"
)

// JUnitResult is a wrapper around the suite[s] which enable results to
// be either a single suite or a collection of suites. For instance,
// e2e tests (which use the onsi/ginkgo reporter) report a single, top-level
// testsuite whereas other tools report a top-level testsuites object
// which may have 1+ testsuite children. Only one of the fields should be
// set, not both.
type JUnitResult struct {
	Suites JUnitTestSuites
}

// UnmarshalXML will unmarshal the document into either the 'Suites'
// field of the JUnitResult. If a single testsuite is found (instead of a testsuites object)
// it is unmarshalled and then added into the set of JUnitResult.Suites.Suites
func (j *JUnitResult) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var returnErr error
	if start.Name.Local == "testsuites" {
		returnErr = d.DecodeElement(&j.Suites, &start)
	} else {
		var s JUnitTestSuite
		returnErr = d.DecodeElement(&s, &start)
		j.Suites = JUnitTestSuites{Suites: []JUnitTestSuite{s}}
	}
	for i := range j.Suites.Suites {
		if j.Suites.Suites[i].Name == "" {
			j.Suites.Suites[i].Name = fmt.Sprintf("testsuite-%03d", i+1)
		}
	}
	return returnErr
}

// JUnitTestSuites is a collection of JUnit test suites.
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite is a single JUnit test suite which may contain many
// testcases.
type JUnitTestSuite struct {
	XMLName    xml.Name        `xml:"testsuite"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Time       float64         `xml:"time,attr"`
	Name       string          `xml:"name,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase is a single test case with its result.
type JUnitTestCase struct {
	XMLName      xml.Name             `xml:"testcase"`
	Classname    string               `xml:"classname,attr"`
	Name         string               `xml:"name,attr"`
	Time         string               `xml:"time,attr"`
	SkipMessage  *JUnitSkipMessage    `xml:"skipped,omitempty"`
	Failure      *JUnitFailureMessage `xml:"failure,omitempty"`
	ErrorMessage *JUnitErrorMessage   `xml:"error,omitempty"`
	SystemOut    string               `xml:"system-out,omitempty"`
	SystemErr    string               `xml:"system-err,omitempty"`
}

// JUnitSkipMessage contains the reason why a testcase was skipped.
type JUnitSkipMessage struct {
	Message string `xml:"message,attr"`
}

// JUnitFailureMessage contains data related to a failed test.
type JUnitFailureMessage struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitErrorMessage contains data related to a failed test.
type JUnitErrorMessage struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitProperty represents a key/value pair used to define properties.
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitFilter keeps only the tests that match the predicate function.
func JUnitFilter(predicate func(testCase JUnitTestCase) bool, testSuite JUnitTestSuite) []JUnitTestCase {
	out := make([]JUnitTestCase, 0)
	for _, tc := range testSuite.TestCases {
		if predicate(tc) {
			out = append(out, tc)
		}
	}
	return out
}

// JUnitAlphabetizedTestCases implements Sort over the list of testCases.
type JUnitAlphabetizedTestCases []JUnitTestCase

func (a JUnitAlphabetizedTestCases) Len() int           { return len(a) }
func (a JUnitAlphabetizedTestCases) Less(i, j int) bool { return a[i].Name < a[j].Name }
func (a JUnitAlphabetizedTestCases) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

// predicate functions

// JUnitSkipped returns true if the test was skipped.
func JUnitSkipped(testCase JUnitTestCase) bool { return testCase.SkipMessage != nil }

// JUnitPassed returns true if the test passed.
func JUnitPassed(testCase JUnitTestCase) bool {
	return testCase.SkipMessage == nil && testCase.Failure == nil && testCase.ErrorMessage == nil
}

// JUnitFailed returns true if the test failed.
func JUnitFailed(testCase JUnitTestCase) bool {
	return testCase.SkipMessage == nil && testCase.Failure != nil
}

// JUnitErrored returns true if the test errored.
func JUnitErrored(testCase JUnitTestCase) bool {
	return testCase.SkipMessage == nil && testCase.Failure == nil && testCase.ErrorMessage != nil
}

func JunitProcessFile(pluginDir, currentFile string) (Item, error) {
	relPath, err := filepath.Rel(pluginDir, currentFile)
	if err != nil {
		logrus.Errorf("Error making path %q relative to %q: %v", pluginDir, currentFile, err)
		relPath = currentFile
	}

	resultObj := Item{
		Name:   filepath.Base(currentFile),
		Status: StatusUnknown,
		Metadata: map[string]string{
			MetadataFileKey: relPath,
			MetadataTypeKey: MetadataTypeFile,
		},
	}

	infile, err := os.Open(currentFile)
	if err != nil {
		resultObj.Metadata["error"] = err.Error()
		resultObj.Status = StatusUnknown

		return resultObj, errors.Wrapf(err, "opening file %v", currentFile)
	}
	defer infile.Close()

	resultObj, err = junitProcessReader(
		infile,
		resultObj.Name,
		resultObj.Metadata,
	)
	if err != nil {
		return resultObj, errors.Wrap(err, "error processing junit")
	}

	return resultObj, nil
}

func junitProcessReader(r io.Reader, name string, metadata map[string]string) (Item, error) {
	rootItem := Item{
		Name:     name,
		Status:   StatusPassed,
		Metadata: metadata,
	}

	if r == nil {
		rootItem.Status = StatusUnknown
		if rootItem.Metadata == nil {
			rootItem.Metadata = map[string]string{}
		}
		rootItem.Metadata["error"] = "no data source for junit"
		return rootItem, errors.New("no data source for junit")
	}

	decoder := xml.NewDecoder(r)
	junitResults := JUnitResult{}
	if err := decoder.Decode(&junitResults); err != nil {
		rootItem.Status = StatusUnknown
		if rootItem.Metadata == nil {
			rootItem.Metadata = map[string]string{}
		}
		rootItem.Metadata["error"] = err.Error()
		return rootItem, errors.Wrap(err, "decoding junit")
	}

	for _, ts := range junitResults.Suites.Suites {
		suiteItem := Item{
			Name:   ts.Name,
			Status: StatusPassed,
		}
		for _, t := range ts.TestCases {
			status := StatusUnknown
			switch {
			case JUnitPassed(t):
				status = StatusPassed
			case JUnitFailed(t):
				rootItem.Status = StatusFailed
				suiteItem.Status = StatusFailed
				status = StatusFailed
			case JUnitErrored(t):
				rootItem.Status = StatusFailed
				suiteItem.Status = StatusFailed
				status = StatusFailed
			case JUnitSkipped(t):
				status = StatusSkipped
			}
			testItem := Item{Name: t.Name, Status: status, Details: map[string]interface{}{}, Metadata: map[string]string{}}

			// Different JUnit implementations build the objects in slightly different ways.
			// Some will only use contents, some only the message attribute. Here we just concat
			// the values, separated with a space.
			hasFailureContents := t.Failure != nil && (t.Failure.Message != "" || t.Failure.Contents != "")
			if hasFailureContents {
				testItem.Details[JUnitFailureKey] = strings.TrimSpace(t.Failure.Message + " " + t.Failure.Contents)
			}
			hasErrorContents := t.ErrorMessage != nil && (t.ErrorMessage.Message != "" || t.ErrorMessage.Contents != "")
			if hasErrorContents {
				testItem.Details[JUnitErrorKey] = strings.TrimSpace(t.ErrorMessage.Message + " " + t.ErrorMessage.Contents)
			}

			if t.SystemOut != "" {
				testItem.Details[JUnitStdoutKey] = t.SystemOut
			}
			if t.SystemErr != "" {
				testItem.Details[JUnitStderrKey] = t.SystemErr
			}

			suiteItem.Items = append(suiteItem.Items, testItem)
		}
		rootItem.Items = append(rootItem.Items, suiteItem)
	}

	return rootItem, nil
}
