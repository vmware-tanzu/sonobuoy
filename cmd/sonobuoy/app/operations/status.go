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

package operations

import (
	"errors"
)

// Status determines the status of the sonobuoy run in order to assist the user.
func GetStatus( /*opts?*/ ) error {
	// Do the following:
	// 1. Check to see if the heptio namespace exists
	// 2. If it exists check to see if it's blocking(finished) or not
	// 3. If it's still running, post state as running w/breadcrumb to call logs to inspect the details.
	// 4. TODO: Wedge detection w/timeouts () tests can wedge in places, and sometimes the forwarder can also wedge.
	return errors.New("not implemented")
}
