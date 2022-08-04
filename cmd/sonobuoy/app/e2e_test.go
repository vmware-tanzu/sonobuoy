package app

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/sirupsen/logrus"
)

func TestFilterTests(t *testing.T) {
	testCases := []struct {
		desc        string
		input       []string
		focus, skip string
		expect      []string
	}{
		{
			desc:   "No focus or skip",
			input:  []string{"abc", "bcd", "cde"},
			expect: []string{"abc", "bcd", "cde"},
		}, {
			desc:   "Focus",
			input:  []string{"abc", "bcd", "cde"},
			focus:  "b",
			expect: []string{"abc", "bcd"},
		}, {
			desc:   "Skip",
			input:  []string{"abc", "bcd", "cde"},
			skip:   "b",
			expect: []string{"cde"},
		}, {
			desc:   "Focus and Skip",
			input:  []string{"abc", "bcd", "cde"},
			focus:  "b",
			skip:   "a",
			expect: []string{"bcd"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var f, s *regexp.Regexp
			var err error
			if len(tc.focus) > 0 {
				f, err = regexp.Compile(tc.focus)
				if err != nil {
					t.Fatalf("Failed to compile test focus value: %v", err)
				}
			}
			if len(tc.skip) > 0 {
				s, err = regexp.Compile(tc.skip)
				if err != nil {
					t.Fatalf("Failed to compile test skip value: %v", err)
				}
			}
			output := filterTests(tc.input, f, s)
			if diff := pretty.Compare(tc.expect, output); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}

func TestPrintTestList(t *testing.T) {
	testCases := []struct {
		desc   string
		input  []string
		expect []string
		mode   string
	}{
		{
			desc:   "Tests print all test names",
			input:  []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
			mode:   "tests",
			expect: []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
		}, {
			desc:   "Just tags",
			input:  []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
			mode:   "tags",
			expect: []string{"[tag1]", "[tag2]", "[tag3]", "[tag4]"},
		}, {
			desc:   "Tags and counts",
			input:  []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
			mode:   "tagCounts",
			expect: []string{"[tag1]:2", "[tag2]:2", "[tag3]:1", "[tag4]:1"},
		}, {
			desc:   "Defaults to test output",
			input:  []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
			mode:   "badinput",
			expect: []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var b bytes.Buffer
			printTestList(&b, tc.mode, tc.input)
			if diff := pretty.Compare(b.String(), fmt.Sprintln(strings.Join(tc.expect, "\n"))); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}

func TestTagCountsFromList(t *testing.T) {
	testCases := []struct {
		desc   string
		input  []string
		expect map[string]int
	}{
		{
			desc:   "Multiple tags in a test",
			input:  []string{"[tag1]abc[tag2]", "[tag2]bcd[tag3]", "[tag4]c[tag1]de"},
			expect: map[string]int{"[tag1]": 2, "[tag2]": 2, "[tag3]": 1, "[tag4]": 1},
		}, {
			desc:   "Tags with special chars",
			input:  []string{"[feature: something]abc[tag2]"},
			expect: map[string]int{`[feature: something]`: 1, `[tag2]`: 1},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := tagCountsFromList(tc.input)
			if diff := pretty.Compare(output, tc.expect); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}

func TestGetTestsRedirects(t *testing.T) {
	testCases := []struct {
		desc       string
		getVersion string
		expect     []string
	}{
		{
			desc:       "Can get static data",
			getVersion: "v1",
			expect:     []string{"v1 data line 1", "v1 data line 2"},
		}, {
			desc:       "Can get redirected data",
			getVersion: "v0",
			expect:     []string{"v1 data line 1", "v1 data line 2"},
		}, {
			desc:       "Can get modified redirected data",
			getVersion: "v2",
			expect:     []string{"v1 data line 2", "v2 data line 1"},
		}, {
			desc:       "Can handle multiple redirects with modifications and it remains sorted",
			getVersion: "v3",
			expect:     []string{"_v3 data line 1", "v1 data line 2", "v2 data line 1"},
		},
	}

	logrus.SetLevel(logrus.TraceLevel)
	mux := http.NewServeMux()

	mux.HandleFunc("/v0.gz", func(w http.ResponseWriter, req *http.Request) {
		gw := gzip.NewWriter(w)
		gw.Write([]byte("#v1\n"))
		gw.Close()
	})
	mux.HandleFunc("/v1.gz", func(w http.ResponseWriter, req *http.Request) {
		gw := gzip.NewWriter(w)
		gw.Write([]byte("v1 data line 1\nv1 data line 2"))
		gw.Close()
	})
	mux.HandleFunc("/v2.gz", func(w http.ResponseWriter, req *http.Request) {
		gw := gzip.NewWriter(w)
		gw.Write([]byte("#v1\n+v2 data line 1\n-v1 data line 1"))
		gw.Close()
	})
	mux.HandleFunc("/v3.gz", func(w http.ResponseWriter, req *http.Request) {
		gw := gzip.NewWriter(w)
		gw.Write([]byte("#v2\n+_v3 data line 1"))
		gw.Close()
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output, err := getTests(e2eInputOnline, ts.URL, tc.getVersion)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}
			if diff := pretty.Compare(tc.expect, output); diff != "" {
				t.Fatalf("Expected %v but got diff:\n\n%s\n", tc.expect, diff)
			}
		})
	}
}
