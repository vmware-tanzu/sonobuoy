/*
Copyright the Sonobuoy contributors 2021

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

package types

import (
	"fmt"
	"strings"
)

// SecurityContextMode determines the security context object for the aggregator pod.
type SecurityContextMode string

const (
	SecurityContextModeNonRoot SecurityContextMode = "nonroot"
	SecurityContextModeNone    SecurityContextMode = "none"
)

var securityContextModeMap = map[string]SecurityContextMode{
	string(SecurityContextModeNonRoot): SecurityContextModeNonRoot,
	string(SecurityContextModeNone):    SecurityContextModeNone,
}

// String needed for pflag.Value.
func (r *SecurityContextMode) String() string { return string(*r) }

// Type needed for pflag.Value.
func (r *SecurityContextMode) Type() string { return "securityContextMode" }

func (r *SecurityContextMode) Set(str string) error {
	upcase := strings.ToLower(str)
	mode, ok := securityContextModeMap[upcase]
	if !ok {
		return fmt.Errorf("unknown SecurityContextMode mode %s", str)
	}
	*r = mode
	return nil
}
