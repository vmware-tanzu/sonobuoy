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
			p2:     ProgressUpdate{AppendTotals: true, AppendCompleted: 5, Message: "bar"},
			expect: ProgressUpdate{Total: 6, Completed: 6, Message: "bar"},
		}, {
			desc:   "both appending",
			p1:     ProgressUpdate{AppendTotals: false, AppendCompleted: 2, AppendFailing: []string{"a", "b"}, Message: "foo"},
			p2:     ProgressUpdate{AppendTotals: true, AppendCompleted: 5, AppendFailing: []string{"c", "d"}, Message: "bar"},
			expect: ProgressUpdate{AppendTotals: true, AppendCompleted: 7, AppendFailing: []string{"a", "b", "c", "d"}, Message: "bar"},
		}, {
			desc:   "starting from empty",
			p1:     ProgressUpdate{},
			p2:     ProgressUpdate{Node: "nonempty", AppendTotals: true, AppendCompleted: 5, AppendFailing: []string{"c", "d"}, Message: "bar"},
			expect: ProgressUpdate{Node: "nonempty", Completed: 5, Total: 5, Failures: []string{"c", "d"}, Message: "bar"},
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
