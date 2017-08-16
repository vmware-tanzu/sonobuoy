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
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/sethgrid/pester"
)

// DoRequest calls the given callback which returns an io.Reader, and submits
// the results, with error handling, and falls back on uploading JSON with the
// error message if the callback fails. (This way, problems gathering data
// don't result in the server waiting forever for results that will never
// come.)
func DoRequest(url string, callback func() (io.Reader, error)) error {
	input, err := callback()
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
		if err != nil {
			return errors.WithStack(err)
		}

		// And if we can't even do that, log it.
		resp, err := pester.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			errlog.LogError(errors.Wrapf(err, "could not send error message to master URL (%v)", url))
		}

		return errors.WithStack(err)
	}

	req, err := http.NewRequest(http.MethodPut, url, input)
	if err != nil {
		return errors.Wrapf(err, "error constructing master request to %v", url)
	}

	resp, err := pester.Do(req)
	if err != nil {
		return errors.Wrapf(err, "error dialing master at %v", url)
	}
	if resp.StatusCode != http.StatusOK {
		// TODO: retry logic for something like a 429 or otherwise
		return errors.Errorf("got a %v response when dialing master to %v", resp.StatusCode, url)
	}

	return nil
}
