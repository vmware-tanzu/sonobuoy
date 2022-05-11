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

package client

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	pluginaggregation "github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type DeletionError struct {
	filename string
	err      error
}

func (d *DeletionError) Error() string {
	return errors.Wrapf(d.err, "failed to delete file %q", d.filename).Error()
}

// RetrieveResults copies results from a sonobuoy run into a Reader in tar format.
// It also returns a channel of errors, where any errors encountered when writing results
// will be sent, and an error in the case where the config validation fails.
func (c *SonobuoyClient) RetrieveResults(cfg *RetrieveConfig) (io.Reader, <-chan error, error) {
	if cfg == nil {
		return nil, nil, errors.New("nil RetrieveConfig provided")
	}

	if err := cfg.Validate(); err != nil {
		return nil, nil, errors.Wrap(err, "config validation failed")
	}

	ec := make(chan error, 1)
	client, err := c.Client()
	if err != nil {
		return nil, ec, nil
	}

	// Determine sonobuoy pod name
	podName, err := pluginaggregation.GetAggregatorPodName(client, cfg.Namespace)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get the name of the aggregator pod to fetch results from")
	}

	logrus.Tracef("Running command %v on the aggregator", tarCmd(cfg.Path))
	restClient := client.CoreV1().RESTClient()
	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(cfg.Namespace).
		SubResource("exec").
		Param("container", config.AggregatorContainerName)
	req.VersionedParams(&corev1.PodExecOptions{
		Container: config.AggregatorContainerName,
		Command:   tarCmd(cfg.Path),
		Stdin:     false,
		Stdout:    true,
		Stderr:    false,
	}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(c.RestConfig, "POST", req.URL())
	if err != nil {
		ec <- err
		return nil, ec, nil
	}
	reader, writer := io.Pipe()
	go func(writer *io.PipeWriter, ec chan error) {
		defer writer.Close()
		defer close(ec)
		err = executor.Stream(remotecommand.StreamOptions{
			Stdout: writer,
			Tty:    false,
		})
		if err != nil {
			ec <- err
		}
	}(writer, ec)

	return reader, ec, nil
}

/** Everything below this marker originally was copy/pasta'd from k8s/k8s. The modification are:
  exporting UntarAll, returning the list of files created, and the fix for undrained readers. Also
  added the filename override as opposed to using a file prefix and reconciled the filename/destDir
  values.  **/

// UntarAll expects a reader that contains tar'd data. It will untar the contents of the reader and write
// the output into destDir. If filename specified, then files will be named "filename, filename-01, filename-02". It returns a list of all the
// files it created.
func UntarAll(reader io.Reader, destDir, filename string) (filenames []string, returnErr error) {
	// Reconcile variables in case filename is also a path.
	destDir = filepath.Join(destDir, filepath.Dir(filename))
	if filename != "" {
		// filepath.Base("") returns "." which is a directory, not a file base name as expected.
		filename = filepath.Base(filename)
	}

	entrySeq, filenameCount := -1, -1
	filenames = []string{}
	// Adding compression per `splat` subcommand implementation
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		returnErr = err
		return
	}
	defer gzReader.Close()
	tarReader := tar.NewReader(gzReader)

	// Ensure the reader gets drained. Tar on some platforms doesn't
	// seem to consume the end-of-archive marker (long string of 0s).
	// If we fail to read them all then any pipes we are connected to
	// may hang.
	defer func() {
		err := drainReader(reader)
		switch {
		case err == nil:
			return
		case returnErr == nil && err != nil:
			returnErr = err
		case returnErr != nil && err != nil:
			logrus.Warnf("Error while draining reader (%v) but there was a previous error that may be the root cause: %v", err, returnErr)
			return
		}
	}()

	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return filenames, err
			}
			break
		}

		// Avoid naively joining paths and allowing escape of intended directory.
		// Recommended by github CodeQL; go/zipslip
		if strings.Contains(header.Name, "..") {
			continue
		}

		entrySeq++
		mode := header.FileInfo().Mode()
		outFileName := filepath.Join(destDir, header.Name)
		baseName := filepath.Dir(outFileName)

		if err := os.MkdirAll(baseName, 0755); err != nil {
			return filenames, err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(outFileName, 0755); err != nil {
				return filenames, err
			}
			continue
		}

		// handle coping remote file into local directory
		if entrySeq == 0 && !header.FileInfo().IsDir() {
			exists, err := dirExists(outFileName)
			if err != nil {
				return filenames, err
			}
			if exists {
				outFileName = filepath.Join(outFileName, filepath.Base(header.Name))
			}
		}

		if mode&os.ModeSymlink != 0 {
			err := os.Symlink(header.Linkname, outFileName)
			if err != nil {
				return filenames, err
			}
		} else {
			filenameCount++
			// For regular file, respect requested name.
			if len(filename) > 0 {
				outFileName = filepath.Join(destDir, getFilename(filename, filenameCount))
			}
			outFile, err := os.Create(outFileName)
			if err != nil {
				return filenames, err
			}
			filenames = append(filenames, outFileName)
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return filenames, err
			}
			if err := outFile.Close(); err != nil {
				return filenames, err
			}
		}
	}

	if entrySeq == -1 {
		//if no file was copied
		return filenames, errors.New("no valid entries in result")
	}
	return filenames, nil
}

// getFilename, given foo.ext should return foo.ext, foo-01.ext, foo-02.ext...
func getFilename(f string, i int) string {
	if i <= 0 {
		return f
	}
	ext := filepath.Ext(f)
	base := filepath.Base(f)
	baseWithoutExt := strings.TrimRight(base, ext)

	return fmt.Sprintf("%v-%02d%v", baseWithoutExt, i, ext)
}

// UntarFile untars the file, filename, into the given destination directory with the given prefix. If delete
// is true, it deletes the original file after extraction.
func UntarFile(filename string, destination string, delete bool) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %q for extraction after downloading it", filename)
	}
	defer f.Close()

	_, err = UntarAll(f, destination, "")
	if err != nil {
		return errors.Wrapf(err, "failed to extract file %q after downloading it", filename)
	}
	if err := os.Remove(filename); err != nil {
		// Returning a specific type of error so that consumers can ignore if desired. The file did
		// get untar'd successfully and so this error has less significance.
		return &DeletionError{filename: filename, err: err}
	}
	return nil
}

// dirExists checks if a path exists and is a directory.
func dirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func tarCmd(path string) []string {
	return []string{
		"/sonobuoy",
		"splat",
		path,
	}
}

func drainReader(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "failed to drain the tar reader")
	}

	for i := range b {
		if b[i] != 0 {
			return fmt.Errorf("non-zero data %v (byte %v) read after tar EOF", string(b[i]), b[i])
		}
	}
	return nil
}
