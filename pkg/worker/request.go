/*
Copyright 2018 Heptio Inc.

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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sethgrid/pester"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

// DoRequest calls the given callback which returns an io.Reader, and submits
// the results, with error handling, and falls back on uploading JSON with the
// error message if the callback fails. (This way, problems gathering data
// don't result in the server waiting forever for results that will never
// come.)
func DoRequest(url string, client *http.Client, callback func() (io.Reader, string, string, error)) error {
	// Create a client with retry logic. Manually log each error as they occur rather than
	// saving them internal to the client. This helps debug issues which may have relied on retries.
	pesterClient := pester.NewExtendedClient(client)
	pesterClient.KeepLog = false
	pesterClient.LogHook = func(e pester.ErrEntry) {
		var errDetailed error
		if e.Err != nil {
			errDetailed = errors.Wrapf(e.Err,
				"error entry for attempt: %v, verb: %v, time: %v, URL: %v",
				e.Attempt, e.Verb, e.Time, e.URL,
			)
		} else {
			errDetailed = fmt.Errorf(
				"error entry for attempt: %v, verb: %v, time: %v, URL: %v",
				e.Attempt, e.Verb, e.Time, e.URL,
			)
		}
		errlog.LogError(errDetailed)
	}

	input, filename, mimeType, err := callback()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error gathering host data"))

		// If the callback couldn't get the data, we should send the reason why to
		// the server.
		errobj := map[string]string{
			"error": err.Error(),
		}
		errbody, err := json.Marshal(errobj)
		if err != nil {
			return errors.WithStack(err)
		}
		req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(errbody))
		req.Header.Add("content-type", mimeType)
		if err != nil {
			return errors.WithStack(err)
		}

		// And if we can't even do that, log it.
		resp, err := pesterClient.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
		if err != nil {
			errlog.LogError(errors.Wrapf(err, "could not send error message to aggregator URL (%s)", url))
		}

		return errors.WithStack(err)
	}

	req, err := http.NewRequest(http.MethodPut, url, input)
	if err != nil {
		return errors.Wrapf(err, "error constructing aggregator request to %v", url)
	}
	req.Header.Add("content-type", mimeType)
	if len(filename) > 0 {
		req.Header.Add("content-disposition", fmt.Sprintf("attachment;filename=%v", filename))
	} else {
		req.Header.Add("content-disposition", "attachment")
	}

	resp, err := pesterClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "error encountered dialing aggregator at %v", url)
	}
	switch resp.StatusCode {
	case http.StatusConflict:
		// 409 indicates we've already submitted results. Can occur in some daemonset cases and
		// isn't useful to error here.
		errlog.LogError(errors.Errorf("got a %v response when dialing aggregator to %v. Logging and proceeding as normal.", resp.StatusCode, url))
	case http.StatusOK:
		return nil
	default:
		return errors.Errorf("got a %v response when dialing aggregator to %v. Logging and proceeding as normal.", resp.StatusCode, url)
	}
	return nil
}
