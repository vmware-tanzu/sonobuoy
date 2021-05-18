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

package results

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/vmware-tanzu/sonobuoy/pkg/config"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

const (
	// PluginsDir defines where in the archive directories for plugin results are.
	PluginsDir = "plugins/"

	// ResultsDir defines where in the archive the plugin results are.
	// Example: plugins/<name>/results
	ResultsDir = "results/"

	// ErrorsDir defines where in the archive the errors running the plugin get reported.
	// These are the Sonobuoy reported errors, e.g. failure to start a plugin, timeout, etc.
	// This is not the appropriate directory for things like test failures.
	// Example: plugins/<name>/errors
	ErrorsDir = "errors/"

	// DefaultErrFile is the file name used when Sonobuoy is reporting an error running a plugin.
	// Is written into the ErrorsDir directory.
	DefaultErrFile = "error.json"

	namespacedResourcesDir    = "resources/ns/"
	nonNamespacedResourcesDir = "resources/cluster/"
	metadataDir               = "meta/"
	defaultNodesFile          = "Nodes.json"
	defaultServerVersionFile  = "serverversion.json"
	defaultServerGroupsFile   = "servergroups.json"

	// InfoFile contains data not that isn't strictly in another location
	// but still relevent to post-processing or understanding the run in some way.
	InfoFile = "info.json"
)

// Versions corresponding to Kubernetes minor version values. We used to
// roughly version our results tarballs in sync with minor version patches
// and so checking the server version for one of these prefixes would be
// sufficient to inform the parser where certain files would be.
const (
	// UnknownVersion lets the consumer know if this client can detect the archive version or not.
	UnknownVersion = "v?.?"
	VersionEight   = "v0.8"
	VersionNine    = "v0.9"
	VersionTen     = "v0.10"
	VersionFifteen = "v0.15"
)

var (
	// v15 is the first version we started used a typed version. Allows more clean comparisons
	// between versions.
	v15 = goversion.Must(goversion.NewVersion("v0.15.0"))

	// errStopWalk is a special cased error when using reader.Walk which will stop
	// processing but will not be bubbled up. Used to prevent reading until EOF when you
	// want to stop mid-reader.
	errStopWalk = errors.New("stop")
)

// Reader holds a reader and a version. It uses the version to know where to
// find files within the archive.
type Reader struct {
	io.Reader
	Version string
}

// NewReaderWithVersion creates a results.Reader that interprets a results
// archive of the version passed in.
// Useful if the reader can be read only once and if the version of the data to
// read is known.
func NewReaderWithVersion(reader io.Reader, version string) *Reader {
	return &Reader{
		Reader:  reader,
		Version: version,
	}
}

// NewReaderFromBytes is a helper constructor that will discover the version of the archive
// and return a new Reader with the correct version already populated.
func NewReaderFromBytes(data []byte) (*Reader, error) {
	r := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new gzip reader")
	}
	version, err := DiscoverVersion(gzipReader)
	if err != nil {
		return nil, errors.Wrap(err, "error discovering version")
	}
	if _, err = r.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Wrap(err, "error seeking to start")
	}
	if err = gzipReader.Reset(r); err != nil {
		return nil, errors.Wrap(err, "error reseting gzip reader")
	}
	return &Reader{
		Reader:  gzipReader,
		Version: version,
	}, nil
}

// DiscoverVersion takes a Sonobuoy archive stream and extracts just the
// version of the archive.
func DiscoverVersion(reader io.Reader) (string, error) {
	r := &Reader{
		Reader: reader,
	}

	conf := &config.Config{}

	err := r.WalkFiles(func(path string, info os.FileInfo, err error) error {
		return ExtractConfig(path, info, conf)
	})
	if err != nil {
		return "", errors.Wrap(err, "error extracting config")
	}

	parsedVersion, err := goversion.NewVersion(conf.Version)
	if err != nil {
		return "", errors.Wrap(err, "parsing version")
	}
	segments := parsedVersion.Segments()
	if len(segments) < 2 {
		return "", fmt.Errorf("version %q only has %d segments, need at least 2", conf.Version, len(segments))
	}

	// Get rid of any of the extra version information that doesn't affect archive layout.
	// Example: v0.10.0-a2b3d4
	var version string
	switch {
	case strings.HasPrefix(conf.Version, VersionEight):
		version = VersionEight
	case strings.HasPrefix(conf.Version, VersionNine):
		version = VersionNine
	case strings.HasPrefix(conf.Version, VersionTen):
		version = VersionTen
	case parsedVersion.LessThan(v15):
		version = VersionTen
	default:
		version = VersionFifteen
	}
	return version, nil
}

// tarFileInfo implements os.FileInfo and extends the Sys() method to
// return a reader to a file in a tar archive.
type tarFileInfo struct {
	os.FileInfo
	io.Reader
}

// Sys is going to be an io.Reader to a file in a tar archive.
// This is how data is extracted from the archive.
func (t *tarFileInfo) Sys() interface{} {
	return t.Reader
}

// WalkFiles walks all of the files in the archive. Processing stops at the
// first error. The error is returned except in the special case of errStopWalk
// which will stop processing but nil will be returned.
func (r *Reader) WalkFiles(walkfn filepath.WalkFunc) error {
	tr := tar.NewReader(r)
	var err error
	var header *tar.Header
	for {
		if err != nil {
			break
		}
		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error getting next file in archive")
		}
		info := &tarFileInfo{
			header.FileInfo(),
			tr,
		}
		err = walkfn(path.Clean(header.Name), info, err)
	}

	if err == errStopWalk || err == io.EOF {
		return nil
	}
	return err
}

// Functions to be used within a walkfn.

// ExtractBytes pulls out bytes into a buffer for any path matching file.
func ExtractBytes(file string, path string, info os.FileInfo, buf *bytes.Buffer) error {
	if file == path {
		reader, ok := info.Sys().(io.Reader)
		if !ok {
			return errors.New("info.Sys() is not a reader")
		}
		_, err := buf.ReadFrom(reader)
		if err != nil {
			return errors.Wrap(err, "could not read from buffer")
		}
	}
	return nil
}

// ExtractIntoStruct takes a predicate function and some file information
// and decodes the contents of the file that matches the predicate into the
// interface passed in (generally a pointer to a struct/slice).
func ExtractIntoStruct(predicate func(string) bool, path string, info os.FileInfo, object interface{}) error {
	if predicate(path) {
		reader, ok := info.Sys().(io.Reader)
		if !ok {
			return errors.New("info.Sys() is not a reader")
		}
		// TODO(chuckha) Perhaps find a more robust way to handle different data formats.
		if strings.HasSuffix(path, "xml") {
			decoder := xml.NewDecoder(reader)
			if err := decoder.Decode(object); err != nil {
				return errors.Wrap(err, "error decoding xml into object")
			}
			return nil
		}

		// If it's not xml it's probably json
		decoder := json.NewDecoder(reader)
		if err := decoder.Decode(object); err != nil {
			return errors.Wrap(err, "error decoding json into object")
		}
	}
	return nil
}

// ExtractFileIntoStruct is a helper for a common use case of extracting
// the contents of one file into the object.
func ExtractFileIntoStruct(file, path string, info os.FileInfo, object interface{}) error {
	return ExtractIntoStruct(func(p string) bool {
		return file == p
	}, path, info, object)
}

// ExtractConfig populates the config object regardless of version.
func ExtractConfig(path string, info os.FileInfo, conf *config.Config) error {
	return ExtractIntoStruct(func(file string) bool {
		return path == ConfigFile(VersionTen) || path == ConfigFile(VersionEight)
	}, path, info, conf)
}

// Functions for helping with backwards compatibility

// Metadata is the location of the metadata directory in the results archive.
func (r *Reader) Metadata() string {
	return metadataDir
}

// ServerVersionFile is the location of the file that contains the Kubernetes
// version Sonobuoy ran on.
func (r *Reader) ServerVersionFile() string {
	switch r.Version {
	case VersionEight:
		return path.Join("serverversion", "serverversion.json")
	default:
		return defaultServerVersionFile
	}
}

// NamespacedResources returns the path to the directory that contains
// information about namespaced Kubernetes resources.
func (r *Reader) NamespacedResources() string {
	return namespacedResourcesDir
}

// NonNamespacedResources returns the path to the non-namespaced directory.
func (r *Reader) NonNamespacedResources() string {
	switch r.Version {
	case VersionEight:
		return path.Join("resources", "non-ns")
	default:
		return nonNamespacedResourcesDir
	}
}

// NodesFile returns the path to the file that lists the nodes of the Kubernetes
// cluster.
func (r *Reader) NodesFile() string {
	return path.Join(r.NonNamespacedResources(), defaultNodesFile)
}

// ServerGroupsFile returns the path to the groups the Kubernetes API supported at the time of the run.
func (r *Reader) ServerGroupsFile() string {
	return defaultServerGroupsFile
}

// ConfigFile returns the path to the sonobuoy config file.
// This is not a method as it is used to determine the version of the archive.
func ConfigFile(version string) string {
	switch version {
	case VersionEight:
		return "config.json"
	default:
		return path.Join("meta", "config.json")
	}
}

// RunInfoFile returns the path to the Sonobuoy RunInfo file which is extra metadata about the run.
// This was added in v0.16.1. The function will return the same string even for earlier
// versions where that file does not exist.
func (r *Reader) RunInfoFile() string {
	return path.Join(metadataDir, InfoFile)
}

// PluginResultsItem returns the results file from the given plugin if found, error otherwise.
func (r *Reader) PluginResultsItem(plugin string) (*Item, error) {
	resultObj := &Item{}

	reader, err := r.PluginResultsReader(plugin)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(reader)
	err = decoder.Decode(resultObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode yaml results for plugin %v", plugin)
	}

	return resultObj, nil
}

// PluginResultsReader returns the results file from the given plugin if found, error otherwise.
func (r *Reader) PluginResultsReader(plugin string) (io.Reader, error) {
	resultsPath := path.Join(PluginsDir, plugin, PostProcessedResultsFile)
	return r.FileReader(resultsPath)
}

// FileReader returns a reader for a file in the archive.
func (r *Reader) FileReader(filename string) (io.Reader, error) {
	var returnReader io.Reader

	found := false
	err := r.WalkFiles(
		func(path string, info os.FileInfo, err error) error {
			if err != nil || found {
				return err
			}

			if path == filename {
				found = true
				reader, ok := info.Sys().(io.Reader)
				if !ok {
					return errors.New("info.Sys() is not a reader")
				}
				returnReader = reader
				return errStopWalk
			}

			return nil
		})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk archive for file %v", filename)
	}
	if !found {
		return nil, fmt.Errorf("failed to find file %q in archive", filename)
	}

	return returnReader, nil
}
