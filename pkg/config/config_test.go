/*
Copyright 2017 Heptio Inc.

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

package config

import (
	"reflect"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg1 := NewWithDefaults()
	cfg2 := NewWithDefaults()

	if reflect.DeepEqual(&cfg2, &cfg1) {
		t.Fatalf("Defaults should not match UUIDs collided")
	}

	// set UUIDs to be the same
	cfg1.UUID = "0xDEADBEEF"
	cfg2.UUID = "0xDEADBEEF"

	if !reflect.DeepEqual(&cfg2, &cfg1) {
		t.Fatalf("Defaults should match but didn't")
	}
}

func TestFilters(t *testing.T) {
	cfg1 := NewWithDefaults()
	checkresources := cfg1.FilterResources(ClusterResources)

	for _, felement := range ClusterResources {
		if checkresources[felement] == false {
			t.Fatalf("Failed to properly filter")
		}
	}
}
