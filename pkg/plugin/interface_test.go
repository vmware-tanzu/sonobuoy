package plugin

import (
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestCombineUpdates(t *testing.T) {
	testCases := []struct {
		desc   string
		p1, p2 ProgressUpdate
		expect ProgressUpdate
	}{
		{
			desc:   "p2 overrides p1",
			p1:     ProgressUpdate{Total: 1, Completed: 1, Message: "foo"},
			p2:     ProgressUpdate{Total: 5, Completed: 5, Message: "bar"},
			expect: ProgressUpdate{Total: 5, Completed: 5, Message: "bar"},
		}, {
			desc:   "p2 appends p1",
			p1:     ProgressUpdate{Total: 1, Completed: 1, Failures: []string{"a"}, Message: "foo"},
			p2:     ProgressUpdate{AppendCompleted: 5, AppendFailing: []string{"B"}, Message: "bar"},
			expect: ProgressUpdate{Total: 1, Completed: 6, Failures: []string{"a", "B"}, Message: "bar"},
		}, {
			desc:   "p2 appends totals",
			p1:     ProgressUpdate{Total: 1, Completed: 1, Message: "foo"},
			p2:     ProgressUpdate{AppendTotals: true, AppendCompleted: 5, AppendFailing: []string{"a"}, Message: "bar"},
			expect: ProgressUpdate{Total: 7, Completed: 6, Failures: []string{"a"}, Message: "bar"},
		}, {
			desc:   "both appending",
			p1:     ProgressUpdate{AppendTotals: false, AppendCompleted: 2, AppendFailing: []string{"a", "b"}, Message: "foo"},
			p2:     ProgressUpdate{AppendTotals: true, AppendCompleted: 5, AppendFailing: []string{"c", "d"}, Message: "bar"},
			expect: ProgressUpdate{AppendTotals: true, AppendCompleted: 7, AppendFailing: []string{"a", "b", "c", "d"}, Message: "bar"},
		}, {
			desc:   "starting from empty",
			p1:     ProgressUpdate{},
			p2:     ProgressUpdate{Node: "nonempty", AppendTotals: true, AppendCompleted: 5, AppendFailing: []string{"c", "d"}, Message: "bar"},
			expect: ProgressUpdate{Node: "nonempty", Completed: 5, Total: 7, Failures: []string{"c", "d"}, Message: "bar"},
		}, {
			desc:   "appending failure included in totals but not completed",
			p1:     ProgressUpdate{},
			p2:     ProgressUpdate{Node: "nonempty", AppendTotals: true, AppendFailing: []string{"c", "d"}, Message: "bar"},
			expect: ProgressUpdate{Node: "nonempty", Total: 2, Failures: []string{"c", "d"}, Message: "bar"},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			output := CombineUpdates(tc.p1, tc.p2)
			if diff := pretty.Compare(tc.expect, output); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}

func TestProgressUpdateFormatting(t *testing.T) {
	testCases := []struct {
		desc   string
		p ProgressUpdate
		expect string
	}{
		{
			desc:   "Zero total should produce NO 'Remaining:'",
			p:     ProgressUpdate{Node: "nonempty", Completed: 1, Failures: []string{"c", "d"}, Message: "bar"},
			expect: "Passed:  1, Failed:  2",
		}, {
			desc:   "When Total matches the total, Remaining SHOULD printed",
			p:     ProgressUpdate{Node: "nonempty", Completed: 1, Total: 3, Failures: []string{"c", "d"}, Message: "bar"},
			expect: "Passed:  1, Failed:  2, Remaining:  0",
		}, {
			desc:   "When Total is less than failures+completed, Remaining should NOT be printed",
			p:     ProgressUpdate{Node: "nonempty", Completed: 2, Total: 1, Failures: []string{"c"}, Message: "bar"},
			expect: "Passed:  2, Failed:  1",
		}, {
			desc:   "When Total is more than failures+completed, Remaining SHOULD be printed",
			p:     ProgressUpdate{Node: "nonempty", Completed: 2, Total: 50, Failures: []string{"c", "d"}, Message: "bar"},
			expect: "Passed:  2, Failed:  2, Remaining: 46",
		}, {
			desc:   "When total is negative, Remaining should NOT be printed",
			p:     ProgressUpdate{Node: "nonempty", Completed: 2, Total: -1, Failures: []string{"c", "d"}, Message: "bar"},
			expect: "Passed:  2, Failed:  2",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := tc.p.FormatPluginProgress(); got != tc.expect{
				t.Fatalf("\n\n%s: expected '%s', got '%s'\n", tc.desc, tc.expect, got)
			}
		})
	}
}
