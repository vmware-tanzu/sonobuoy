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
	"errors"
	"fmt"
	"strings"
)

// NodeSelectors are k-v pairs that will be added to the aggregator.
type NodeSelectors map[string]string

func (i *NodeSelectors) String() string { return fmt.Sprint((map[string]string)(*i)) }
func (i *NodeSelectors) Type() string   { return "nodeSelectors" }

// Set parses the value from the CLI and places it into the internal map. Expected
// form is key:value. If no `:` is found or it is the last character
// ("x" or "x:") then the value will be set as the empty string.
func (i *NodeSelectors) Set(str string) error {
	parts := strings.SplitN(str, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected form key:value but got %v parts when splitting by ':'", len(parts))
	}

	if len(parts[1]) == 0 {
		return errors.New("expected form key:value with a non-empty value, but got value of length 0")
	}

	k := parts[0]
	mapI := (map[string]string)(*i)

	if mapI == nil {
		mapI = map[string]string{}
	}

	mapI[k] = parts[1]

	*i = mapI
	return nil
}
