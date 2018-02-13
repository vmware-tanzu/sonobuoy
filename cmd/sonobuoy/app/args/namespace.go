package args

import "github.com/spf13/cobra"

// Namespace represents a Kubernetes namespace
type Namespace string

const usage = "the namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously."

//AddNamespaceFlag adds a Namespace flag to the given command
func AddNamespaceFlag(namespace *Namespace, cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(
		namespace,
		"namespace",
		"n",
		usage,
	)
}

// String needed for pflag.Value
func (n *Namespace) String() string { return string(*n) }

// Type needed for pflag.Value
func (n *Namespace) Type() string { return "Namespace" }

// Set the namespace with a given string.
func (n *Namespace) Set(str string) error {
	*n = Namespace(str)
	return nil
}

// Get returns the namespace, or a default one if none is set
func (n *Namespace) Get() string {
	if n != nil && *n != "" {
		return string(*n)
	}

	return "heptio-sonobuoy"
}
