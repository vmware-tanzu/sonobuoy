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

	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/utils/mode"
)

// RunConfig are the input options for running
// TODO: We should expose FOCUS and other options with sane defaults
type RunConfig struct {
	Mode mode.Name
}

func Run(cfg RunConfig) error {
	// Do the following:
	// 1. Validate mode of input
	// 2. Pull the api-settings in order to generate the correct .yaml
	// 3. Generate the yaml, follow kubeadm as a pattern here, and we may want to
	//    subst the params that plumb all the way through.
	// 4. Submit the .yaml - Here is where it will get weird b/c you you may need to submit the resources separately b
	return errors.New("not implemented")
}
