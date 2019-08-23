package results

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

type errReader struct {
	io.Reader
}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("testErr")
}

func TestJUnitProcessReader(t *testing.T) {
	xmlFromFile := func(path string) io.Reader {
		f, err := os.Open(path)
		defer f.Close()
		var bb bytes.Buffer
		_, err = io.Copy(&bb, f)
		if err != nil {
			t.Fatalf("Failed to read test file %v: %v", path, err)
		}
		return &bb
	}
	itemFromFile := func(t *testing.T, path string) Item {
		i := Item{}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read test file %v: %v", path, err)
		}
		err = json.Unmarshal(b, &i)
		if err != nil {
			t.Fatalf("Failed to unmarshal test data: %v", err)
		}
		return i
	}
	tcs := []struct {
		desc      string
		input     io.Reader
		name      string
		meta      map[string]string
		expect    Item
		expectErr string

		// Just a handy way to allow us to update these via a flag if needed without
		// repeating the path in multiple places. Will load/set `expect` field in test.
		expectItemFromFile string
	}{
		{
			desc: "Nil reader doesnt panic",
			expect: Item{
				Status:   StatusUnknown,
				Metadata: map[string]string{"error": "no data source for junit"},
			},
			expectErr: "no data source for junit",
		}, {
			desc:               "Failing reader doesnt panic and returns errors",
			input:              &errReader{bytes.NewBufferString("text-wont-matter")},
			expectItemFromFile: filepath.Join("testdata", "item_badjunit.json"),
			expectErr:          "decoding junit: testErr",
		}, {
			desc:               "Can unmarshal suites",
			input:              xmlFromFile(filepath.Join("testdata", "junit_good_junit.xml")),
			expectItemFromFile: filepath.Join("testdata", "item_good_junit.json"),
		}, {
			desc:               "Can unmarshal multiple suites, each in their own branch",
			input:              xmlFromFile(filepath.Join("testdata", "junit_multiple_suites.xml")),
			expectItemFromFile: filepath.Join("testdata", "item_multiple_suites.json"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			item, err := junitProcessReader(tc.input, tc.name, tc.meta)

			// Allow for fast updating of golden files.
			if *update && len(tc.expectItemFromFile) > 0 {
				b, err := json.MarshalIndent(item, "", "")
				if err != nil {
					t.Fatalf("Failed to marshal expected Item for debug: %v", err)
				}
				t.Logf("Updating goldenfile %v", tc.expectItemFromFile)
				ioutil.WriteFile(tc.expectItemFromFile, b, 0666)
				return
			}

			// Load from file (if applicable) then compare.
			if len(tc.expectItemFromFile) > 0 {
				tc.expect = itemFromFile(t, tc.expectItemFromFile)
			}
			if diff := pretty.Compare(item, tc.expect); diff != "" {
				t.Errorf("\n\n%s\n", diff)
			}
			if err != nil && len(tc.expectErr) == 0 {
				t.Errorf("Expected nil error but got %q", err)
			}
			if err == nil && len(tc.expectErr) > 0 {
				t.Errorf("Expected error %q but got nil", tc.expectErr)
			}
			if err != nil && fmt.Sprint(err) != tc.expectErr {
				t.Errorf("Expected error to be %q but got %q", tc.expectErr, err)
			}
		})
	}
}

func TestJUnitResult_UnmarshalXML(t *testing.T) {
	tcs := []struct {
		desc   string
		input  string
		expect junitResult
	}{
		{
			desc:  "top-level testsuites",
			input: `<testsuites><testsuite name="testsuite1"></testsuite><testsuite name="testsuite2"></testsuite></testsuites>`,
			expect: junitResult{
				suites: JUnitTestSuites{
					Suites: []JUnitTestSuite{
						{Name: "testsuite1"},
						{Name: "testsuite2"},
					},
				},
			},
		}, {
			desc:  "top-level testsuite",
			input: `<testsuite name="testsuite1"></testsuite>`,
			expect: junitResult{
				suites: JUnitTestSuites{
					Suites: []JUnitTestSuite{
						{Name: "testsuite1"},
					},
				},
			},
		}, {
			desc:  "Empty testsuite name gets filled with unique values",
			input: `<testsuite></testsuite>`,
			expect: junitResult{
				suites: JUnitTestSuites{
					Suites: []JUnitTestSuite{
						{Name: "testsuite-001"},
					},
				},
			},
		}, {
			desc:  "Empty testsuite names get filled with unique values",
			input: `<testsuites><testsuite></testsuite><testsuite></testsuite></testsuites>`,
			expect: junitResult{
				suites: JUnitTestSuites{
					Suites: []JUnitTestSuite{
						{Name: "testsuite-001"},
						{Name: "testsuite-002"},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out := junitResult{}
			err := xml.Unmarshal([]byte(tc.input), &out)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// XMLName fields complicate the comparison since it is set during
			// deserialization. Unset all those values before comparing.
			out.suites.XMLName = xml.Name{}
			for i := range out.suites.Suites {
				out.suites.Suites[i].XMLName = xml.Name{}
			}

			if diff := pretty.Compare(out, tc.expect); diff != "" {
				t.Errorf("\n\n%s\n", diff)
			}
		})
	}
}
