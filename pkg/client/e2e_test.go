package client

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"

	"k8s.io/client-go/rest"
)

func TestGetTests(t *testing.T) {
	testCases := []struct {
		desc      string
		path      string
		show      string
		expect    int
		expectErr string
	}{
		{
			desc:   "Shows failed tests",
			path:   "results/testdata/results-0.10.tar.gz",
			show:   "failed",
			expect: 0,
		}, {
			desc:   "Shows passed tests",
			path:   "results/testdata/results-0.10.tar.gz",
			show:   "passed",
			expect: 1,
		}, {
			desc:      "Errs if missing results",
			path:      "results/testdata/results-0.10-missing-e2e.tar.gz",
			show:      "failed",
			expect:    0,
			expectErr: `failed to find results file "plugins/e2e/results/global/junit_01.xml" in archive`,
		}, {
			desc:      "Errs differently if not a tarfile",
			path:      "testdata/test_ssh.key",
			show:      "failed",
			expect:    0,
			expectErr: `failed to walk results archive: error getting next file in archive: archive/tar: invalid tar header`,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// do gettests
			sbc, err := NewSonobuoyClient(&rest.Config{}, nil)
			if err != nil {
				t.Fatalf("Failed to get Sonobuoy client")
			}

			f, err := os.Open(tC.path)
			if err != nil {
				panic(err)
			}

			var r io.Reader = f
			if strings.HasSuffix(tC.path, "tar.gz") {
				gzr, err := gzip.NewReader(f)
				if err != nil {
					t.Fatalf("Could not make a gzip reader: %v", err)
				}
				defer gzr.Close()
				r = gzr
			}

			results, err := sbc.GetTests(r, tC.show)
			switch {
			case err != nil && len(tC.expectErr) == 0:
				t.Fatalf("Expected nil error but got %v", err)
			case err != nil && len(tC.expectErr) > 0:
				if fmt.Sprint(err) != tC.expectErr {
					t.Errorf("Expected error \n\t%q\nbut got\n\t%q", tC.expectErr, err)
				}
			case err == nil && len(tC.expectErr) > 0:
				t.Fatalf("Expected error %v but got nil", tC.expectErr)
			default:
				// OK
			}

			if len(results) != tC.expect {
				t.Errorf("Expected %v results but got %v: %v", tC.expect, len(results), results)
			}
		})
	}
}
func TestString(t *testing.T) {
	testCases := []struct {
		desc   string
		cases  PrintableTestCases
		expect string
	}{
		{
			desc:   "No tests should report empty string",
			cases:  PrintableTestCases([]results.JUnitTestCase{}),
			expect: "",
		}, {
			desc:   "Nil tests should report empty string",
			cases:  PrintableTestCases(nil),
			expect: "",
		}, {
			desc: "Should not end with extra new line",
			cases: PrintableTestCases([]results.JUnitTestCase{
				{Name: "a"},
				{Name: "b"},
			}),
			expect: "a\nb",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out := tc.cases.String()
			if out != tc.expect {
				t.Errorf("Expected %q but got %q", tc.expect, out)
			}
		})
	}
}

func TestFocus(t *testing.T) {
	testCases := []struct {
		desc   string
		cases  PrintableTestCases
		expect string
	}{
		{
			desc:   "No test should result in an empty string",
			cases:  []results.JUnitTestCase{},
			expect: "",
		},
		{
			desc: "Single test with no regexp characters is not changed",
			cases: []results.JUnitTestCase{
				{Name: "this is a test"},
			},
			expect: "this is a test",
		},
		{
			desc: "Test with special regexp characters should be escaped",
			cases: []results.JUnitTestCase{
				{Name: "[sig-apps] test-1 (1.15) [Conformance]"},
			},
			expect: `\[sig-apps\] test-1 \(1\.15\) \[Conformance\]`,
		},
		{
			desc: "Multiple tests should be separated with '|'",
			cases: []results.JUnitTestCase{
				{Name: "[sig-apps] test-1 [Conformance]"},
				{Name: "[sig-apps] test-2"},
			},
			expect: `\[sig-apps\] test-1 \[Conformance\]|\[sig-apps\] test-2`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out := Focus(tc.cases)
			if out != tc.expect {
				t.Errorf("Expected %q but got %q", tc.expect, out)
			}
		})
	}
}
