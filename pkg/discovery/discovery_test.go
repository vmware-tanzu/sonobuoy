package discovery

import (
	"github.com/heptio/sonobuoy/pkg/config"
	"testing"
)

func TestGetPodLogNamespaceFilter(t *testing.T) {
	testCases := []struct {
		name     string
		input    *config.Config
		expected string
	}{
		{
			name:     "Provided both Namespaces and SonobuoyNamespace will generate filter to concatenate them by OR",
			input:    &config.Config{
				Namespace: config.DefaultNamespace,
				Limits: config.LimitConfig{
					PodLogs: config.PodLogLimits{
						Namespaces: ".*",
						SonobuoyNamespace: &[]bool{true}[0],
					},
				},
			},
			expected: ".*|" + config.DefaultNamespace,
		},
		{
			name:     "Provided only Namespaces will generate filter which includes only Namespaces regex",
			input:    &config.Config{
				Namespace: config.DefaultNamespace,
				Limits: config.LimitConfig{
					PodLogs: config.PodLogLimits{
						Namespaces: ".*",
						SonobuoyNamespace: &[]bool{false}[0],
					},
				},
			},
			expected: ".*",
		},
		{
			name:     "Provided neither Namespaces nor SonobuoyNamespace will output an empty filter",
			input:    &config.Config{
				Namespace: config.DefaultNamespace,
				Limits: config.LimitConfig{
					PodLogs: config.PodLogLimits{
						Namespaces: "",
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