/*
Copyright Sonobuoy contributors 2021

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
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// the test has started running
	actionRun = "run"
	// the test has been paused
	actionPause = "pause"
	// the test has continued running
	actionCont = "cont"
	// the test passed
	actionPass = "pass"
	// the benchmark printed log output but did not fail
	actionBench = "bench"
	// the test or benchmark failed
	actionFail = "fail"
	// the test printed output
	actionOutput = "output"
	// the test was skipped or the package contained no tests
	actionSkip = "skip"
)

// testEvent is a single message emitted from test2json.
// Stolen from https://golang.org/cmd/test2json/ and github.com/cdr/gott
type testEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

func GojsonProcessFile(pluginDir, currentFile string) (Item, error) {
	relPath, err := filepath.Rel(pluginDir, currentFile)
	if err != nil {
		logrus.Errorf("Error making path %q relative to %q: %v", pluginDir, currentFile, err)
		relPath = currentFile
	}

	resultObj := Item{
		Name:   filepath.Base(currentFile),
		Status: StatusUnknown,
		Metadata: map[string]string{
			MetadataFileKey: relPath,
			MetadataTypeKey: MetadataTypeFile,
		},
	}

	infile, err := os.Open(currentFile)
	if err != nil {
		resultObj.Metadata["error"] = err.Error()
		resultObj.Status = StatusUnknown

		return resultObj, errors.Wrapf(err, "opening file %v", currentFile)
	}
	defer infile.Close()

	resultObj, err = gojsonProcessReader(
		infile,
		resultObj.Name,
		resultObj.Metadata,
	)
	if err != nil {
		return resultObj, errors.Wrap(err, "error processing gojson")
	}

	return resultObj, nil
}

func gojsonProcessReader(r io.Reader, name string, metadata map[string]string) (Item, error) {
	rootItem := Item{
		Name:     name,
		Status:   StatusPassed,
		Metadata: metadata,
	}

	if r == nil {
		rootItem.Status = StatusUnknown
		if rootItem.Metadata == nil {
			rootItem.Metadata = map[string]string{}
		}
		rootItem.Metadata["error"] = "no data source for gojson"
		return rootItem, errors.New("no data source for gojson")
	}

	decoder := json.NewDecoder(r)
	for {
		currentTest := testEvent{}
		if err := decoder.Decode(&currentTest); err == io.EOF {
			break
		} else if err != nil {
			rootItem.Status = StatusUnknown
			if rootItem.Metadata == nil {
				rootItem.Metadata = map[string]string{}
			}
			rootItem.Metadata["error"] = err.Error()
			return rootItem, errors.Wrap(err, "decoding gojson")
		}

		if i := gojsonEventToItem(currentTest); i != nil {
			rootItem.Items = append(rootItem.Items, *i)
		}
	}
	return rootItem, nil
}

func gojsonEventToItem(event testEvent) *Item {
	if contains([]string{
		actionCont, actionBench, actionOutput, actionPause, actionRun,
	}, event.Action) {
		logrus.WithField("action", event.Action).WithField("test", event.Test).Trace("Skipping gojson event")
		return nil
	}

	if event.Test == "" {
		logrus.WithField("action", event.Action).Trace("Skipping gojson event due to no test name attached")
		return nil
	}

	i := &Item{
		Name: event.Test,
	}
	switch event.Action {
	case actionPass:
		i.Status = StatusPassed
	case actionSkip:
		i.Status = StatusSkipped
	case actionFail:
		i.Status = StatusFailed
	default:
		i.Status = StatusUnknown
	}
	return i
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
