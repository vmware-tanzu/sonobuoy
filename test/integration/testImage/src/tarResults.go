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
	"github.com/viniciuschiele/tarx"
)

var cmdTarFile = &cobra.Command{
	Use:   "tar-file",
	Short: "Returns a tarball with files as results",
	Args:  cobra.MinimumNArgs(1),
	RunE:  reportTarFile,
}

func reportTarFile(cmd *cobra.Command, args []string) error {
	outpath := os.TempDir()
	fmt.Printf("Using tmp dir %v for results. Will remove them after tarring the results\n", outpath)
	defer os.RemoveAll(outpath)

	for _, f := range args {
		targetFile := f
		resultsFile := filepath.Join(outpath, filepath.Base(targetFile))

		// Copy file to location Sonobuoy can get it.
		_, err := copyFile(targetFile, resultsFile)
		if err != nil {
			return errors.Wrapf(err, "failed to copy file %v to %v", targetFile, resultsFile)
		}
	}

	// Create tarball.
	tb := filepath.Join(resultsDir, "results.tar.gz")
	err := tarx.Compress(tb, outpath, &tarx.CompressOptions{Compression: tarx.Gzip})
	if err != nil {
		return errors.Wrap(err, "failed to create tarball")
	}

	// Report location to Sonobuoy.
	err = ioutil.WriteFile(doneFile, []byte(tb), os.FileMode(0666))
	return errors.Wrap(err, "failed to write to done file")
}
