package args

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubeconfig represents an explict or implict kubeconfig
type Kubeconfig struct {
	*clientcmd.ClientConfigLoadingRules
}

// AddKubeconfigFlag adds a kubeconfig flag to the provided command
func AddKubeconfigFlag(cfg *Kubeconfig, cmd *cobra.Command) {
	cmd.PersistentFlags().Var(cfg, "kubeconfig", "Explict kubeconfig file")
}

// String needed for pflag.Value
func (c *Kubeconfig) String() string {
	if c.ClientConfigLoadingRules != nil {
		return c.ExplicitPath
	}
	return ""
}

// Type needed for pflag.Value
func (c *Kubeconfig) Type() string { return "Kubeconfig" }

// Set sets the explicit path of the loader to the provided config file
func (c *Kubeconfig) Set(str string) error {
	if c.ClientConfigLoadingRules == nil {
		c.ClientConfigLoadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	c.ExplicitPath = str
	return nil
}

// Get returns a rest Config, possibly based on a provided config
func (c *Kubeconfig) Get() (*rest.Config, error) {
	if c.ClientConfigLoadingRules == nil {
		c.ClientConfigLoadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(c, configOverrides)
	return kubeConfig.ClientConfig()
}
