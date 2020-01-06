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
)

func TestGivenAnyGenConfigFlags(t *testing.T) {
	getSampleFlagsWithChanged := func(s []string) *genFlags {
		sampleFlags := &genFlags{}
		GenFlagSet(sampleFlags, DetectRBACMode)
		for _, v := range s {
			sampleFlags.genflags.Set(v, "foo")
		}
		return sampleFlags
	}

	testCases := []struct {
		desc      string
		inFlags   *genFlags
		whitelist []string
		expect    bool
	}{
		{
			desc:      "Nothing changed return true",
			inFlags:   getSampleFlagsWithChanged(nil),
			whitelist: []string{},
			expect:    false,
		}, {
			desc:      "One changed flag return true",
			inFlags:   getSampleFlagsWithChanged([]string{"kubeconfig"}),
			whitelist: []string{},
			expect:    true,
		}, {
			desc:      "One changed flag return false if in whitelist",
			inFlags:   getSampleFlagsWithChanged([]string{"kubeconfig"}),
			whitelist: []string{"kubeconfig"},
			expect:    false,
		}, {
			desc:      "One changed flag return true if not in whitelist",
			inFlags:   getSampleFlagsWithChanged([]string{"e2e-focus"}),
			whitelist: []string{"flaga", "flagb", "flagc"},
			expect:    true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out := givenAnyGenConfigFlags(tc.inFlags, tc.whitelist)
			if out != tc.expect {
				t.Errorf("Expected %v but got %v", tc.expect, out)
			}
		})
	}
}
