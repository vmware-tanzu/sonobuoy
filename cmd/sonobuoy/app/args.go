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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	e2eFocusFlag = "e2e-focus"
	e2eSkipFlag  = "e2e-skip"
)

// AddNamespaceFlag initialises a namespace flag
func AddNamespaceFlag(str *string, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVarP(
		str, "namespace", "n", config.DefaultPluginNamespace,
		"The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously.",
	)
}

// AddModeFlag initialises a mode flag
func AddModeFlag(mode *ops.Mode, cmd *cobra.Command) {
	*mode = ops.Conformance // default
	cmd.PersistentFlags().Var(
		mode, "mode",
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
	cmd.MarkFlagFilename("kubeconfig")
}

// AddSonobuoyConfigFlag adds a SonobuoyConfig flag to the provided command
func AddSonobuoyConfigFlag(cfg *SonobuoyConfig, cmd *cobra.Command) {
	cmd.PersistentFlags().Var(
		cfg, "config",
		"path to a sonobuoy configuration JSON file. Overrides --mode",
	)
	cmd.MarkFlagFilename("config", "json")
}

// AddE2EConfig adds two arguments: --e2e-focus and --e2e-skip. These are not taken as pointers, as they are only used by GetE2EConfig.
func AddE2EConfig(cmd *cobra.Command) {
	cmd.PersistentFlags().String(
		e2eFocusFlag, "",
		"Specify the E2E_FOCUS flag to the conformance tests. Overrides --mode.",
	)
	cmd.PersistentFlags().String(
		e2eSkipFlag, "",
		"Specify the E2E_SKIP flag to the conformance tests. Overrides --mode.",
	)

}

// GetE2EConfig gets the E2EConfig from the mode, then overrides them with e2e-focus and e2e-skip if they are provided.
func GetE2EConfig(mode ops.Mode, cmd *cobra.Command) (*ops.E2EConfig, error) {
	flags := cmd.PersistentFlags()
	cfg := mode.Get().E2EConfig
	if flags.Changed(e2eFocusFlag) {
		focus, err := flags.GetString(e2eFocusFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve focus flag")
		}
		cfg.Focus = focus
	}

	if flags.Changed(e2eSkipFlag) {
		skip, err := flags.GetString(e2eSkipFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve skip flag")
		}
		cfg.Skip = skip
	}
	return &cfg, nil
}
