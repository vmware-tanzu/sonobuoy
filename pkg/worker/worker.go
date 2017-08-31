/*
Copyright 2017 Heptio Inc.

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

package worker

import (
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GatherResults is the consumer of a co-scheduled container that agrees on the following
// contract:
//
// 1. Output data will be placed into an agreed upon results directory.
// 2. The Job will wait for a done file
// 3. The done file contains a single string of the results to be sent to the master
func GatherResults(waitfile string, url string) error {
	var inputFileName []byte
	var err error

	// just loop looking for a file.
	done := false
	logrus.Infof("Waiting on: (%v)", waitfile)
	for !done {
		inputFileName, err = ioutil.ReadFile(waitfile) // For read access.
		if err != nil {
			// There is no need to log here, just wait for the results.
			logrus.Infof("Sleeping")
			time.Sleep(1 * time.Second)
		} else {
			done = true
		}
	}

	s := string(inputFileName)
	logrus.Infof("Detected done file, transmitting: (%v)", s)

	// Append a file extension, if there is one
	filenameParts := strings.SplitN(s, ".", 2)
	if len(filenameParts) == 2 {
		url += "." + filenameParts[1]
	}

	// transmit back the results file.
	return DoRequest(url, func() (io.Reader, error) {
		outfile, err := os.Open(s)
		return outfile, errors.WithStack(err)
	})
}
