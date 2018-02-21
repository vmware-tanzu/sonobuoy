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

import "github.com/spf13/cobra"

// AddNamespaceFlag initialises a namespace flag
func AddNamespaceFlag(str *string, cmd *cobra.Command) {
	// TODO(timothysc) This variable default needs saner image defaults from ops.f(n) or config
	cmd.PersistentFlags().StringVarP(
		str, "namespace", "n", "heptio-sonobuoy",
		"The namespace to run Sonobuoy in. Only one Sonobuoy run can exist per namespace simultaneously.",
	)
}
