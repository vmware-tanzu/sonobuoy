package image

import (
	"github.com/spf13/cobra"
)

type ID string

// AddFlag adds a sonobuoy-image flag to existing command
func AddFlag(id *ID, cmd *cobra.Command) {
	cmd.PersistentFlags().Var(
		id, "sonobuoy-image",
		"what container image to use for the sonobuoy worker and container",
	)
}

// String needed for pflag.Value
func (i *ID) String() string { return string(*i) }

// Type needed for pflag.Value
func (i *ID) Type() string { return "Sonobuoy Container Image ID" }

//Set the image ID. Returns an error when not a valid image ID.
func (i *ID) Set(id string) error {
	*i = ID(id)
	return nil
}

// Get returns the provided ID, or a default if none has been provided
func (i *ID) Get() string {
	if i == nil || *i == "" {
		return "gcr.io/heptio-images/sonobuoy:master"
	}

	return i.String()
}
