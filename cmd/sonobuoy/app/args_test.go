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

	ops "github.com/heptio/sonobuoy/pkg/client"
)

func TestGetE2EConfig(t *testing.T) {
	testCases := []struct {
		desc      string
		mode      ops.Mode
		flagArgs  []string
		expect    *ops.E2EConfig
		expectErr bool
	}{
		{
			desc:     "Default",
			mode:     "certified-conformance",
			flagArgs: []string{},
			expect: &ops.E2EConfig{
				Focus:    `\[Conformance\]`,
				Parallel: "1",
			},
		}, {
			desc:     "Flags settable",
			mode:     "certified-conformance",
			flagArgs: []string{"--e2e-focus=foo", "--e2e-skip=bar", "--e2e-parallel=2"},
			expect: &ops.E2EConfig{
				Focus:    `foo`,
				Skip:     `bar`,
				Parallel: "2",
			},
		}, {
			desc:      "Focus regexp validated settable",
			mode:      "certified-conformance",
			flagArgs:  []string{"--e2e-focus=*"},
			expectErr: true,
		}, {
			desc:      "Skip regexp validated settable",
			mode:      "certified-conformance",
			flagArgs:  []string{"--e2e-skip=*"},
			expectErr: true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ExitOnError)
			AddE2EConfigFlags(fs)
			fs.Parse(tC.flagArgs)
			cfg, err := GetE2EConfig(tC.mode, fs)
			switch {
			case err != nil && !tC.expectErr:
				t.Fatalf("Expected no error but got %v", err)
			case err == nil && tC.expectErr:
				t.Fatalf("Expected error but got none")
			}

			switch {
			case cfg == nil && tC.expect != nil:
				t.Fatalf("Expected value %+v but got nil", tC.expect)
			case cfg != nil && tC.expect == nil:
				t.Fatalf("Expected nil value but got %+v", cfg)
			case cfg == nil && tC.expect == nil:
				return
			case *tC.expect != *cfg:
				t.Errorf("Expected config %+v but got %+v", tC.expect, cfg)
			}
		})
	}
}
