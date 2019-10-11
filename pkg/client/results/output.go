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

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// SaveProcessedResults saves the given item in the predefined location
// for the postprocessed results (the path base/plugins/plugin_name/sonobuoy_results)
func SaveProcessedResults(pluginName, baseDir string, item Item) error {
	resultsFile := filepath.Join(baseDir, PluginsDir, pluginName, PostProcessedResultsFile)
	if err := os.MkdirAll(filepath.Dir(resultsFile), 0755); err != nil {
		return errors.Wrap(err, "error creating plugin directory")
	}

	outfile, err := os.Create(resultsFile)
	if err != nil {
		return errors.Wrap(err, "error creating results file")
	}
	defer outfile.Close()

	enc := yaml.NewEncoder(outfile)
	defer enc.Close()
	err = enc.Encode(item)
	return errors.Wrap(err, "error writing to results file")
}
