/*
Copyright the Sonobuoy contributors 2022

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

package k8s

import (
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// MergeEnv will combine the values from two env var sets with priority being
// given to values in the first set in case of collision. Afterwards, any env
// var with a name in the removal set will be removed.
func MergeEnv(e1, e2 []corev1.EnvVar, removeKeys map[string]struct{}) []corev1.EnvVar {
	envSet := map[string]corev1.EnvVar{}
	returnEnv := []corev1.EnvVar{}
	for _, e := range e1 {
		envSet[e.Name] = e
	}
	for _, e := range e2 {
		if _, seen := envSet[e.Name]; !seen {
			envSet[e.Name] = e
		}
	}
	for name, v := range envSet {
		if _, remove := removeKeys[name]; !remove {
			returnEnv = append(returnEnv, v)
		}
	}

	sort.Slice(returnEnv, func(i, j int) bool {
		return strings.ToLower(returnEnv[i].Name) < strings.ToLower(returnEnv[j].Name)
	})

	return returnEnv
}
