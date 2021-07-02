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
	"os"

	"github.com/vmware-tanzu/sonobuoy/pkg/config"
)

// NewGenConfig is a GenConfig using the default config and NonDisruptiveConformance mode
func NewGenConfig() *GenConfig {
	return &GenConfig{
		Config:     config.New(),
		EnableRBAC: true,
	}
}

// NewRunConfig is a RunConfig with DefaultGenConfig and and preflight checks enabled.
func NewRunConfig() *RunConfig {
	return &RunConfig{
		GenConfig: *NewGenConfig(),
	}
}

// NewDeleteConfig is a DeleteConfig using default images, RBAC enabled, and DeleteAll enabled.
func NewDeleteConfig() *DeleteConfig {
	return &DeleteConfig{
		Namespace:  config.DefaultNamespace,
		EnableRBAC: true,
		DeleteAll:  false,
	}
}

// NewLogConfig is a LogConfig with follow disabled and default images.
func NewLogConfig() *LogConfig {
	return &LogConfig{
		Follow:    false,
		Namespace: config.DefaultNamespace,
		Out:       os.Stdout,
	}
}
