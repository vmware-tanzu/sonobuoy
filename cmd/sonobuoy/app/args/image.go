package args

import (
	"github.com/spf13/cobra"
)

// SonobuoyImage is the name of
type SonobuoyImage string

// AddSonobuoyImageFlag adds a sonobuoy-image flag to existing command
func AddSonobuoyImageFlag(id *SonobuoyImage, cmd *cobra.Command) {
	cmd.PersistentFlags().Var(
		id, "sonobuoy-image",
		"what container image to use for the sonobuoy worker and container",
	)
}

// String needed for pflag.Value
func (i *SonobuoyImage) String() string { return string(*i) }

// Type needed for pflag.Value
func (i *SonobuoyImage) Type() string { return "Sonobuoy Container Image" }

//Set the image SonobuoyImage. Returns an error when not a valid image SonobuoyImage.
func (i *SonobuoyImage) Set(id string) error {
	*i = SonobuoyImage(id)
	return nil
}

// Get returns the provided SonobuoyImage, or a default if none has been provided
func (i *SonobuoyImage) Get() string {
	if i == nil || *i == "" {
		return "gcr.io/heptio-images/sonobuoy:master"
	}

	return i.String()
}
