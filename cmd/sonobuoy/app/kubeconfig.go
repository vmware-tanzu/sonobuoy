/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	// Add auth providers
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// Kubeconfig represents an explict or implict kubeconfig
type Kubeconfig struct {
	*clientcmd.ClientConfigLoadingRules
}

// Make sure Kubeconfig implements Value properly
var _ pflag.Value = &Kubeconfig{}

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
