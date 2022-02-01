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

package plugin

import (
	"context"
	"crypto/tls"
	"io"
	"path"
	"time"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// GlobalResult is used in place of a node name when the results apply
	// to the entire cluster as opposed to a single node (e.g. when running
	// a job and not a daemonset).
	GlobalResult = "global"
)

// Interface represents what all plugins must implement to be run and have
// results be aggregated.
type Interface interface {
	// Run runs a plugin, declaring all resources it needs, and then
	// returns.  It does not block and wait until the plugin has finished.
	Run(kubeClient kubernetes.Interface, hostname string, cert *tls.Certificate, ownerPod *v1.Pod, progressPort, resultDir string) error

	// Cleanup cleans up all resources created by the plugin
	Cleanup(kubeClient kubernetes.Interface)

	// Monitor continually checks for problems in the resources created by a
	// plugin (either because it won't schedule, or the image won't
	// download, too many failed executions, etc) and sends the errors as
	// Result objects through the provided channel. It should return once the context
	// is cancelled.
	Monitor(ctx context.Context, kubeClient kubernetes.Interface, availableNodes []v1.Node, resultsCh chan<- *Result)

	// ExpectedResults is an array of Result objects that a plugin should
	// expect to submit.
	ExpectedResults(nodes []v1.Node) []ExpectedResult

	// GetName returns the name of this plugin
	GetName() string

	// SkipCleanup returns whether cleanup for this plugin should be skipped or not.
	SkipCleanup() bool

	// GetResultFormat states the type of results this plugin generates and facilates post-processing
	// those results.
	GetResultFormat() string

	// GetResultFiles returns the specific files to target for post-processing. If empty, each
	// result format specifies its own heuristic for determining those files.
	GetResultFiles() []string

	// GetDescription returns the human-readable description of the plugin.
	GetDescription() string

	// GetSourceURL returns the URL where the plugin came from and where updates to it will be located.
	GetSourceURL() string
}

// ExpectedResult is an expected result that a plugin will submit.  This is so
// the aggregation server can know when it all results have been received.
type ExpectedResult struct {
	NodeName   string
	ResultType string
}

// Result represents a result we got from a dispatched plugin, returned to the
// aggregation server over HTTP.  Errors running a plugin are also considered a
// Result, if they have an Error property set.
type Result struct {
	NodeName   string
	ResultType string
	MimeType   string
	Filename   string
	Body       io.Reader
	Error      string
}

// ProgressUpdate is the structure that the Sonobuoy worker sends to the aggregator
// to inform it of plugin progress. More TBD.
type ProgressUpdate struct {
	PluginName string    `json:"name"`
	Node       string    `json:"node"`
	Timestamp  time.Time `json:"timestamp"`

	Message string `json:"msg"`

	Total     int64 `json:"total"`
	Completed int64 `json:"completed"`

	Errors   []string `json:"errors,omitempty"`
	Failures []string `json:"failures,omitempty"`

	AppendTotals    bool     `json:"appendtotals,omitempty"`
	AppendCompleted int64    `json:"appendcompleted,omitempty"`
	AppendFailing   []string `json:"appendfailing,omitempty"`
}

// CombineUpdates applies a second progress update onto the first. Latest message
// node and plugin name are always used.
func CombineUpdates(p1, p2 ProgressUpdate) ProgressUpdate {
	// If not appending, the new update is just the most recent one.
	if !p2.IsAppending() {
		return p2
	}

	p1.Node = p2.Node
	p1.PluginName = p2.PluginName
	p1.Timestamp = p2.Timestamp
	p1.Message = p2.Message

	// If we started from a known place, just append the new values.
	if !p1.IsAppending() {
		p1.Completed += p2.AppendCompleted
		if p2.AppendTotals {
			p1.Total += p2.AppendCompleted
			p1.Total += int64(len(p2.AppendFailing))
		}
		if len(p2.AppendFailing) > 0 {
			p1.Failures = append(p1.Failures, p2.AppendFailing...)
		}
		return p1
	}

	// Both are appending. Just return a single appending update with totals.
	p1.AppendCompleted = p1.AppendCompleted + p2.AppendCompleted
	if len(p2.AppendFailing) > 0 {
		p1.AppendFailing = append(p1.AppendFailing, p2.AppendFailing...)
	}
	// If these differ the resulting update may not be the same as appending both updates individually
	// since they would have had different impacts. Deferring to the second though.
	p1.AppendTotals = p2.AppendTotals
	return p1
}

func (p *ProgressUpdate) IsAppending() bool {
	return p.AppendTotals || len(p.AppendFailing) > 0 || p.AppendCompleted > 0
}

func (p *ProgressUpdate) IsEmpty() bool {
	return p.PluginName == "" &&
		p.Node == "" &&
		p.Message == "" &&
		p.Total == 0 &&
		p.Completed == 0 &&
		len(p.Errors) == 0 &&
		len(p.Failures) == 0 &&
		!p.AppendTotals &&
		p.AppendCompleted == 0 &&
		len(p.AppendFailing) == 0
}

// IsSuccess returns whether the Result represents a successful plugin result,
// versus one that was unsuccessful (for instance, from a dispatched plugin not
// being able to launch.)
func (r *Result) IsSuccess() bool {
	return r.Error == ""
}

// IsTimeout returns whether or not the Result represents the error case when Sonobuoy
// experiences a timeout waiting for results.
func (r *Result) IsTimeout() bool {
	return r.Error == TimeoutErrMsg
}

// Path is the path within the "plugins" section of the results tarball where
// this Result should be stored, not including a file extension.
func (r *Result) Path() string {
	if !r.IsSuccess() {
		return path.Join(r.ResultType, "errors", r.NodeName)
	}

	return path.Join(r.ResultType, "results", r.NodeName)
}

// Selection is the user specified input to load and initialize plugins
type Selection struct {
	Name string `json:"name"`
}

// AggregationConfig are the config settings for the server that aggregates plugin results
type AggregationConfig struct {
	BindAddress      string `json:"bindaddress"`
	BindPort         int    `json:"bindport"`
	AdvertiseAddress string `json:"advertiseaddress"`
	TimeoutSeconds   int    `json:"timeoutseconds"`
}

// WorkerConfig is the file given to the sonobuoy worker to configure it to phone home.
type WorkerConfig struct {
	// AggregatorURL is the URL we talk to the aggregator pod on for submitting results
	AggregatorURL string `json:"aggregatorurl,omitempty" mapstructure:"aggregatorurl"`

	// NodeName is the node name we should call ourselves when sending results
	NodeName string `json:"nodename,omitempty" mapstructure:"nodename"`

	// ResultsDir is the directory that's expected to contain the host's root filesystem
	ResultsDir string `json:"resultsdir,omitempty" mapstructure:"resultsdir"`

	// ResultType is the type of result (to be put in the HTTP URL's path) to be
	// sent back to sonobuoy.
	ResultType string `json:"resulttype,omitempty" mapstructure:"resulttype"`

	// ProgressUpdatesPort is the port on which the Sonobuoy worker will listen for progress
	// updates from the plugin.
	ProgressUpdatesPort string `json:"progressport"  mapstructure:"progressport"`

	CACert     string `json:"cacert,omitempty" mapstructure:"cacert"`
	ClientCert string `json:"clientcert,omitempty" mapstructure:"clientcert"`
	ClientKey  string `json:"clientkey,omitempty" mapstructure:"clientkey"`
}

// ID returns a unique identifier for this expected result to distinguish it
// from the rest of the results that may be seen by the aggregation server.
func (er *ExpectedResult) ID() string {
	if er.NodeName == "" {
		return er.ResultType
	}

	return er.ResultType + "/" + er.NodeName
}

// Key returns a unique identifier for this result to match it up
// against an expected result.
func (r *Result) Key() string {
	// Jobs should default to stating they are global results, being
	// safe and doing that defaulting here too.
	nodeName := r.NodeName
	if r.NodeName == "" {
		nodeName = GlobalResult
	}

	return r.ResultType + "/" + nodeName
}

// Key returns a unique identifier for this ProgressUpdate to match it up
// against an known plugins running.
func (s ProgressUpdate) Key() string {
	// Jobs should default to stating they are global results, being
	// safe and doing that defaulting here too.
	nodeName := s.Node
	if s.Node == "" {
		nodeName = GlobalResult
	}

	return s.PluginName + "/" + nodeName
}

// FormatPluginProgress returns a string that represents the current progress
// The string can then be used for printing updates.
// The format of the output is the following:
// Passed:S, Failed: F, Remaining: R
// Where S, F and R are numbers, 
// Corresponding to S = ProgressUpdate.Completed, F = len(ProgressUpdate.Failures),
// and ProgressUpdate.Total - S - F respectively
// and the ", Remaining: R" part is printed only if R is not negative
// 
func (s *ProgressUpdate) FormatPluginProgress() (output string) {
	//Minumum size of each field, in characters
	minSize := 3
	if s == nil {
		return ""
	}
	output = fmt.Sprintf("Passed:%[1]*[2]v, Failed:%[1]*[3]v", minSize, s.Completed, int64(len(s.Failures)))
	var remaining int64 = s.Total - s.Completed - int64(len(s.Failures))
	if remaining >= 0 {
		output += fmt.Sprintf(", Remaining:%[1]*[2]v", minSize, remaining)
	}
	return output
}
