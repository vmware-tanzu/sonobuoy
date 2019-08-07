package discovery

import (
	"testing"

	"github.com/heptio/sonobuoy/pkg/client/results"
	"github.com/heptio/sonobuoy/pkg/config"

	"github.com/kylelemons/godebug/pretty"
)

func TestGetPodLogNamespaceFilter(t *testing.T) {
	testCases := []struct {
		name     string
		input    *config.Config
		expected string
	}{
		{
			name: "Provided both Namespaces and SonobuoyNamespace will generate filter to concatenate them by OR",
			input: &config.Config{
				Namespace: config.DefaultNamespace,
				Limits: config.LimitConfig{
					PodLogs: config.PodLogLimits{
						Namespaces:        ".*",
						SonobuoyNamespace: &[]bool{true}[0],
					},
				},
			},
			expected: ".*|" + config.DefaultNamespace,
		},
		{
			name: "Provided only Namespaces will generate filter which includes only Namespaces regex",
			input: &config.Config{
				Namespace: config.DefaultNamespace,
				Limits: config.LimitConfig{
					PodLogs: config.PodLogLimits{
						Namespaces:        ".*",
						SonobuoyNamespace: &[]bool{false}[0],
					},
				},
			},
			expected: ".*",
		},
		{
			name: "Provided neither Namespaces nor SonobuoyNamespace will output an empty filter",
			input: &config.Config{
				Namespace: config.DefaultNamespace,
				Limits: config.LimitConfig{
					PodLogs: config.PodLogLimits{
						Namespaces:        "",
						SonobuoyNamespace: &[]bool{false}[0],
					},
				},
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		nsFilter := getPodLogNamespaceFilter(tc.input)

		if nsFilter != tc.expected {
			t.Errorf("GetPodLogNamespaceFilter() expected %s, got %s", tc.expected, nsFilter)
		}
	}
}

func TestStatusCounts(t *testing.T) {
	tcs := []struct {
		desc        string
		input       *results.Item
		inputCounts map[string]int
		expected    map[string]int
	}{
		{
			desc:        "Nil item",
			inputCounts: map[string]int{},
			expected:    map[string]int{},
		}, {
			desc:        "Single leaf",
			input:       &results.Item{Status: "foo"},
			inputCounts: map[string]int{},
			expected:    map[string]int{"foo": 1},
		}, {
			desc: "Multiple leafs",
			input: &results.Item{
				Status: "not-leaf-dont-count",
				Items: []results.Item{
					{Status: "foo"},
					{Status: "foo"},
					{Status: "bar"},
				},
			},
			inputCounts: map[string]int{},
			expected:    map[string]int{"foo": 2, "bar": 1},
		}, {
			desc: "Multiple leafs, varying depth",
			input: &results.Item{
				Status: "not-leaf-dont-count",
				Items: []results.Item{
					{Status: "foo"},
					{Status: "foo"},
					{
						Status: "also-not-leaf",
						Items: []results.Item{
							{Status: "foo"},
							{Status: "bar"},
						},
					},
				},
			},
			inputCounts: map[string]int{},
			expected:    map[string]int{"foo": 3, "bar": 1},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			statusCounts(tc.input, tc.inputCounts)
			if diff := pretty.Compare(tc.expected, tc.inputCounts); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
