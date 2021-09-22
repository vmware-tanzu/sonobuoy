package app

import "os"

const (
	FeaturesAll = "SONOBUOY_ALL_FEATURES"

	FeaturePluginInstallation = "SONOBUOY_PLUGIN_INSTALLATION"

	FeatureWaitOutputProgressByDefault = "SONOBUOY_WAIT_PROGRESS"
)

var (
	featureDefaultMap = map[string]bool{
		FeaturePluginInstallation:          true,
		FeatureWaitOutputProgressByDefault: true,
	}
)

// featureEnabled returns if the named feature is enabled based on the current env and defaults.
func featureEnabled(feature string) bool {
	return featureEnabledCore(feature, os.Getenv(FeaturesAll), os.Getenv(feature), featureDefaultMap)
}

// Extracted logic here for testing so we can modify the env and defaults easily.
func featureEnabledCore(featureName, allEnv, featureEnv string, defaultMap map[string]bool) bool {
	// Allow features we default as true to be turned off while still relatively new so if major
	// bugs are found we have workarounds.
	if featureEnv == "false" {
		return false
	}
	return defaultMap[featureName] || allEnv == "true" || featureEnv == "true"
}
