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
			name:  "Empty is unknown",
			event: testEvent{},
			want:  &Item{Status: StatusUnknown},
		}, {
			name:  "Pass",
			event: testEvent{Action: actionPass},
			want:  &Item{Status: StatusPassed},
		}, {
			name:  "Failure",
			event: testEvent{Action: actionFail},
			want:  &Item{Status: StatusFailed},
		}, {
			name:  "Skip",
			event: testEvent{Action: actionSkip},
			want:  &Item{Status: StatusSkipped},
		}, {
			name:  "Others unknown",
			event: testEvent{Action: "weirdvalue"},
			want:  &Item{Status: StatusUnknown},
		}, {
			name:  "Name is correct",
			event: testEvent{Action: actionPass, Test: "testname"},
			want:  &Item{Status: StatusPassed, Name: "testname"},
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
