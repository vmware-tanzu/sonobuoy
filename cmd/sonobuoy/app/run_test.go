/*
Copyright the Sonobuoy contributors 2019

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
	"testing"

	"github.com/spf13/pflag"
)

func TestGivenAnyGenConfigFlags(t *testing.T) {
	getSampleFlagsWithChanged := func(s []string) *pflag.FlagSet {
		sampleFlags := &genFlags{}
		fs := GenFlagSet(sampleFlags, DetectRBACMode)
		for _, v := range s {
			fs.Set(v, "foo")
		}
		return fs
	}

	testCases := []struct {
		desc         string
		inFlags      *pflag.FlagSet
		allowedFlags []string
		expect       bool
	}{
		{
			desc:         "Nothing changed return true",
			inFlags:      getSampleFlagsWithChanged(nil),
			allowedFlags: []string{},
			expect:       false,
		}, {
			desc:         "One changed flag return true",
			inFlags:      getSampleFlagsWithChanged([]string{"kubeconfig"}),
			allowedFlags: []string{},
			expect:       true,
		}, {
			desc:         "One changed flag return false if in allowed list",
			inFlags:      getSampleFlagsWithChanged([]string{"kubeconfig"}),
			allowedFlags: []string{"kubeconfig"},
			expect:       false,
		}, {
			desc:         "One changed flag return true if not in allowed list",
			inFlags:      getSampleFlagsWithChanged([]string{"e2e-focus"}),
			allowedFlags: []string{"flaga", "flagb", "flagc"},
			expect:       true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out := givenAnyGenConfigFlags(tc.inFlags, tc.allowedFlags)
			if out != tc.expect {
				t.Errorf("Expected %v but got %v", tc.expect, out)
			}
		})
	}
}
