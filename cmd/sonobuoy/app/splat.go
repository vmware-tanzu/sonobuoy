/*
Copyright 2021 VMware Inc.

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

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"

	"github.com/spf13/cobra"
)

func NewCmdSplat() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "splat AGGREGATOR_RESULTS_PATH",
		Short: "Reads all tarballs in the specified directory and prints the content to STDOUT (for internal use)",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runSplat(args[0]); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
		Hidden: true,
		Args:   cobra.ExactArgs(1),
	}
	return cmd
}

func runSplat(dirPath string) error {
	sonobuoyResults, err := filepath.Glob(filepath.Join(dirPath, "*.tar.gz"))
	if err != nil {
		return err
	}

	if err = loadResults(os.Stdout, sonobuoyResults); err != nil {
		return err
	}

	return nil
}

func loadResults(w *os.File, filenames []string) error {
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, filename := range filenames {
		if err := addFileToTarball(tarWriter, filename); err != nil {
			return err
		}
	}

	return nil
}

func addFileToTarball(tarWriter *tar.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	infoHeader, err := file.Stat()
	if err != nil {
		return err
	}

	// archive path of content as basename instead of full path
	// so un-archiving will result in the working directory, not AGGREGATOR_RESULTS_PATH
	header, err := tar.FileInfoHeader(infoHeader, infoHeader.Name())
	if err != nil {
		return err
	}

	if err = tarWriter.WriteHeader(header); err != nil {
		return err
	}

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if _, err := tarWriter.Write(fileContent); err != nil {
		return err
	}

	return nil
}
