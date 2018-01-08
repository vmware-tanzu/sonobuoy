package results

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heptio/sonobuoy/pkg/config"
)

const (
	hostsDir                  = "hosts/"
	namespacedResourcesDir    = "resources/ns/"
	nonNamespacedResourcesDir = "resources/cluster/"
	pluginsDir                = "plugins/"
	podLogs                   = "podlogs/"
	metadataDir               = "meta/"

	defaultServicesFile      = "Services.json"
	defaultNodesFile         = "Nodes.json"
	defaultServerVersionFile = "serverversion.json"
	defaultServerGroupsFile  = "servergroups.json"
)

// TODO(chuckha) Wrap errors.

const (
	// UnknownVersion lets the consumer know if this client can detect the archive version or not.
	UnknownVersion = "v?.?"
	versionEight   = "v0.8"
	versionNine    = "v0.9"
	versionTen     = "v0.10"
)

// Reader holds a reader and a version. It uses the version to know
// where to find certain files within the archive stream.
type Reader struct {
	io.Reader
	Version string
}

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
		return nil, err
	}
	version, err := DiscoverVersion(gzipReader)
	if err != nil {
		return nil, err
	}
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	err = gzipReader.Reset(r)
	if err != nil {
		return nil, err
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
		return "", err
	}
	var version string
	// Get rid of any of the extra version information that doesn't affect archive layout.
	// Example: v0.10.0-a2b3d4
	if strings.HasPrefix(conf.Version, versionEight) {
		version = versionEight
	} else if strings.HasPrefix(conf.Version, versionNine) {
		version = versionNine
	} else if strings.HasPrefix(conf.Version, versionTen) {
		version = versionTen
	} else {
		return "", errors.New("cannot discover Sonobuoy archive version")
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

// WalkFiles walks all of the files in the archive.
func (r *Reader) WalkFiles(walkfn filepath.WalkFunc) error {
	tr := tar.NewReader(r)
	var err error
	var header *tar.Header
	for {
		header, err = tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		info := &tarFileInfo{
			header.FileInfo(),
			tr,
		}
		err = walkfn(filepath.Clean(header.Name), info, err)
	}
	return nil

}

// Functions to be used within a walkfn

// ExtractBytes pulls out bytes into a buffer for any path matching file.
func ExtractBytes(file string, path string, info os.FileInfo, buf *bytes.Buffer) error {
	if file == path {
		reader, ok := info.Sys().(io.Reader)
		if !ok {
			return errors.New("info.Sys() is not a reader")
		}
		_, err := buf.ReadFrom(reader)
		if err != nil {
			return err
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
		// TODO(chuckha) there must be a better way
		if strings.HasSuffix(path, "xml") {
			decoder := xml.NewDecoder(reader)
			err := decoder.Decode(object)
			if err != nil {
				return err
			}
			return nil
		}

		// If it's not xml it's probably json
		decoder := json.NewDecoder(reader)
		err := decoder.Decode(object)

		if err != nil {
			return err
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
		return path == ConfigFile(versionTen) || path == ConfigFile(versionEight)
	}, path, info, conf)
}

// Functions for helping with backwards compatibility

func (a *Reader) Metadata() string {
	return metadataDir
}

func (a *Reader) ServerVersionFile() string {
	switch a.Version {
	case versionEight:
		return "serverversion/serverversion.json"
	default:
		return defaultServerVersionFile
	}
}

// NamespacedResources returns the path to
func (a *Reader) NamespacedResources() string {
	return namespacedResourcesDir
}

// NonNamespacedResources returns the path to the non-namespaced directory.
func (a *Reader) NonNamespacedResources() string {
	switch a.Version {
	case versionEight:
		return "resources/non-ns/"
	default:
		return nonNamespacedResourcesDir
	}
}

// NodesFile returns the path to the file listing nodes.
func (a *Reader) NodesFile() string {
	return filepath.Join(a.NonNamespacedResources(), defaultNodesFile)
}

// ServerGroupsFile returns the path to the groups the k8s api supported at the time of the run.
func (a *Reader) ServerGroupsFile() string {
	return defaultServerGroupsFile
}

// ConfigFile returns the path to the sonobuoy config file.
// This is not a method as it is used to determine the version of the archive.
func ConfigFile(version string) string {
	switch version {
	case versionEight:
		return "config.json"
	default:
		return "meta/config.json"
	}
}
