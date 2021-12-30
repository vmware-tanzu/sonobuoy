/*
Copyright 2021 Sonobuoy Contributors

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
	"sort"

	"github.com/spf13/cobra"
)

type e2eModeOptions struct {
	name        string
	desc        string
	focus, skip string
	parallel    bool
}

const (
	// E2eModeQuick runs a single E2E test and the systemd log tests.
	E2eModeQuick string = "quick"

	// E2eModeNonDisruptiveConformance runs all of the `Conformance` E2E tests which are not marked as disuprtive and the systemd log tests.
	E2eModeNonDisruptiveConformance string = "non-disruptive-conformance"

	// E2eModeCertifiedConformance runs all of the `Conformance` E2E tests and the systemd log tests.
	E2eModeCertifiedConformance string = "certified-conformance"

	// nonDisruptiveSkipList should generally just need to skip disruptive tests since upstream
	// will disallow the other types of tests from being tagged as Conformance. However, in v1.16
	// two disruptive tests were  not marked as such, meaning we needed to specify them here to ensure
	// user workload safety. See https://github.com/kubernetes/kubernetes/issues/82663
	// and https://github.com/kubernetes/kubernetes/issues/82787
	nonDisruptiveSkipList = `\[Disruptive\]|NoExecuteTaintManager`
	conformanceFocus      = `\[Conformance\]`
	quickFocus            = "Pods should be submitted and removed"
)

// validModes is a map of the various valid modes. Name is duplicated as the key and in the e2eModeOptions itself.
var validModes = map[string]e2eModeOptions{
	E2eModeQuick: {
		name: E2eModeQuick, focus: quickFocus,
		desc: "Quick mode runs a single test to create and destroy a pod. Fastest way to check basic cluster operation.",
	},
	E2eModeNonDisruptiveConformance: {
		name: E2eModeNonDisruptiveConformance, focus: conformanceFocus, skip: nonDisruptiveSkipList,
		desc: "Non-destructive conformance mode runs all of the conformance tests except those that would disrupt other cluster operations (e.g. tests that may cause nodes to be restarted or impact cluster permissions).",
	},
	E2eModeCertifiedConformance: {
		name: E2eModeCertifiedConformance, focus: conformanceFocus,
		desc: "Certified conformance mode runs the entire conformance suite, even disruptive tests. This is typically run in a dev environment to earn the CNCF Certified Kubernetes status.",
	},
}

func validE2eModes() []string {
	keys := []string{}
	for key := range validModes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func NewCmdModes() *cobra.Command {
	var modesCmd = &cobra.Command{
		Use:   "modes",
		Short: "Display the various modes in which to run the e2e plugin",
		Run:   showModes(),
		Args:  cobra.ExactArgs(0),
	}
	return modesCmd
}

func showModes() func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		for i, key := range validE2eModes() {
			opt := validModes[key]
			if i != 0 {
				fmt.Println("")
			}
			fmt.Printf("Mode: %v\n", opt.name)
			fmt.Printf("Description: %v\n", opt.desc)
			fmt.Printf("E2E_FOCUS: %v\n", opt.focus)
			fmt.Printf("E2E_SKIP: %v\n", opt.skip)
			fmt.Printf("E2E_PARALLEL: %v\n", opt.parallel)
		}
	}
}
