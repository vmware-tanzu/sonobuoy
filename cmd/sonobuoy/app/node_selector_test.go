/*
Copyright the Sonobuoy contributors 2020

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
	"testing"

	"github.com/kylelemons/godebug/pretty"
)

func TestSetNodeSelector(t *testing.T) {
	testCases := []struct {
		desc      string
		expectErr string
		ns        NodeSelectors
		expect    NodeSelectors
		input     string
	}{
		{
			desc:      "Empty key and value should error",
			expectErr: "expected form key:value but got 1 parts when splitting by ':'",
		},
		{
			desc:      "Empty value should error",
			input:     "key:",
			expectErr: "expected form key:value with a non-empty value, but got value of length 0",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.ns.Set(tc.input)
			switch {
			case err != nil && len(tc.expectErr) == 0:
				t.Fatalf("Expected nil error but got %q", err)
			case err != nil && len(tc.expectErr) > 0:
				if fmt.Sprint(err) != tc.expectErr {
					t.Errorf("Expected error \n\t%q\nbut got\n\t%q", tc.expectErr, err)
				}
				return
			case err == nil && len(tc.expectErr) > 0:
				t.Fatalf("Expected error %q but got nil", tc.expectErr)
			default:
				// No error
			}

			if diff := pretty.Compare(tc.expect, tc.ns); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
