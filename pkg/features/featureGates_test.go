package features

import (
	"testing"
)

func Test_FeatureEnabled(t *testing.T) {
	// Choose arbitrary feature to test against
	feature := "foo"
	tests := []struct {
		name       string
		want       bool
		allEnv     string
		featureEnv string
		defaultVal bool
	}{
		{
			name:   "All false",
			want:   false,
			allEnv: "false", featureEnv: "false", defaultVal: false,
		}, {
			name:   "Explicit false overrides all else",
			want:   false,
			allEnv: "true", featureEnv: "false", defaultVal: true,
		}, {
			name:   "Explicit true overrides all else",
			want:   true,
			allEnv: "false", featureEnv: "true", defaultVal: false,
		}, {
			name:   "All true overrides default",
			want:   true,
			allEnv: "true", featureEnv: "", defaultVal: false,
		}, {
			name:   "Can default to true even if all env is false",
			want:   true,
			allEnv: "false", featureEnv: "", defaultVal: true,
		}, {
			name:   "Default value used if others empty",
			want:   true,
			allEnv: "", featureEnv: "", defaultVal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := enabledCore(feature, tt.allEnv, tt.featureEnv, map[string]bool{feature: tt.defaultVal}); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
