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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const (
	doneFileBase      = "done"
	defaultResultsDir = "/tmp/sonobuoy/results"
)

var (
	doneFile   string
	resultsDir string
)

func init() {
	resultsDir = os.Getenv("SONOBUOY_RESULTS_DIR")
	if resultsDir == "" {
		resultsDir = defaultResultsDir
	}
	doneFile = filepath.Join(resultsDir, doneFileBase)
	fmt.Printf("Using results dir %v and donefile %v\n", resultsDir, doneFile)
}

func main() {
	rootCmd := &cobra.Command{Use: "testImage", Version: "0.0.1"}
	rootCmd.AddCommand(cmdSingleFile)
	rootCmd.AddCommand(cmdTarFile)
	rootCmd.Execute()
}
