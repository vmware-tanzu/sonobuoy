package kubeconfig

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config represents an explict or implict kubeconfig
type Config struct {
	*clientcmd.ClientConfigLoadingRules
}

// String needed for pflag.Value
func (c *Config) String() string {
	if c.ClientConfigLoadingRules != nil {
		return c.ExplicitPath
	}
	return ""
}

// Type needed for pflag.Value
func (c *Config) Type() string { return "Kubeconfig" }

// Set sets the explicit path of the loader to the provided config file
func (c *Config) Set(str string) error {
	if c.ClientConfigLoadingRules == nil {
		c.ClientConfigLoadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	c.ExplicitPath = str
	return nil
}

// AddFlag adds a kubeconfig flag to the provided command
func AddFlag(cfg *Config, cmd *cobra.Command) {
	cmd.PersistentFlags().Var(cfg, "kubeconfig", "Explict kubeconfig file")
}

// Get returns a rest Config, possibly based on a provided config
func (c *Config) Get() (*rest.Config, error) {
	if c.ClientConfigLoadingRules == nil {
		c.ClientConfigLoadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(c, configOverrides)
	return kubeConfig.ClientConfig()
}
