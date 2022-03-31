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

package app

import (
	"fmt"
	"sort"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	v1 "k8s.io/api/core/v1"
)

type ImagePullPolicy v1.PullPolicy

var pullPolicyMap = map[string]ImagePullPolicy{
	string(v1.PullAlways):       ImagePullPolicy(v1.PullAlways),
	string(v1.PullNever):        ImagePullPolicy(v1.PullNever),
	string(v1.PullIfNotPresent): ImagePullPolicy(v1.PullIfNotPresent),
}

func (i *ImagePullPolicy) String() string { return string(*i) }
func (i *ImagePullPolicy) Type() string   { return "ImagePullPolicy" }

func (i *ImagePullPolicy) Set(str string) error {
	// Allow lowercase pull policies in command line
	upcase := cases.Title(language.AmericanEnglish).String(str)
	policy, ok := pullPolicyMap[upcase]
	if !ok {
		return fmt.Errorf("unknown pull policy %q", str)
	}
	*i = policy
	return nil
}

func ValidPullPolicies() []string {
	valid := make([]string, len(pullPolicyMap))
	i := 0
	for key := range pullPolicyMap {
		valid[i] = key
		i++
	}
	sort.StringSlice(valid).Sort()
	return valid
}
