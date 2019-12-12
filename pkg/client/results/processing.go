/*
Copyright the Sonobuoy contributors 2019

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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/daemonset"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// StatusFailed is the key we base junit pass/failure off of and save into
	// our canonical results format.
	StatusFailed = "failed"

	// StatusPassed is the key we base junit pass/failure off of and save into
	// our canonical results format.
	StatusPassed = "passed"

	// StatusSkipped is the key we base junit pass/failure off of and save into
	// our canonical results format.
	StatusSkipped = "skipped"

	// StatusUnknown is the key we fallback to in our canonical results format
	// if another can not be determined.
	StatusUnknown = "unknown"

	// StatusTimeout is the key used when the plugin does not report results within the
	// timeout period. It will be treated as a failure (e.g. its parent will be marked
	// as a failure).
	StatusTimeout = "timeout"

	// PostProcessedResultsFile is the name of the file we create when doing
	// postprocessing on the plugin results.
	PostProcessedResultsFile = "sonobuoy_results.yaml"

	// metadataFileKey is the key used in an Item's metadata field when the Item is
	// representing the a file summary (and its leaf nodes are individual tests or suites).
	metadataFileKey = "file"

	// metadataTypeKey is the key used in an Item's metadata field when describing what type
	// of entry in the tree it is. Currently we just tag summaries, files, and nodes.
	metadataTypeKey = "type"

	metadataTypeNode    = "node"
	metadataTypeFile    = "file"
	metadataTypeSummary = "summary"
)

// ResultFormat constants are the supported values for the resultFormat field
// which enables post processing.
const (
	ResultFormatJUnit = "junit"
	ResultFormatE2E   = "e2e"
	ResultFormatRaw   = "raw"
)

// postProcessor is a function which takes two strings: the plugin directory and the
// filepath in question, and parse it to create an Item.
type postProcessor func(string, string) (Item, error)

// fileSelector is a type of a function which, given a filename and the FileInfo will
// determine whether or not that file should be postprocessed. Allows matching a specific
// file only or all files with a given suffix (for instance).
type fileSelector func(string, os.FileInfo) bool

// Item is the central format for plugin results. Various plugin
// types can be transformed into this simple format and set at a standard
// location in our results tarball for simplified processing by any consumer.
type Item struct {
	Name     string            `json:"name" yaml:"name"`
	Status   string            `json:"status" yaml:"status"`
	Metadata map[string]string `json:"meta,omitempty" yaml:"meta,omitempty"`
	Details  map[string]string `json:"details,omitempty" yaml:"details,omitempty"`
	Items    []Item            `json:"items,omitempty" yaml:"items,omitempty"`
}

// Empty returns true if the Item is empty.
func (i Item) Empty() bool {
	if i.Name == "" && i.Status == "" && len(i.Items) == 0 && len(i.Metadata) == 0 {
		return true
	}
	return false
}

// GetSubTreeByName traverses the tree and returns a reference to the
// subtree whose root has the given name.
func (i *Item) GetSubTreeByName(root string) *Item {
	if i == nil {
		return nil
	}

	if root == "" || i.Name == root {
		return i
	}

	if len(i.Items) > 0 {
		for _, v := range i.Items {
			subItem := (&v).GetSubTreeByName(root)
			if subItem != nil {
				return subItem
			}
		}
	}

	return nil
}

// aggregateStatus defines the aggregation rules for status. Failures bubble
// up and otherwise the status is assumed to pass as long as there are >=1 result.
// If 0 items are aggregated, StatusUnknown is returned.
func aggregateStatus(items ...Item) string {
	// Avoid the situation where we get 0 results (because the plugin partially failed to run)
	// but we report it as passed.
	if len(items) == 0 {
		return StatusUnknown
	}

	failedFound, unknownFound := false, false
	for i := range items {
		// Branches should just aggregate their leaves and return the result.
		if len(items[i].Items) > 0 {
			items[i].Status = aggregateStatus(items[i].Items...)
		}

		// Empty status should be updated to unknown.
		if items[i].Status == "" {
			items[i].Status = StatusUnknown
		}

		switch {
		case isFailureStatus(items[i].Status):
			failedFound = true
		case items[i].Status == StatusUnknown:
			unknownFound = true
		default:
		}
	}

	// Only return once all processing is completed; otherwise other leaves don't get resolved.
	if failedFound {
		return StatusFailed
	} else if unknownFound {
		return StatusUnknown
	}

	// Only pass if no failures found.
	return StatusPassed
}

// isFailureStatus returns true if the status is any one of the failure modes (e.g.
// StatusFailed or StatusTimeout).
func isFailureStatus(s string) bool {
	return s == StatusFailed || s == StatusTimeout
}

// PostProcessPlugin will inspect the files in the given directory (representing
// the location of the results directory for a sonobuoy run, not the plugin specific
// results directory). Based on the type of plugin results, it will record what tests
// passed/failed (if junit) or record what files were produced (if raw) and return
// that information in an Item object. All errors encountered are returned.
func PostProcessPlugin(p plugin.Interface, dir string) (Item, []error) {
	i := Item{}
	var errs []error

	switch p.GetResultFormat() {
	case ResultFormatE2E, ResultFormatJUnit:
		i, errs = processPluginWithProcessor(p, dir, junitProcessFile, fileOrExtension(p.GetResultFiles(), ".xml"))
	case ResultFormatRaw:
		i, errs = processPluginWithProcessor(p, dir, rawProcessFile, fileOrAny(p.GetResultFiles()))
	default:
		// Default to raw format so that consumers can still expect the aggregate file to exist and
		// can navigate the output of the plugin more easily.
		i, errs = processPluginWithProcessor(p, dir, rawProcessFile, fileOrAny(p.GetResultFiles()))
	}

	i.Status = aggregateStatus(i.Items...)
	return i, errs
}

// processNodesWithProcessor is called to invoke processDir on each node-specific directory contained
// underneath the given dir. The directory is assumed to be either the results directory or errors directory
// which should have the nodes as subdirectories. It returns an item for each node processed and an error
// only if it couldn't open the original directory. Any errors while processing a specific node are logged
// but not returned.
func processNodesWithProcessor(p plugin.Interface, baseDir, dir string, processor postProcessor, selector fileSelector) ([]Item, error) {
	pdir := path.Join(baseDir, PluginsDir, p.GetName())

	nodeDirs, err := ioutil.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return []Item{}, err
	}

	results := []Item{}

	for _, nodeDirInfo := range nodeDirs {
		if !nodeDirInfo.IsDir() {
			continue
		}
		nodeName := filepath.Base(nodeDirInfo.Name())
		nodeItem := Item{
			Name:     nodeName,
			Metadata: map[string]string{metadataTypeKey: metadataTypeNode},
		}
		items, err := processDir(p, pdir, filepath.Join(dir, nodeName), processor, selector)
		nodeItem.Items = items
		if err != nil {
			logrus.Warningf("Error processing results entries for node %v, plugin %v: %v", nodeDirInfo.Name(), p.GetName(), err)
		}
		results = append(results, nodeItem)
	}

	return results, nil
}

// processPluginWithProcessor will apply the processor to the chosen files. It will also process the <plugin>/errors
// directory for errors. One item will be returned with the results already aggregated. All errors encountered will be
// returned.
func processPluginWithProcessor(p plugin.Interface, baseDir string, processor postProcessor, selector fileSelector) (Item, []error) {
	pdir := path.Join(baseDir, PluginsDir, p.GetName())
	pResultsDir := path.Join(pdir, ResultsDir)
	pErrorsDir := path.Join(pdir, ErrorsDir)
	errs := []error{}

	_, isDS := p.(*daemonset.Plugin)
	results := Item{
		Name:     p.GetName(),
		Metadata: map[string]string{metadataTypeKey: metadataTypeSummary},
	}

	if isDS {
		items, err := processNodesWithProcessor(p, baseDir, pResultsDir, processor, selector)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pResultsDir))
		}
		errItems, err := processNodesWithProcessor(p, baseDir, pErrorsDir, errProcessor, errSelector())
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pErrorsDir))
		}

		results.Items = append(results.Items, items...)
		results.Items = append(results.Items, errItems...)
	} else {
		items, err := processDir(p, pdir, pResultsDir, processor, selector)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pResultsDir))
		}
		results.Items = items

		items, err = processDir(p, pdir, pErrorsDir, errProcessor, errSelector())
		if err != nil && !os.IsNotExist(err) {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pErrorsDir))
		}
		results.Items = append(results.Items, items...)
	}

	results.Status = aggregateStatus(results.Items...)
	return results, errs
}

// errProcessor takes two strings: the plugin directory and the filepath in question, and parse it to create an Item.
// Intended to be used when parsing the errors directory which holds Sonobuoy reported errors for the plugin.
func errProcessor(pluginDir string, currentFile string) (Item, error) {
	relPath, err := filepath.Rel(pluginDir, currentFile)
	if err != nil {
		logrus.Errorf("Error making path %q relative to %q: %v", pluginDir, currentFile, err)
		relPath = currentFile
	}

	resultObj := Item{
		Name:     filepath.Base(currentFile),
		Status:   StatusFailed,
		Metadata: map[string]string{"file": relPath},
		Details:  map[string]string{},
	}

	infile, err := os.Open(currentFile)
	if err != nil {
		resultObj.Metadata["error"] = err.Error()
		resultObj.Status = StatusUnknown

		return resultObj, errors.Wrapf(err, "opening file %v", currentFile)
	}
	defer infile.Close()

	dec := json.NewDecoder(infile)
	result := map[string]interface{}{}
	if err := dec.Decode(&result); err != nil {
		return resultObj, errors.Wrapf(err, "decoding file %v", currentFile)
	}

	// Just copy the data from the saved error file.
	for k, v := range result {
		resultObj.Details[k] = fmt.Sprint(v)
	}

	// Surface the error to be the name of the "test" to make the error mode more visible to end users.
	// Seeing `error.json` wouldn't be helpful.
	if resultObj.Details["error"] != "" {
		resultObj.Name = resultObj.Details["error"]
	}

	if isTimeoutErr(resultObj) {
		resultObj.Status = StatusTimeout
	}

	return resultObj, nil
}

// isTimeoutErr is the snippet of logic that determines whether or not a given Item represents
// a timeout error (i.e. Sonobuoy timed out waiting for results).
func isTimeoutErr(i Item) bool {
	return strings.Contains(i.Details["error"], "timeout")
}

// processDir will walk the files in a given directory, using the fileSelector function to
// choose which files to process with the postProcessor. The plugin directory is also passed in
// (e.g. plugins/e2e) in order to make filepaths relative to that directory.
func processDir(p plugin.Interface, pluginDir, dir string, processor postProcessor, shouldProcessFile fileSelector) ([]Item, error) {
	results := []Item{}

	err := filepath.Walk(dir, func(curPath string, info os.FileInfo, err error) error {
		if shouldProcessFile(curPath, info) {
			newItem, err := processor(pluginDir, curPath)
			if err != nil {
				logrus.Errorf("Error processing file %v: %v", curPath, err)
			}
			results = append(results, newItem)
		}
		return nil
	})
	return results, err
}

func sliceContains(set []string, val string) bool {
	for _, v := range set {
		if v == val {
			return true
		}
	}
	return false
}

// fileOrExtension returns a function which will return true for files
// which have the exact name of the file given or the given extension (if
// no file is given). If the filename given is empty, it will be ignored
// and the extension matching will be used. If "*" is passed as the extension
// all files will match.
func fileOrExtension(files []string, ext string) fileSelector {
	return func(fPath string, info os.FileInfo) bool {
		if info == nil || info.IsDir() {
			return false
		}

		if len(files) > 0 {
			return sliceContains(files, filepath.Base(fPath))
		}
		return ext == "*" || strings.HasSuffix(fPath, ext)
	}
}

func fileOrAny(files []string) func(fPath string, info os.FileInfo) bool {
	return fileOrExtension(files, "*")
}

func errSelector() fileSelector {
	return fileOrExtension([]string{DefaultErrFile}, "")
}
