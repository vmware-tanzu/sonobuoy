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

package main

import (
	"os"

	"github.com/heptio/sonobuoy/cmd/sonobuoy/app"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

// Main entry point of the program. Execute methods historically would log
// errors and exit manually via os.Exit in which case the error handling
// here never is invoked. We want to move towards commands that return errors
// and use this generic log/exit logic.
func main() {
	err := app.NewSonobuoyCommand().Execute()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
}
