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
	"github.com/spf13/pflag"
	"k8s.io/api/core/v1"
)

// AddNamespaceFlag initialises a namespace flag.
func AddNamespaceFlag(str *string, flags *pflag.FlagSet) {
	flags.StringVarP(
		str, "namespace", "n", config.DefaultNamespace,
		"The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously.",
	)
}

// AddModeFlag initialises a mode flag.
// The mode is a preset configuration of sonobuoy configuration and e2e configuration variables.
// Mode can be partially or fully overridden by specifying config, e2e-focus, and e2e-skip.
// The variables specified by those flags will overlay the defaults provided by the given mode.
func AddModeFlag(mode *ops.Mode, flags *pflag.FlagSet) {
	*mode = ops.Conformance // default
	flags.VarP(
		mode, "mode", "m",
		fmt.Sprintf("What mode to run sonobuoy in. Valid modes are %s.", strings.Join(ops.GetModes(), ", ")),
	)
}

// AddSonobuoyImage initialises an image url flag.
func AddSonobuoyImage(image *string, flags *pflag.FlagSet) {
	flags.StringVar(
		image, "sonobuoy-image", config.DefaultImage,
		"Container image override for the sonobuoy worker and container.",
	)
}

// AddKubeConformanceImage initialises an image url flag.
func AddKubeConformanceImage(image *string, flags *pflag.FlagSet) {
	flags.StringVar(
		image, "kube-conformance-image", config.DefaultKubeConformanceImage,
		"Container image override for the kube conformance image.",
	)
}

// AddKubeconfigFlag adds a kubeconfig flag to the provided command.
func AddKubeconfigFlag(cfg *Kubeconfig, flags *pflag.FlagSet) {
	// The default is the empty string (look in the environment)
	flags.Var(cfg, "kubeconfig", "Path to explict kubeconfig file.")
}

// AddSonobuoyConfigFlag adds a SonobuoyConfig flag to the provided command.
func AddSonobuoyConfigFlag(cfg *SonobuoyConfig, flags *pflag.FlagSet) {
	flags.Var(
		cfg, "config",
		"Path to a sonobuoy configuration JSON file. Overrides --mode.",
	)
}

const (
	e2eFocusFlag    = "e2e-focus"
	e2eSkipFlag     = "e2e-skip"
	e2eParallelFlag = "e2e-parallel"
)

// AddE2EConfigFlags adds three arguments: --e2e-focus, --e2e-skip and
// --e2e-parallel. These are not taken as pointers, as they are only used by
// GetE2EConfig. Instead, they are returned as a Flagset which should be passed
// to GetE2EConfig. The returned flagset will be added to the passed in flag set.
//
// e2e-parallel is added as a hidden flag that should only be used by "power"
// users. Using e2e-parallel incorrectly has the potential to destroy clusters!
func AddE2EConfigFlags(flags *pflag.FlagSet) *pflag.FlagSet {
	e2eFlags := pflag.NewFlagSet("e2e", pflag.ExitOnError)
	modeName := ops.Conformance
	defaultMode := modeName.Get()
	e2eFlags.String(
		e2eFocusFlag, defaultMode.E2EConfig.Focus,
		"Specify the E2E_FOCUS flag to the conformance tests. Overrides --mode.",
	)
	e2eFlags.String(
		e2eSkipFlag, defaultMode.E2EConfig.Skip,
		"Specify the E2E_SKIP flag to the conformance tests. Overrides --mode.",
	)
	e2eFlags.String(
		e2eParallelFlag, defaultMode.E2EConfig.Parallel,
		"Specify the E2E_PARALLEL flag to the conformance tests. Overrides --mode.",
	)
	e2eFlags.MarkHidden(e2eParallelFlag)
	flags.AddFlagSet(e2eFlags)
	return e2eFlags
}

// GetE2EConfig gets the E2EConfig from the mode, then overrides them with e2e-focus, e2e-skip and e2e-parallel if they
// are provided. We can't rely on the zero value of the flags, as "" is a valid focus, skip or parallel value.
func GetE2EConfig(mode ops.Mode, flags *pflag.FlagSet) (*ops.E2EConfig, error) {
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

	if flags.Changed(e2eParallelFlag) {
		parallel, err := flags.GetString(e2eParallelFlag)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't retrieve parallel flag")
		}
		cfg.Parallel = parallel
	}
	return &cfg, nil
}

// AddRBACModeFlags adds an E2E Argument with the provided default.
func AddRBACModeFlags(mode *RBACMode, flags *pflag.FlagSet, defaultMode RBACMode) {
	*mode = defaultMode // default
	flags.Var(
		mode, "rbac",
		// Doesn't use the map in app.rbacModeMap to preserve order so we can add an explanation for detect.
		"Whether to enable rbac on Sonobuoy. Valid modes are Enable, Disable, and Detect (query the server to see whether to enable Sonobuoy).",
	)
}

// AddSkipPreflightFlag adds a boolean flag to skip preflight checks.
func AddSkipPreflightFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "skip-preflight", false,
		"If true, skip all checks before kicking off the sonobuoy run.",
	)
}

// AddDeleteAllFlag adds a boolean flag for deleting everything (including E2E tests).
func AddDeleteAllFlag(flag *bool, flags *pflag.FlagSet) {
	flags.BoolVar(
		flag, "all", false,
		"In addition to deleting Sonobuoy namespaces, also clean up dangling e2e- namespaces.",
	)
}

// AddImagePullPolicyFlag adds a boolean flag for deleting everything (including E2E tests).
func AddImagePullPolicyFlag(policy *ImagePullPolicy, flags *pflag.FlagSet) {
	*policy = ImagePullPolicy(v1.PullAlways) //default
	flags.Var(
		policy, "image-pull-policy",
		fmt.Sprintf("The ImagePullPolicy Sonobuoy should use for the aggregators and workers. Valid options are %s.", strings.Join(ValidPullPolicies(), ", ")),
	)
}
