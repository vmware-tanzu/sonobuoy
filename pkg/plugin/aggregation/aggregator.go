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

// Package aggregation is responsible for hosting an HTTP server which
// aggregates results from all of the nodes that are running sonobuoy agent. It
// is not responsible for dispatching the nodes (see pkg/dispatch), only
// expecting their results.
package aggregation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/tarball"
)

const (
	gzipMimeType       = "application/gzip"
	defaultRetryWindow = 120 * time.Second
)

// Aggregator is responsible for taking results from an HTTP server (configured
// elsewhere), saving them to the filesystem, and keeping track of what has
// been seen so far, so that we can return when all expected results are
// present and accounted for.
type Aggregator struct {
	// OutputDir is the directory to write the node results
	OutputDir string

	// Results stores a map of check-in results the server has seen
	Results map[string]*plugin.Result

	// ExpectedResults stores a map of results the server should expect
	ExpectedResults map[string]*plugin.ExpectedResult

	// LatestProgressUpdates is the map that saves the most recent progress update sent by
	// each plugin.
	LatestProgressUpdates map[string]*plugin.ProgressUpdate

	// FailedResults is a map to track which plugin results were received
	// but returned errors during processing. This enables us to retry results
	// that failed to process if the client tries, as opposed to rejecting
	// them as duplicates. Important if connection resets or network issues
	// are common.
	FailedResults map[string]time.Time

	// resultEvents is a channel that is written to when results are seen
	// by the server, so we can block until we're done.
	resultEvents chan *plugin.Result

	// resultsMutex prevents race conditions if two identical results
	// come in at the same time.
	resultsMutex sync.Mutex

	// progressMutex prevents race conditions between plugins updating their progresses.
	progressMutex sync.Mutex

	// retryWindow is the duration which the server will continue to block during
	// Wait() after a FailedResult has been reported, even if all expected results
	// are accounted for. This prevents racing the client retries that may occur.
	retryWindow time.Duration
}

// httpError is an internal error type which allows us to unify result processing
// across http and non-http flows.
type httpError struct {
	err  error
	code int
}

// HttpCode returns the http code associated with the error or an InternalServerError
// if none is set.
func (e *httpError) HttpCode() int {
	if e.code != 0 {
		return e.code
	}
	return http.StatusInternalServerError
}

// Error describes the error.
func (e *httpError) Error() string {
	return e.err.Error()
}

// keyer interface is for type swhich can generate a unique key for their type
// based on their data (e.g. what you'd use for a map key of lookups)
type keyer interface {
	Key() string
}

// NewAggregator constructs a new Aggregator object to write the given result
// set out to the given output directory.
func NewAggregator(outputDir string, expected []plugin.ExpectedResult) *Aggregator {
	aggr := &Aggregator{
		OutputDir:             outputDir,
		Results:               make(map[string]*plugin.Result, len(expected)),
		ExpectedResults:       make(map[string]*plugin.ExpectedResult, len(expected)),
		FailedResults:         make(map[string]time.Time, len(expected)),
		LatestProgressUpdates: make(map[string]*plugin.ProgressUpdate, len(expected)),
		resultEvents:          make(chan *plugin.Result, len(expected)),
		retryWindow:           defaultRetryWindow,
	}

	for i, expResult := range expected {
		aggr.ExpectedResults[expResult.ID()] = &expected[i]
	}

	return aggr
}

// Wait blocks until all expected results have come in.
func (a *Aggregator) Wait(stop chan bool) {
	for !a.isComplete() {
		select {
		case <-a.resultEvents:
		case <-stop:
			return
		}
	}

	// Give all clients a chance to retry failed requests.
	for _, failedTime := range a.FailedResults {
		remainingTime := retryWindowRemaining(failedTime, time.Now(), a.retryWindow)

		// A sleep for 0 or < 0 returns immediately.
		time.Sleep(remainingTime)
	}
}

// retryWindowRemaining wraps the awkward looking calculation to see the time beteween
// two events and subtract out a given duration. If the returned duration is 0 or negative
// it means that the time between the first and second events is equal or greater to the
// window's duration.
func retryWindowRemaining(first, second time.Time, window time.Duration) time.Duration {
	return first.Add(window).Sub(second)
}

// isComplete returns true if sure all expected results have checked in.
func (a *Aggregator) isComplete() bool {
	a.resultsMutex.Lock()
	defer a.resultsMutex.Unlock()

	for _, result := range a.ExpectedResults {
		if _, ok := a.Results[result.ID()]; !ok {
			return false
		}
	}

	return true
}

func (a *Aggregator) isExpected(obj keyer) bool {
	_, ok := a.ExpectedResults[obj.Key()]
	return ok
}

func (a *Aggregator) isResultDuplicate(result *plugin.Result) bool {
	_, ok := a.Results[result.Key()]
	return ok
}

// processResult is the centralized location for result processing. It is thread-safe
// and checks for whether or not the result should be excluded due to be either
// unexpected or a duplicate. Errors returned via this method will be of the type
// *httpError so that HTTP servers can respond appropriately to clients.
func (a *Aggregator) processResult(result *plugin.Result) error {
	a.resultsMutex.Lock()
	defer a.resultsMutex.Unlock()

	resultID := result.Key()

	// Make sure we were expecting this result
	if !a.isExpected(result) {
		return &httpError{
			err:  fmt.Errorf("result %v unexpected", resultID),
			code: http.StatusForbidden,
		}
	}

	// Don't allow duplicates unless it failed to process fully.
	isDup := a.isResultDuplicate(result)
	_, hadErrs := a.FailedResults[resultID]
	if isDup && !hadErrs {
		return &httpError{
			err:  fmt.Errorf("result %v already received", resultID),
			code: http.StatusConflict,
		}
	}

	// Send an event that we got this result even if we get an error, so
	// that Wait() doesn't hang forever on problems.
	defer func() {
		a.Results[result.Key()] = result
		a.resultEvents <- result
	}()

	if err := a.handleResult(result); err != nil {
		// Drop a breadcrumb so that we reconsider new results from this result.
		a.FailedResults[result.Key()] = time.Now()
		return &httpError{
			err:  fmt.Errorf("error handling result %v: %v", resultID, err),
			code: http.StatusInternalServerError,
		}
	}

	// Upon success, we no longer want to keep processing duplicate results.
	delete(a.FailedResults, result.Key())

	return nil
}

// processProgressUpdate is the main aggregator logic for handling the progress updates from plugins.
// We first
func (a *Aggregator) processProgressUpdate(progress plugin.ProgressUpdate) error {
	a.resultsMutex.Lock()
	expected := a.isExpected(progress)
	a.resultsMutex.Unlock()

	// Make sure we were expecting this result
	if !expected {
		return &httpError{
			err:  fmt.Errorf("progress update for %v unexpected", progress.Key()),
			code: http.StatusForbidden,
		}
	}

	// Set this as the most recent progress update. Another routine is responsible for updating the
	// aggregator status annotation with that information.
	a.progressMutex.Lock()
	a.LatestProgressUpdates[progress.Key()] = &progress
	a.progressMutex.Unlock()

	return nil
}

// HandleHTTPResult is called every time the HTTP server gets a well-formed
// request with results. This method is responsible for returning with things
// like a 409 conflict if a node has checked in twice (or a 403 forbidden if a
// node isn't expected), as well as actually calling handleResult to write the
// results to OutputDir.
func (a *Aggregator) HandleHTTPResult(result *plugin.Result, w http.ResponseWriter) {
	err := a.processResult(result)
	if err != nil {
		switch t := err.(type) {
		case *httpError:
			logrus.Errorf("Result processing error (%v): %v", t.HttpCode(), t.Error())
			http.Error(
				w,
				t.Error(),
				t.HttpCode(),
			)
		default:
			logrus.Errorf("Result processing error (%v): %v", http.StatusInternalServerError, t.Error())
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
		}
	}
}

// HandleHTTPProgressUpdate wraps the aggregators processProgressUpdate method in such a way as to respond
// with appropriate logging and HTTP codes.
func (a *Aggregator) HandleHTTPProgressUpdate(progress plugin.ProgressUpdate, w http.ResponseWriter) {
	err := a.processProgressUpdate(progress)
	if err != nil {
		switch t := err.(type) {
		case *httpError:
			logrus.Errorf("Progress update error (%v): %v", t.HttpCode(), t.Error())
			http.Error(
				w,
				t.Error(),
				t.HttpCode(),
			)
		default:
			logrus.Errorf("Progress update error (%v): %v", http.StatusInternalServerError, t.Error())
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
		}
	}
}

// IngestResults takes a channel of results and handles them as they come in.
// Since most plugins submit over HTTP, this method is currently only used to
// consume an error stream from each plugin's Monitor() function.
//
// If we support plugins that are just simple commands that the Sonobuoy aggregator
// runs, those plugins can submit results through the same channel.
func (a *Aggregator) IngestResults(ctx context.Context, resultsCh <-chan *plugin.Result) {
	for {
		var result *plugin.Result
		var more bool

		select {
		case <-ctx.Done():
			return
		case result, more = <-resultsCh:
		}

		if result == nil && !more {
			return
		}

		logInternalResult(result)
		err := a.processResult(result)
		if err != nil {
			switch t := err.(type) {
			case *httpError:
				logrus.Errorf("Result processing error (%v): %v", t.HttpCode(), t.Error())
			default:
				logrus.Errorf("Result processing error (%v): %v", http.StatusInternalServerError, t.Error())
			}
		}
	}
}

func logInternalResult(r *plugin.Result) {
	log := logrus.WithField("plugin_name", r.ResultType)
	log = log.WithField("node", r.NodeName)
	log.Info("received internal aggregator result")
}

// handleResult takes a given plugin Result and writes it out to the
// filesystem, signaling to the resultEvents channel when complete.
func (a *Aggregator) handleResult(result *plugin.Result) error {
	if result.MimeType == gzipMimeType {
		return a.handleArchiveResult(result)
	}

	// Create the output directory for the result. Will be of the
	// form .../plugins/:results_type/results/[:node|global]/filename.json
	resultsDir := filepath.Join(a.OutputDir, result.Path())
	if result.Filename == "" {
		result.Filename = defaultFilename
	}
	resultFile := filepath.Join(resultsDir, result.Filename)

	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		errors.Wrapf(err, "couldn't create directory %q", resultsDir)
		return err
	}

	outFile, err := os.Create(resultFile)
	if err != nil {
		return errors.Wrapf(err, "couldn't create results file %q", resultFile)
	}
	defer outFile.Close()

	if _, err = io.Copy(outFile, result.Body); err != nil {
		err = errors.Wrapf(err, "could not write body to file %q", resultFile)
		return err
	}

	return nil
}

func (a *Aggregator) handleArchiveResult(result *plugin.Result) error {
	resultsDir := filepath.Join(a.OutputDir, result.Path())

	return errors.Wrapf(
		tarball.DecodeTarball(result.Body, resultsDir),
		"couldn't decode result %v", result.Path(),
	)
}
