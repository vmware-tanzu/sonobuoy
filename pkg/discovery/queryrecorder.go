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

package discovery

import (
	"encoding/json"
	"os"
	"path"
	"time"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
)

type QueryRecorder struct {
	queries []*queryData
}

func NewQueryRecorder() *QueryRecorder {
	return &QueryRecorder{
		queries: make([]*queryData, 0),
	}
}

// queryData captures the results of the run for post-processing
type queryData struct {
	QueryObj    string `json:"queryobj,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ElapsedTime string `json:"time,omitempty"`
	Error       error  `json:"error,omitempty"`
}

func (q *QueryRecorder) RecordQuery(name string, namespace string, duration time.Duration, recerr error) {
	if recerr != nil {
		errlog.LogError(errors.Wrapf(recerr, "error querying %v", name))
	}
	summary := &queryData{
		QueryObj:    name,
		Namespace:   namespace,
		ElapsedTime: duration.String(),
		Error:       recerr,
	}

	q.queries = append(q.queries, summary)
}

func (q *QueryRecorder) DumpQueryData(filepath string) error {
	// Ensure the leading path is created
	err := os.MkdirAll(path.Dir(filepath), 0755)
	if err != nil {
		return err
	}

	// Format the query data as JSON
	data, err := json.Marshal(q.queries)
	if err != nil {
		return err
	}

	// Create the file
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the data
	_, err = f.Write(data)
	return err
}
