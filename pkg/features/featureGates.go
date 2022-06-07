package features

import "os"

const (
	All = "SONOBUOY_ALL_FEATURES"

	PluginInstallation = "SONOBUOY_PLUGIN_INSTALLATION"

	WaitOutputProgressByDefault = "SONOBUOY_WAIT_PROGRESS"
)

var (
	featureDefaultMap = map[string]bool{
		PluginInstallation:          true,
		WaitOutputProgressByDefault: true,
	}
)

// Enabled returns if the named feature is enabled based on the current env and defaults.
func Enabled(feature string) bool {
	return enabledCore(feature, os.Getenv(All), os.Getenv(feature), featureDefaultMap)
}

// Extracted logic here for testing so we can modify the env and defaults easily.
func enabledCore(featureName, allEnv, featureEnv string, defaultMap map[string]bool) bool {
	// Allow features we default as true to be turned off while still relatively new so if major
	// bugs are found we have workarounds.
	if featureEnv == "false" {
		return false
	}
	return defaultMap[featureName] || allEnv == "true" || featureEnv == "true"
}
