package worker

import (
	"github.com/spf13/viper"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		desc        string
		expectedCfg *plugin.WorkerConfig
		env         map[string]string
	}{
		{
			desc: "No environment variables results in default config values",
			expectedCfg: &plugin.WorkerConfig{
				ResultsDir:          "/tmp/results",
				ProgressUpdatesPort: "8099",
			},
		},
		{
			desc: "Aggregator URL is set in config if env var is set",
			expectedCfg: &plugin.WorkerConfig{
				AggregatorURL:       "aggregator",
				ResultsDir:          plugin.ResultsDir,
				ProgressUpdatesPort: defaultProgressUpdatesPort,
			},
			env: map[string]string{
				"AGGREGATOR_URL": "aggregator",
			},
		},
		{
			desc: "Aggregator URL is set in config if only deprecated master url env var is set",
			expectedCfg: &plugin.WorkerConfig{
				AggregatorURL:       "master",
				ResultsDir:          plugin.ResultsDir,
				ProgressUpdatesPort: defaultProgressUpdatesPort,
			},
			env: map[string]string{
				"MASTER_URL": "master",
			},
		},
		{
			desc: "Aggregator URL env var takes precedence if both new and deprecated env vars set",
			expectedCfg: &plugin.WorkerConfig{
				AggregatorURL:       "aggregator",
				ResultsDir:          plugin.ResultsDir,
				ProgressUpdatesPort: defaultProgressUpdatesPort,
			},
			env: map[string]string{
				"MASTER_URL":     "master",
				"AGGREGATOR_URL": "aggregator",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			viper.Reset()
			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("unable to set environment variable %q to value %q", k, v)
				}
			}
			defer func() {
				for k := range tc.env {
					if err := os.Unsetenv(k); err != nil {
						t.Fatalf("unable to unset environment variable %q", k)
					}
				}
			}()

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("unexepected err from LoadConfig %q", err)
			}
			if !reflect.DeepEqual(cfg, tc.expectedCfg) {
				t.Fatalf("expected config to be %q, got %q", tc.expectedCfg, cfg)
			}
		})
	}
}
