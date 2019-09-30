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
	"fmt"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
)

// Mode identifies a specific mode of running Sonobuoy.
// A mode is a defined configuration of plugins and E2E Focus and Config.
// Modes form the base level defaults, which can then be overriden by the e2e flags
// and the config flag.
type Mode string

const (
	// Quick runs a single E2E test and the systemd log tests.
	Quick Mode = "quick"

	// NonDisruptiveConformance runs all of the `Conformance` E2E tests which are not marked as disuprtive and the systemd log tests.
	NonDisruptiveConformance Mode = "non-disruptive-conformance"

	// CertifiedConformance runs all of the `Conformance` E2E tests and the systemd log tests.
	CertifiedConformance Mode = "certified-conformance"
)

// nonDisruptiveSkipList should generally just need to skip disruptive tests since upstream
// will disallow the other types of tests from being tagged as Conformance. However, in v1.16
// two disruptive tests were  not marked as such, meaning we needed to specify them here to ensure
// user workload safety. See https://github.com/kubernetes/kubernetes/issues/82663
// and https://github.com/kubernetes/kubernetes/issues/82787
const nonDisruptiveSkipList = `\[Disruptive\]|NoExecuteTaintManager`

var modeMap = map[string]Mode{
	string(NonDisruptiveConformance): NonDisruptiveConformance,
	string(Quick):                    Quick,
	string(CertifiedConformance):     CertifiedConformance,
}

// ModeConfig represents the sonobuoy configuration for a given mode.
type ModeConfig struct {
	// E2EConfig is the focus and skip vars for the conformance tests.
	E2EConfig E2EConfig
	// Selectors are the plugins selected by this mode.
	Selectors []plugin.Selection
}

// String needed for pflag.Value
func (m *Mode) String() string { return string(*m) }

// Type needed for pflag.Value
func (m *Mode) Type() string { return "Mode" }

// Set the name with a given string. Returns error on unknown mode.
func (m *Mode) Set(str string) error {
	// Allow other casing from user in command line (e.g. Quick & quick)
	lcase := strings.ToLower(str)
	mode, ok := modeMap[lcase]
	if !ok {
		return fmt.Errorf("unknown mode %s", str)
	}
	*m = mode
	return nil
}

// Get returns the ModeConfig associated with a mode name, or nil
// if there's no associated mode
func (m *Mode) Get() *ModeConfig {
	switch *m {
	case CertifiedConformance:
		return &ModeConfig{
			E2EConfig: E2EConfig{
				Focus:    `\[Conformance\]`,
				Parallel: "1",
			},
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case NonDisruptiveConformance:
		return &ModeConfig{
			E2EConfig: E2EConfig{
				Focus:    `\[Conformance\]`,
				Skip:     nonDisruptiveSkipList,
				Parallel: "1",
			},
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case Quick:
		return &ModeConfig{
			E2EConfig: E2EConfig{
				Focus:    "Pods should be submitted and removed",
				Parallel: "1",
			},
			Selectors: []plugin.Selection{
				{Name: "e2e"},
			},
		}
	default:
		return nil
	}
}

// GetModes gets a list of all available modes.
func GetModes() []string {
	keys := make([]string, len(modeMap))
	i := 0
	for k := range modeMap {
		keys[i] = k
		i++
	}
	return keys
}
