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
	"fmt"
	"strings"

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/spf13/cobra"
)

// TODO (timothysc) - add a general override for --config for all commands which is the sonobuoy config

// AddNamespaceFlag initialises a namespace flag
func AddNamespaceFlag(str *string, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		str, "namespace", "n", config.DefaultPluginNamespace,
		"The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously.",
	)
}

// AddE2EModeFlag initialises a mode flag
func AddE2EModeFlag(mode *ops.Mode, cmd *cobra.Command) {
	*mode = ops.Conformance // default
	cmd.PersistentFlags().Var(
		mode, "e2e-mode",
		fmt.Sprintf("What mode to run sonobuoy in. [%s]", strings.Join(ops.GetModes(), ", ")),
	)
}

// AddSonobuoyImage initialises an image url flag
func AddSonobuoyImage(image *string, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		image, "sonobuoy-image", config.DefaultImage,
		"Container image override for the sonobuoy worker and container",
	)
}

// AddKubeconfigFlag adds a kubeconfig flag to the provided command
func AddKubeconfigFlag(cfg *Kubeconfig, cmd *cobra.Command) {
	// The default is the empty string (look in the environment)
	cmd.PersistentFlags().Var(cfg, "kubeconfig", "Explict kubeconfig file")
}
