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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdSingleFile = &cobra.Command{
	Use:   "single-file",
	Short: "Returns a single file as results.",
	Long:  "Writes all the given files to the results directory but only writes the done file (if applicable) for the last one.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  reportSingleFile,
}

func reportSingleFile(cmd *cobra.Command, args []string) error {
	resultsFile := filepath.Join(resultsDir, filepath.Base(args[0]))
	for _, targetFile := range args {
		f := filepath.Join(resultsDir, filepath.Base(targetFile))

		// Copy file to location Sonobuoy can get it.
		_, err := copyFile(targetFile, f)
		if err != nil {
			return errors.Wrapf(err, "failed to copy file %v to %v", targetFile, resultsFile)
		}
	}

	// Report location to Sonobuoy.
	if cmd.Flags().Lookup("no-done").Value.String() == "true" {
		fmt.Println("no-done is set, exiting without writing done file")
		return nil
	}
	err := ioutil.WriteFile(doneFile, []byte(resultsFile), os.FileMode(0666))
	return errors.Wrap(err, "failed to write to done file")
}
