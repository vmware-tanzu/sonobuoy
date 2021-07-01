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

package client_test

import (
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/dynamic"
	"k8s.io/client-go/rest"
)

// Get a rest config from somewhere.
var cfg *rest.Config

// Example shows how to create a client and run Sonobuoy.
func Example() {
	// Get an APIHelper with default implementations from client-go.
	apiHelper, err := dynamic.NewAPIHelperFromRESTConfig(cfg)
	if err != nil {
		panic(err)
	}

	// client.NewSonobuoyClient returns a struct that implements the client.Interface.
	sonobuoy, err := client.NewSonobuoyClient(cfg, apiHelper)
	if err != nil {
		panic(err)
	}

	// Each feature of Sonobuoy requires a config to customize the behavior.

	// Build up a RunConfig struct.
	// The command line client provides default values with override flags.
	runConfig := client.RunConfig{
		GenConfig: client.GenConfig{
			PluginEnvOverrides: map[string]map[string]string{
				"e2e": {"E2E_FOCUS": "[sig-networking]"},
			},
			Config:          config.New(),
			EnableRBAC:      true,
			ImagePullPolicy: "Always",
		},
	}

	// Runs sonobuoy on the cluster configured in $HOME/.kube/config.
	if err = sonobuoy.Run(&runConfig); err != nil {
		panic(err)
	}
}
