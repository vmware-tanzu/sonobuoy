/*
Copyright the Sonobuoy project contributors 2019

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

import "testing"

// TestCommands exists to ensure that some test goes through
// the code paths of generating all the commands. This helps with
// getting a better sense of code coverage and provides a place to
// add more tests to later.
func TestCommands(t *testing.T) {
	c := NewSonobuoyCommand()
	if c == nil {
		t.Fatal("Expected non-nil command; got nil")
	}
}
