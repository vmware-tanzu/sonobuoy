package results

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func Test_GojsonEventToItem(t *testing.T) {
	tests := []struct {
		name  string
		event testEvent
		want  *Item
	}{
		{
			name:  "Empty skipped",
			event: testEvent{},
			want:  nil,
		}, {
			name:  "Empty is unknown",
			event: testEvent{Test: "test"},
			want:  &Item{Name: "test", Status: StatusUnknown},
		}, {
			name:  "Pass",
			event: testEvent{Test: "test", Action: actionPass},
			want:  &Item{Name: "test", Status: StatusPassed},
		}, {
			name:  "Failure",
			event: testEvent{Test: "test", Action: actionFail},
			want:  &Item{Name: "test", Status: StatusFailed},
		}, {
			name:  "Skip",
			event: testEvent{Test: "test", Action: actionSkip},
			want:  &Item{Name: "test", Status: StatusSkipped},
		}, {
			name:  "Others unknown",
			event: testEvent{Test: "test", Action: "weirdvalue"},
			want:  &Item{Name: "test", Status: StatusUnknown},
		}, {
			name:  "Name is correct",
			event: testEvent{Test: "test", Action: actionPass},
			want:  &Item{Name: "test", Status: StatusPassed},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gojsonEventToItem(tt.event); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("gojsonEventToItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GojsonProcessReader(t *testing.T) {
	type args struct {
		r        io.Reader
		name     string
		metadata map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    Item
		wantErr bool
	}{
		{
			name: "Not JSON returns error",
			args: args{
				r:    strings.NewReader(`{`),
				name: "suite",
			},
			want:    Item{Name: "suite", Status: StatusUnknown, Metadata: map[string]string{"error": "unexpected EOF"}},
			wantErr: true,
		}, {
			name: "Single tests get parsed",
			args: args{
				r:    strings.NewReader(`{"action":"pass", "test":"test1"}`),
				name: "suite",
			},
			want: Item{
				Name: "suite", Status: StatusPassed,
				Items: []Item{
					{Name: "test1", Status: StatusPassed},
				},
			},
		}, {
			name: "Multiple tests get parsed in order",
			args: args{
				r:    strings.NewReader(`{"action":"pass", "test":"test1"}{"action":"pass", "test":"test2"}`),
				name: "suite",
			},
			want: Item{
				Name: "suite", Status: StatusPassed,
				Items: []Item{
					{Name: "test1", Status: StatusPassed},
					{Name: "test2", Status: StatusPassed},
				},
			},
		}, {
			name: "Aggregation doesnt happen in this method FYI",
			args: args{
				r:    strings.NewReader(`{"action":"fail", "test":"test1"}{"action":"pass", "test":"test2"}`),
				name: "suite",
			},
			want: Item{
				Name: "suite", Status: StatusPassed,
				Items: []Item{
					{Name: "test1", Status: StatusFailed},
					{Name: "test2", Status: StatusPassed},
				},
			},
		}, {
			name: "Real example",
			args: args{
				r: strings.NewReader(`{"Action":"run","Test":"TestListPods"}
{"Action":"output","Test":"TestListPods","Output":"=== RUN   TestListPods\n"}
{"Action":"output","Test":"TestListPods","Output":"    main_test.go:74: Creating NS custom-64d for test TestListPods\n"}
{"Action":"run","Test":"TestListPods/pod_list"}
{"Action":"output","Test":"TestListPods/pod_list","Output":"=== RUN   TestListPods/pod_list\n"}
{"Action":"run","Test":"TestListPods/pod_list/pods_from_kube-system"}
{"Action":"output","Test":"TestListPods/pod_list/pods_from_kube-system","Output":"=== RUN   TestListPods/pod_list/pods_from_kube-system\n"}
{"Action":"output","Test":"TestListPods/pod_list/pods_from_kube-system","Output":"    custom_test.go:35: found 12 pods\n"}
{"Action":"cont","Test":"TestListPods"}
{"Action":"output","Test":"TestListPods","Output":"=== CONT  TestListPods\n"}
{"Action":"output","Test":"TestListPods","Output":"    main_test.go:83: Deleting NS custom-64d for test TestListPods\n"}
{"Action":"output","Test":"TestListPods","Output":"--- PASS: TestListPods (0.03s)\n"}
{"Action":"output","Test":"TestListPods/pod_list","Output":"    --- PASS: TestListPods/pod_list (0.01s)\n"}
{"Action":"output","Test":"TestListPods/pod_list/pods_from_kube-system","Output":"        --- PASS: TestListPods/pod_list/pods_from_kube-system (0.01s)\n"}
{"Action":"pass","Test":"TestListPods/pod_list/pods_from_kube-system"}
{"Action":"pass","Test":"TestListPods/pod_list"}
{"Action":"pass","Test":"TestListPods"}
{"Action":"run","Test":"TestLongTest"}
{"Action":"output","Test":"TestLongTest","Output":"=== RUN   TestLongTest\n"}
{"Action":"output","Test":"TestLongTest","Output":"    main_test.go:74: Creating NS custom-715 for test TestLongTest\n"}
{"Action":"run","Test":"TestLongTest/pod_list"}
{"Action":"output","Test":"TestLongTest/pod_list","Output":"=== RUN   TestLongTest/pod_list\n"}
{"Action":"run","Test":"TestLongTest/pod_list/pods_from_kube-system"}
{"Action":"output","Test":"TestLongTest/pod_list/pods_from_kube-system","Output":"=== RUN   TestLongTest/pod_list/pods_from_kube-system\n"}
{"Action":"cont","Test":"TestLongTest"}
{"Action":"output","Test":"TestLongTest","Output":"=== CONT  TestLongTest\n"}
{"Action":"output","Test":"TestLongTest","Output":"    main_test.go:83: Deleting NS custom-715 for test TestLongTest\n"}
{"Action":"output","Test":"TestLongTest","Output":"--- PASS: TestLongTest (50.03s)\n"}
{"Action":"output","Test":"TestLongTest/pod_list","Output":"    --- PASS: TestLongTest/pod_list (50.02s)\n"}
{"Action":"output","Test":"TestLongTest/pod_list/pods_from_kube-system","Output":"        --- PASS: TestLongTest/pod_list/pods_from_kube-system (50.02s)\n"}
{"Action":"pass","Test":"TestLongTest/pod_list/pods_from_kube-system"}
{"Action":"pass","Test":"TestLongTest/pod_list"}
{"Action":"pass","Test":"TestLongTest"}
{"Action":"output","Output":"PASS\n"}
{"Action":"pass"}
`),
				name: "suite",
			},
			want: Item{
				Name: "suite", Status: StatusPassed,
				Items: []Item{
					{Name: "TestListPods/pod_list/pods_from_kube-system", Status: StatusPassed},
					{Name: "TestListPods/pod_list", Status: StatusPassed},
					{Name: "TestListPods", Status: StatusPassed},
					{Name: "TestLongTest/pod_list/pods_from_kube-system", Status: StatusPassed},
					{Name: "TestLongTest/pod_list", Status: StatusPassed},
					{Name: "TestLongTest", Status: StatusPassed},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := gojsonProcessReader(tt.args.r, tt.args.name, tt.args.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("gojsonProcessReader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("gojsonProcessReader() diff: %v", pretty.Compare(got, tt.want))
			}
		})
	}
}
