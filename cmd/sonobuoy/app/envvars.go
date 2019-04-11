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
	"fmt"
	"strings"
)

type EnvVars map[string]string

func (i *EnvVars) String() string { return fmt.Sprint(map[string]string(*i)) }
func (i *EnvVars) Type() string   { return "envvar" }

func (i *EnvVars) Set(str string) error {
	eq := strings.Index(str, "=")
	if eq < 0 {
		delete(map[string]string(*i), str)
		return nil
	}
	keyval := strings.SplitN(str, "=", 2)
	map[string]string(*i)[keyval[0]] = keyval[1]
	return nil
}

// Map just casts the EnvVars to its underlying map type.
func (i *EnvVars) Map() map[string]string {
	return map[string]string(*i)
}
