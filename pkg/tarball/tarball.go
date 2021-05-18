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

package tarball

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// DecodeTarball takes a reader and a base directory, and extracts a gzipped tarball rooted on
// the given directory. If there is an error, the imput may only be partially consumed.
// At the moment, the tarball decoder only supports directories, regular files and symlinks.
func DecodeTarball(reader io.Reader, baseDir string) error {
	gzStream, err := gzip.NewReader(reader)
	if err != nil {
		return errors.Wrap(err, "couldn't uncompress reader")
	}
	defer gzStream.Close()

	tarchive := tar.NewReader(gzStream)
	for {
		header, err := tarchive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "couldn't opening tarball from gzip")
		}
		name := path.Clean(header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(baseDir, name), os.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (mkdir)")
			}
		case tar.TypeReg, tar.TypeRegA:
			filePath := filepath.Join(baseDir, name)
			// Directory should come first, but some tarballes are malformed
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (mkdir)")
			}
			file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrap(err, "error decoding tarball for result (open)")
			}
			if _, err := io.CopyN(file, tarchive, header.Size); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (copy)")
			}
		case tar.TypeSymlink:
			if !noTraversal(name, baseDir) {
				return errors.Wrapf(err, "unsafe symlink detected in name: %v", name)
			}
			filePath := filepath.Join(baseDir, name)
			// Directory should come first, but some tarballes are malformed
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return errors.Wrapf(err, "error decoding tarball for result (mkdir)")
			}
			if err := os.Symlink(
				filepath.Join(baseDir, path.Clean(header.Linkname)),
				filepath.Join(baseDir, name),
			); err != nil {
				return errors.Wrap(err, "error decoding tarball for result (ln)")
			}
		default:
		}
	}

	return nil
}

// noTraversal is an ultra-slimmed down function to
// avoid traversals outside the destination folder.
// moby and other repos have much more complex versions
// while this aims at removing the ability to symlink to ".."
// entirely so that it is dead simple. I don't think for our needs
// we really require symlinks much anyways.
// Resolves go/unsafe-unzip-symlink codeQL check.
func noTraversal(candidate, target string) bool {
	if filepath.IsAbs(candidate) {
		return false
	}
	return !strings.Contains(candidate, "..")
}

// DirToTarball tars up an entire directory and outputs the tarball to the specified output path.
// If useGzip is true, it also gzips the resulting tarball.
func DirToTarball(dir, outpath string, useGzip bool) error {
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
