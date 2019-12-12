package discovery

import (
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	pluginaggregation "github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"

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

func TestIntegrateResultsIntoStatus(t *testing.T) {
	testCases := []struct {
		desc         string
		status       *pluginaggregation.Status
		expectStatus *pluginaggregation.Status
		pluginName   string
		item         *results.Item
	}{
		{
			desc: "Updates correct plugin by name for global plugins",
			status: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{
						Plugin: "foo", Node: "global",
					},
				},
			},
			expectStatus: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{
						Plugin: "foo", Node: "global",
						ResultStatus:       "passed",
						ResultStatusCounts: map[string]int{"passed": 1},
					},
				},
			},
			pluginName: "foo",
			item:       &results.Item{Status: "passed"},
		}, {
			desc: "Wont update any if no match by name",
			status: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{
						Plugin: "foo", Node: "global",
					},
				},
			},
			expectStatus: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{
						Plugin: "foo", Node: "global",
					},
				},
			},
			pluginName: "notfoo",
			item:       &results.Item{Status: "passed"},
		}, {
			desc: "Updates each daemonsets node in item",
			status: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{Plugin: "foo", Node: "node1"},
					{Plugin: "foo", Node: "node2"},
				},
			},
			expectStatus: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{
						Plugin: "foo", Node: "node1",
						ResultStatus:       "passed",
						ResultStatusCounts: map[string]int{"passed": 1},
					},
					{
						Plugin: "foo", Node: "node2",
						ResultStatus:       "failed",
						ResultStatusCounts: map[string]int{"failed": 1},
					},
				},
			},
			pluginName: "foo",
			item: &results.Item{
				Status: "failed",
				Items: []results.Item{
					{Name: "node1", Status: "passed"},
					{Name: "node2", Status: "failed"},
				},
			},
		}, {
			desc: "If daemonset missing node then no change to that value",
			status: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{Plugin: "foo", Node: "node1"},
					{Plugin: "foo", Node: "node2"},
				},
			},
			expectStatus: &pluginaggregation.Status{
				Plugins: []pluginaggregation.PluginStatus{
					{
						Plugin: "foo", Node: "node1",
						ResultStatus:       "passed",
						ResultStatusCounts: map[string]int{"passed": 1},
					},
					{Plugin: "foo", Node: "node2"},
				},
			},
			pluginName: "foo",
			item: &results.Item{
				Status: "failed",
				Items: []results.Item{
					{Name: "node1", Status: "passed"},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			integrateResultsIntoStatus(tc.status, tc.pluginName, tc.item)
			if diff := pretty.Compare(tc.status, tc.expectStatus); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
