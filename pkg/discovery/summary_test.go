/*
Copyright the Sonobuoy contributors 2022

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

package discovery

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "update the .golden files")

func TestReadHealthSummary(t *testing.T) {
	//Root directory where all the file sin the tarball are located
	//Simulates the directory where all the files resulting from discovery end up before being compressed into the tarball
	//For the test, this directory is the location where the test files are located
	tarballRootDir := "testdata/healthsummary"

	goldenFilePath := filepath.Join(tarballRootDir, "summary_test.golden")

	got, err := ReadHealthSummary(tarballRootDir)
	if err != nil {
		t.Fatalf("\n\nReadHealthSummary('%s') failed with error %s\n", tarballRootDir, err)
	}

	gotJson, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("\n\nThe value returned from ReadHealthSummary('%s') fails to be marshalled to json: %s\n", tarballRootDir, err)
	}

	if *update {
		ioutil.WriteFile(goldenFilePath, gotJson, 0666)
	} else {
		expectedJson, err := ioutil.ReadFile(goldenFilePath)
		if err != nil {
			t.Fatalf("\n\nFailed to read golden file from '%s': %s\n", goldenFilePath, err)
		}
		if !bytes.Equal(gotJson, expectedJson) {
			t.Fatalf("\n\nExpected %s,\n     got %s\n", expectedJson, gotJson)
		}
	}
}
