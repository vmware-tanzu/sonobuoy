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
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// RawProcessFile will return an Item object with the File value set to the path in question. If the
// file is unable to be stat'd then the status of the Item is StatusFailed (StatusPassed otherwise).
func RawProcessFile(pluginDir, currentFile string) (Item, error) {
	relPath, err := filepath.Rel(pluginDir, currentFile)
	if err != nil {
		logrus.Errorf("Error making path %q relative to %q: %v", pluginDir, currentFile, err)
		relPath = currentFile
	}

	i := Item{
		Name:     filepath.Base(currentFile),
		Status:   StatusPassed,
		Metadata: map[string]string{"file": relPath},
	}

	if _, err := os.Stat(currentFile); err != nil {
		i.Status = StatusFailed
		i.Metadata["error"] = err.Error()
	}

	return i, nil
}
