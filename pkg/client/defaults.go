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

package client

import (
	"github.com/heptio/sonobuoy/pkg/config"
)

var (
	// DefaultGenConfig is a GenConfig using the default config and Conformance mode
	DefaultGenConfig = GenConfig{
		E2EConfig:  &ConformanceModeConfig.E2EConfig,
		Config:     config.NewWithDefaults(),
		Image:      config.DefaultImage,
		Namespace:  config.DefaultPluginNamespace,
		EnableRBAC: true,
	}
	// DefaultRunConfig is a RunConfig with DefaultGenConfig and and preflight checks enabled.
	DefaultRunConfig = RunConfig{
		GenConfig:     DefaultGenConfig,
		SkipPreflight: false,
	}

	// DefaultDeleteConfig is a DeleteConfig using default images, RBAC enabled, and DeleteAll enabled.
	DefaultDeleteConfig = DeleteConfig{
		Namespace:  config.DefaultImage,
		EnableRBAC: true,
		DeleteAll:  false,
	}
	// DefaultLogConfig is a LogConfig with follow disabled and default images.
	DefaultLogConfig = LogConfig{
		Follow:    false,
		Namespace: config.DefaultPluginNamespace,
	}
)
