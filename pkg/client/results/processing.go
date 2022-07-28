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
	"sort"
	"strconv"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/daemonset"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ResultFormat constants are the supported values for the resultFormat field
// which enables post processing.
const (
	ResultFormatJUnit  = "junit"
	ResultFormatE2E    = "e2e"
	ResultFormatGoJSON = "gojson"
	ResultFormatRaw    = "raw"
	ResultFormatManual = "manual"
)

// postProcessor is a function which takes two strings: the plugin directory and the
// filepath in question, and parse it to create an Item.
type postProcessor func(string, string) (Item, error)

// fileSelector is a type of a function which, given a filename and the FileInfo will
// determine whether or not that file should be postprocessed. Allows matching a specific
// file only or all files with a given suffix (for instance).
type fileSelector func(string, os.FileInfo) bool

// hasCustomValues returns true if there is a leaf node with a custom status value.
func hasCustomValues(items ...Item) bool {
	for i := range items {
		if hasCustomValues(items[i].Items...) {
			return true
		}

		// Don't consider non-leaf nodes, since those will be overwritten when rolled up.
		if len(items[i].Items) > 0 {
			continue
		}

		switch items[i].Status {
		case StatusSkipped, StatusPassed, StatusFailed, StatusTimeout, StatusUnknown, StatusEmpty:
			continue
		default:

			return true
		}
	}

	return false
}

// AggregateStatus defines the aggregation rules for status according to the following rules:
// If only pass/fail/unknown values are found, we apply very basic rules:
//     - failure + * = failure
//     - unknown + [pass|unknown] = unknown
//     - empty list = unknown
// If we find other values (e.g. from manual results typically) then we just combine/count them:
//     - foo + bar = 'foo: 1, bar:1'
// useCustom is specified rather than looking for custom values because different branches of the
// result tree may have/not have those values. So instead, we should look down the tree initially to decide.
func AggregateStatus(useCustom bool, items ...Item) string {
	// Avoid the situation where we get 0 results (because the plugin partially failed to run)
	// but we report it as passed.
	if len(items) == 0 {
		return StatusUnknown
	}

	results := map[string]int{}
	var keys []string

	failedFound, unknownFound := false, false
	for i := range items {
		// Branches should just aggregate their leaves and return the result.
		if len(items[i].Items) > 0 {
			items[i].Status = AggregateStatus(useCustom, items[i].Items...)
		}

		// Empty status should be updated to unknown.
		if items[i].Status == "" {
			items[i].Status = StatusUnknown
		}

		s := items[i].Status
		switch {
		case IsFailureStatus(s):
			failedFound = true
		case s == StatusUnknown:
			unknownFound = true
		}

		existingCounts := parseCustomStatus(s)
		for k, v := range existingCounts {
			if _, exists := results[k]; !exists {
				keys = append(keys, k)
			}
			results[k] += v
		}
	}

	if useCustom {
		// Sort to keep ensure result ordering is consistent.
		sort.Strings(keys)

		var parts []string
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%v: %v", k, results[k]))
		}

		return strings.Join(parts, ", ")
	}

	if failedFound {
		return StatusFailed
	} else if unknownFound {
		return StatusUnknown
	}

	// Only pass if no failures found.
	return StatusPassed
}

func parseCustomStatus(s string) map[string]int {
	results := map[string]int{}
	resultsAndCounts := strings.Split(s, ", ")
	for _, v := range resultsAndCounts {
		parts := strings.Split(v, ": ")
		if len(parts) == 1 {
			results[parts[0]] += 1
		} else {
			count, err := strconv.Atoi(parts[1])
			if err != nil {
				logrus.Warningf("Error parsing custom status; expected a 'status: count: %v", err)
				continue
			}
			results[parts[0]] += count
		}
	}
	return results
}

// IsFailureStatus returns true if the status is any one of the failure modes (e.g.
// StatusFailed or StatusTimeout).
func IsFailureStatus(s string) bool {
	return s == StatusFailed || s == StatusTimeout
}

// PostProcessPlugin will inspect the files in the given directory (representing
// the location of the results directory for a sonobuoy run, not the plugin specific
// results directory). Based on the type of plugin results, it will record what tests
// passed/failed (if junit) or record what files were produced (if raw) and return
// that information in an Item object. All errors encountered are returned.
func PostProcessPlugin(p plugin.Interface, dir string) (Item, []error) {
	var i Item
	var errs []error

	switch p.GetResultFormat() {
	case ResultFormatE2E, ResultFormatJUnit:
		logrus.WithField("plugin", p.GetName()).Trace("Using junit post-processor")
		i, errs = processPluginWithProcessor(p, dir, JunitProcessFile, FileOrExtension(p.GetResultFiles(), ".xml"))
	case ResultFormatGoJSON:
		logrus.WithField("plugin", p.GetName()).Trace("Using gojson post-processor")
		i, errs = processPluginWithProcessor(p, dir, GojsonProcessFile, FileOrExtension(p.GetResultFiles(), ".json"))
	case ResultFormatRaw:
		logrus.WithField("plugin", p.GetName()).Trace("Using raw post-processor")
		i, errs = processPluginWithProcessor(p, dir, RawProcessFile, FileOrAny(p.GetResultFiles()))
	case ResultFormatManual:
		logrus.WithField("plugin", p.GetName()).Trace("Using manual post-processor")
		// Only process the specified plugin result files or a Sonobuoy results file.
		i, errs = processPluginWithProcessor(p, dir, manualProcessFile, FileOrExtension(p.GetResultFiles(), ".yaml", ".yml"))
	default:
		logrus.WithField("plugin", p.GetName()).Trace("Defaulting to raw post-processor")
		// Default to raw format so that consumers can still expect the aggregate file to exist and
		// can navigate the output of the plugin more easily.
		i, errs = processPluginWithProcessor(p, dir, RawProcessFile, FileOrAny(p.GetResultFiles()))
	}

	return i, errs
}

// processNodesWithProcessor is called to invoke ProcessDir on each node-specific directory contained
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
			Metadata: map[string]string{MetadataTypeKey: MetadataTypeNode},
		}
		items, err := ProcessDir(p, pdir, filepath.Join(dir, nodeName), processor, selector)
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
	var errs []error
	var items, errItems []Item
	var err error
	_, isDS := p.(*daemonset.Plugin)

	if isDS {
		items, err = processNodesWithProcessor(p, baseDir, pResultsDir, processor, selector)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pResultsDir))
		}
		errItems, err = processNodesWithProcessor(p, baseDir, pErrorsDir, errProcessor, errSelector())
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pErrorsDir))
		}
	} else {
		items, err = ProcessDir(p, pdir, pResultsDir, processor, selector)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pResultsDir))
		}

		errItems, err = ProcessDir(p, pdir, pErrorsDir, errProcessor, errSelector())
		if err != nil && !os.IsNotExist(err) {
			errs = append(errs, errors.Wrapf(err, "processing plugin %q, directory %q", p.GetName(), pErrorsDir))
		}
	}

	results := aggregateAllResultsAndErrors(p.GetName(), items, errItems)
	return results, errs
}

func aggregateAllResultsAndErrors(name string, items, errItems []Item) Item {
	results := Item{
		Name:     name,
		Metadata: map[string]string{MetadataTypeKey: MetadataTypeSummary},
	}

	results.Items = append(results.Items, items...)
	results.Items = append(results.Items, errItems...)
	results.Status = AggregateStatus(hasCustomValues(results.Items...), results.Items...)

	return results
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
		Details:  map[string]interface{}{},
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
		resultObj.Name = fmt.Sprint(resultObj.Details["error"])
	}

	if IsTimeoutErr(resultObj) {
		resultObj.Status = StatusTimeout
	}

	return resultObj, nil
}

// ProcessDir will walk the files in a given directory, using the fileSelector function to
// choose which files to process with the postProcessor. The plugin directory is also passed in
// (e.g. plugins/e2e) in order to make filepaths relative to that directory.
func ProcessDir(p plugin.Interface, pluginDir, dir string, processor postProcessor, shouldProcessFile fileSelector) ([]Item, error) {
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

// FileOrExtension returns a function which will return true for files
// which have the exact name of the file given or the given extension (if
// no file is given). If the filename given is empty, it will be ignored
// and the extension matching will be used. If "*" is passed as the extension
// all files will match.
func FileOrExtension(files []string, exts ...string) fileSelector {
	return func(fPath string, info os.FileInfo) bool {
		if info == nil || info.IsDir() {
			return false
		}

		if len(files) > 0 {
			return sliceContains(files, filepath.Base(fPath))
		}
		for _, ext := range exts {
			if ext == "*" || strings.HasSuffix(fPath, ext) {
				return true
			}
		}
		return false
	}
}

func FileOrAny(files []string) func(fPath string, info os.FileInfo) bool {
	return FileOrExtension(files, "*")
}

func errSelector() fileSelector {
	return FileOrExtension([]string{DefaultErrFile}, "")
}
