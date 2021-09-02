package app

import "os"

const (
	FeaturesAll = "SONOBUOY_ALL_FEATURES"

	FeaturePluginInstallation = "SONOBUOY_PLUGIN_INSTALLATION"

	FeatureWaitOutputProgressByDefault = "SONOBUOY_WAIT_PROGRESS"
)

func featureEnabled(feature string) bool {
	return os.Getenv(FeaturesAll) == "true" || os.Getenv(feature) == "true"
}
