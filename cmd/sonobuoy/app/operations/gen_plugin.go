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

package operations

import (
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/loader"
)

const (
	placeholderHostname  = "<hostname>"
	placeholderNamespace = "sonobuoy"
)

// GenPluginConfig are the input options for running
type GenPluginConfig struct {
	Paths      []string
	PluginName string
}

// GeneratePluginManifest partially initialises a plugin, then dumps out plugin's manifest
func GeneratePluginManifest(cfg GenPluginConfig) ([]byte, error) {
	plugins, err := loader.LoadAllPlugins(
		placeholderNamespace,
		cfg.Paths,
		[]plugin.Selection{{Name: cfg.PluginName}},
	)
	if err != nil {
		return nil, err
	}

	if len(plugins) != 1 {
		return nil, fmt.Errorf("expected 1 plugin, got %v", len(plugins))
	}

	selectedPlugin := plugins[0]
	bytes, err := selectedPlugin.FillTemplate(placeholderHostname)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}
