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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdTarFile = &cobra.Command{
	Use:   "tar-file",
	Short: "Returns a tarball with files as results",
	Args:  cobra.MinimumNArgs(1),
	RunE:  reportTarFile,
}

func reportTarFile(cmd *cobra.Command, args []string) error {
	tmpDir := os.TempDir()
	outPath, err := ioutil.TempDir(tmpDir, "sonobuoy-integration")
	if err != nil {
		return errors.Wrapf(err, "failed to create outPath dir %v", outPath)
	}
	fmt.Printf("Using tmp dir %v for results. Will remove them after tarring the results\n", outPath)
	defer os.RemoveAll(outPath)

	for _, f := range args {
		targetFile := f
		resultsFile := filepath.Join(outPath, filepath.Base(targetFile))
		// Copy file to location Sonobuoy can get it.
		_, err := copyFile(targetFile, resultsFile)
		if err != nil {
			return errors.Wrapf(err, "failed to copy file %v to %v", targetFile, resultsFile)
		}
	}

	// Create tarball.
	tb := filepath.Join(resultsDir, "results.tar.gz")
	err = tarDir(outPath, tb, true)
	if err != nil {
		return errors.Wrap(err, "failed to create tarball")
	}

	// Report location to Sonobuoy.
	err = ioutil.WriteFile(doneFile, []byte(tb), os.FileMode(0666))
	return errors.Wrap(err, "failed to write to done file")
}

// tarDir tars up an entire directory and outputs the tarball to the specified output path.
// If useGzip is true, it also gzips the resulting tarball.
// NOTE: Copied from sonobuoy itself; duplicated since this is a separate binary/project
// so it messed with the build when introducing this method the first time.
// TODO(jschnake): After initial commit in Sonobuoy; just import/reference that method. Just
// couldn't introduce to sonobuoy and reference it here at the same time.
func tarDir(dir, outpath string, useGzip bool) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(dir); err != nil {
		return errors.Wrapf(err, "tar unable to stat directory %v", dir)
	}

	outfile, err := os.Create(outpath)
	if err != nil {
		return errors.Wrapf(err, "creating tarball %v", outpath)
	}
	defer outfile.Close()

	var tw *tar.Writer
	if useGzip {
		gzw := gzip.NewWriter(outfile)
		defer gzw.Close()
		tw = tar.NewWriter(gzw)
	} else {
		tw = tar.NewWriter(outfile)
	}
	defer tw.Close()

	return filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		// Return on any error.
		if err != nil {
			return err
		}

		// Don't include the archive or dir itself.
		if filepath.Join(dir, fi.Name()) == outpath ||
			filepath.Clean(file) == filepath.Clean(dir) {
			return nil
		}

		// Dirs and files; fix invalid handle issue?
		if !fi.Mode().IsRegular() && !fi.Mode().IsDir() {
			return nil
		}

		// Create a new dir/file header.
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return errors.Wrapf(err, "creating file info header %v", fi.Name())
		}

		// Update the name to correctly reflect the desired destination when untaring.
		// 1. Remove directory prefix and leading /
		// 2: Store with Unix path seperators
		// 3: Clean
		header.Name = strings.TrimPrefix(path.Clean(filepath.ToSlash(strings.Replace(file, dir, "", -1))), "/")
		if err := tw.WriteHeader(header); err != nil {
			return errors.Wrapf(err, "writing header for tarball %v", header.Name)
		}

		// Only copy contents of regular files
		if !fi.Mode().IsRegular() {
			return nil
		}

		// Open files, copy into tarfile, and close.
		f, err := os.Open(file)
		if err != nil {
			return errors.Wrapf(err, "opening file %v for writing into tarball", file)
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return errors.Wrapf(err, "creating file %v contents into tarball", file)
	})
}
