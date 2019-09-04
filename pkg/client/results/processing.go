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
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/daemonset"

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

	// PostProcessedResultsFile is the name of the file we create when doing
	// postprocessing on the plugin results.
	PostProcessedResultsFile = "sonobuoy_results.yaml"
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

// aggregateStatus defines the aggregation rules for status. Failures bubble
// up and otherwise the status is assumed to pass as long as there are >=1 result.
// If 0 items are aggregated, StatusUnknown is returned.
func aggregateStatus(items ...Item) string {
	// Avoid the situation where we get 0 results (because the plugin partially failed to run)
	// but we report it as passed.
	if len(items) == 0 {
		return StatusUnknown
	}

	unknownFound := false
	for i := range items {
		// Branches should just aggregate their leaves and return the result.
		if len(items[i].Items) > 0 {
			items[i].Status = aggregateStatus(items[i].Items...)
		}

		// Any failures immediately fail the parent.
		if items[i].Status == StatusFailed {
			return StatusFailed
		}

		if items[i].Status == StatusUnknown {
			unknownFound = true
		}
	}

	if unknownFound {
		return StatusUnknown
	}

	// Only pass if no failures found.
	return StatusPassed
}

// PostProcessPlugin will inspect the files in the given directory (representing
// the location of the results directory for a sonobuoy run, not the plugin specific
// results directory). Based on the type of plugin results, it will record what tests
// passed/failed (if junit) or record what files were produced (if raw) and return
// that information in an Item object.
func PostProcessPlugin(p plugin.Interface, dir string) (Item, error) {
	i := Item{}
	var err error

	switch p.GetResultFormat() {
	case ResultFormatE2E, ResultFormatJUnit:
		i, err = processPluginWithProcessor(p, dir, junitProcessFile, fileOrExtension(p.GetResultFiles(), ".xml"))
	case ResultFormatRaw:
		i, err = processPluginWithProcessor(p, dir, rawProcessFile, fileOrAny(p.GetResultFiles()))
	default:
		// Default to raw format so that consumers can still expect the aggregate file to exist and
		// can navigate the output of the plugin more easily.
		i, err = processPluginWithProcessor(p, dir, rawProcessFile, fileOrAny(p.GetResultFiles()))
	}

	i.Status = aggregateStatus(i.Items...)
	return i, err
}

func processPluginWithProcessor(p plugin.Interface, baseDir string, processor postProcessor, selector fileSelector) (Item, error) {
	pdir := path.Join(baseDir, PluginsDir, p.GetName())
	pResultsDir := path.Join(pdir, ResultsDir)

	_, isDS := p.(*daemonset.Plugin)
	results := Item{
		Name: p.GetName(),
	}

	if isDS {
		nodeDirs, err := ioutil.ReadDir(pResultsDir)
		if err != nil {
			return Item{}, err
		}

		for _, nodeDirInfo := range nodeDirs {
			if !nodeDirInfo.IsDir() {
				continue
			}
			nodeName := filepath.Base(nodeDirInfo.Name())
			nodeItem := Item{
				Name: nodeName,
			}
			items, err := processDir(p, pdir, filepath.Join(pResultsDir, nodeName), processor, selector)
			nodeItem.Items = items
			if err != nil {
				logrus.Warningf("Error processing results entries for node %v, plugin %v: %v", nodeDirInfo.Name(), p.GetName(), err)
			}
			results.Items = append(results.Items, nodeItem)
		}
	} else {
		items, err := processDir(p, pdir, pResultsDir, processor, selector)
		if err != nil {
			logrus.Warningf("Error processing results entries for plugin %v: %v", p.GetName(), err)
		}
		results.Items = items
	}

	results.Status = aggregateStatus(results.Items...)
	return results, nil
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
