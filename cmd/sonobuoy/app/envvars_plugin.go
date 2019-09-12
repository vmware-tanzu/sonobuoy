/*
Copyright the Sonobuoy project contributors 2019

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
)

// PluginEnvVars is a map of plugin (by name) mapped to a k-v map of env var name/values.
type PluginEnvVars map[string]map[string]string

func (i *PluginEnvVars) String() string { return fmt.Sprint((map[string]map[string]string)(*i)) }
func (i *PluginEnvVars) Type() string   { return "pluginenvvar" }

// Set parses the value from the CLI and places it into the internal map. Expected
// form is pluginName.envName=envValue. If no equals is found or it is the last character
// ("x.y" or "x.y=") then the env var will be saved internally as the empty string with
// the meaning that it will be removed from the plugins env vars.
func (i *PluginEnvVars) Set(str string) error {
	parts := strings.SplitN(str, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected form plugin.env=val but got %v parts when splitting by '.'", len(parts))
	}
	pluginName := parts[0]
	mapI := (map[string]map[string]string)(*i)

	if mapI == nil {
		mapI = map[string]map[string]string{}
	}
	if mapI[pluginName] == nil {
		mapI[pluginName] = map[string]string{}
	}

	envParts := strings.SplitN(parts[1], "=", 2)
	switch len(envParts) {
	case 1:
		mapI[pluginName][envParts[0]] = ""
	default:
		mapI[pluginName][envParts[0]] = envParts[1]
	}

	*i = mapI
	return nil
}
