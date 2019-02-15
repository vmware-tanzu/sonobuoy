package client

import (
	"testing"

	"github.com/onsi/ginkgo/reporters"
)

func TestString(t *testing.T) {
	testCases := []struct {
		desc   string
		cases  PrintableTestCases
		expect string
	}{
		{
			desc:   "No tests should report empty string",
			cases:  PrintableTestCases([]reporters.JUnitTestCase{}),
			expect: "",
		}, {
			desc:   "Nil tests should report empty string",
			cases:  PrintableTestCases(nil),
			expect: "",
		}, {
			desc: "Should not end with extra new line",
			cases: PrintableTestCases([]reporters.JUnitTestCase{
				reporters.JUnitTestCase{Name: "a"},
				reporters.JUnitTestCase{Name: "b"},
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
