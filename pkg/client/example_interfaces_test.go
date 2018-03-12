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
	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"k8s.io/client-go/rest"
)

// Get a rest config from somewhere.
var cfg *rest.Config

// Example shows how to create a client and run Sonobuoy.
func Example() {
	// client.NewSonobuoyClient returns a struct that implements the client.Interface.
	sonobuoy, err := client.NewSonobuoyClient(cfg)
	if err != nil {
		panic(err)
	}

	// Each feature of Sonobuoy requires a config to customize the behavior.

	// Build up a RunConfig struct.
	// The command line client provides default values with override flags.
	runConfig := client.RunConfig{
		GenConfig: client.GenConfig{
			E2EConfig: &client.E2EConfig{
				Focus: "[sig-networking]",
				Skip:  "",
			},
			Config:          config.New(),
			Image:           config.DefaultImage,
			Namespace:       config.DefaultNamespace,
			EnableRBAC:      true,
			ImagePullPolicy: "Always",
		},
	}

	// Runs sonobuoy on the cluster configured in $HOME/.kube/config.
	if err = sonobuoy.Run(&runConfig); err != nil {
		panic(err)
	}
}
