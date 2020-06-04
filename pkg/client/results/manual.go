/*
Copyright the Sonobuoy contributors 2020

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
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func manualProcessFile(pluginDir, currentFile string) (Item, error) {
	relPath, err := filepath.Rel(pluginDir, currentFile)
	if err != nil {
		logrus.Errorf("Error making path %q relative to %q: %v", pluginDir, currentFile, err)
		relPath = currentFile
	}

	rootObj := Item{
		Name:   filepath.Base(currentFile),
		Status: StatusUnknown,
		Metadata: map[string]string{
			metadataFileKey: relPath,
			metadataTypeKey: metadataTypeFile,
		},
	}

	infile, err := os.Open(currentFile)
	if err != nil {
		rootObj.Metadata["error"] = err.Error()
		rootObj.Status = StatusUnknown

		return rootObj, errors.Wrapf(err, "opening file %v", currentFile)
	}
	defer infile.Close()

	resultObj, err := manualProcessReader(infile)
	if err != nil {
		return rootObj, errors.Wrap(err, "error processing manual results")
	}

	rootObj.Status = resultObj.Status
	rootObj.Items = resultObj.Items
	rootObj.Details = resultObj.Details

	return rootObj, nil
}

func manualProcessReader(r io.Reader) (Item, error) {
	resultItem := Item{}

	dec := yaml.NewDecoder(r)
	err := dec.Decode(&resultItem)
	if err != nil {
		return resultItem, errors.Wrap(err, "failed to parse yaml results object provided by plugin:")
	}

	return resultItem, nil
}
