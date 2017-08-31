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

package aggregation

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/pkg/errors"
)

// Server is a net/http server that can handle API requests for aggregation of
// results from nodes, sending them back over the Results channel
type Server struct {
	// BindAddr is the address for the HTTP server to listen on, eg. 0.0.0.0:8080
	BindAddr string
	// ResultsCallback is the function that is called when a result is checked in.
	ResultsCallback func(*plugin.Result, http.ResponseWriter)

	stopCh  chan bool
	readyCh chan bool
}

// NewServer constructs a new aggregation server which will listen for results
// on `bindAddr` and pass them to the given results callback.
func NewServer(bindAddr string, resultsCallback func(*plugin.Result, http.ResponseWriter)) *Server {
	return &Server{
		BindAddr:        bindAddr,
		ResultsCallback: resultsCallback,

		stopCh:  make(chan bool),
		readyCh: make(chan bool, 1),
	}
}

const (
	// we're using /api/v1 right now but aren't doing anything intelligent, if we
	// have an /api/v2 later we'll figure out a good strategy for splitting up the
	// handling.

	// resultsByNode is the HTTP path under which node results are PUT
	resultsByNode = "/api/v1/results/by-node/"
	// resultsByNode is the HTTP path under which global (whole-cluster)
	// results are PUT
	resultsGlobal = "/api/v1/results/global/"
)

// Stop stops a running Server
func (s *Server) Stop() {
	s.stopCh <- true
}

// Start starts this HTTP server, binding it to s.BindAddr and sending results
// over the s.Results channel. The first argument is the stop channel, which
// when written to will stop the server and close the HTTP socket. The second
// argument is the "ready" channel, which this function will write to once the
// HTTP server is ready for connections.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/", http.NotFoundHandler())

	// We handle /api/v1/results/by-node here, but we strip the prefix so that the
	// handling function only has to do some simple string splitting to get the node name.
	// An example call may look like `PUT /api/v1/results/by-node/node1/systemd_logs`,
	// which indicates that these are systemd_logs results for node1.
	mux.Handle(resultsByNode, http.StripPrefix(resultsByNode, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.nodeResultsHandler(w, r)
	})))
	// Same thing with /api/v1/results/global
	mux.Handle(resultsGlobal, http.StripPrefix(resultsGlobal, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.globalResultsHandler(w, r)
	})))
	srv := &http.Server{
		Addr:    s.BindAddr,
		Handler: mux,
	}

	l, err := net.Listen("tcp", s.BindAddr)

	if err != nil {
		return errors.Errorf("could not listen on %v: %v", s.BindAddr, err)
	}
	defer l.Close()

	logrus.Infof("Listening for incoming results on %v\n", s.BindAddr)

	done := make(chan error)
	go func() {
		done <- srv.Serve(l)
	}()
	// Let clients know they can send requests now (it's not perfect since the
	// above goroutine may not schedule right away, but it's the best we can do
	// and helps in testing.)
	s.readyCh <- true

	select {
	case <-s.stopCh:
		// Calling l.Close should make the http.Serve() above return
		l.Close()
		<-done
	case err = <-done:
		break
	}

	return err
}

// WaitUntilReady blocks until the server is listening on its configured
// address. This must only be called once for each time Start() is called, or
// it will block indefinitely.
func (s *Server) WaitUntilReady() {
	<-s.readyCh
}

// nodeResultsHandler handles requests to post results by node. Path must be
// stripped of the /api/v1/results/by-node prefix, leaving just
// :nodename/:type. The only supported method is PUT, this does not support
// reading existing data.  Example: PUT
// node1.cluster.local/api/v1/results/by-node/systemd_logs
func (s *Server) nodeResultsHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(r.URL.Path, "/", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	// We accept PUT because the client is specifying the resource identifier via
	// the HTTP path. (As opposed to POST, where typically the clients would post
	// to a base URL and the server picks the final resource path.)
	if r.Method != http.MethodPut {
		http.Error(
			w,
			fmt.Sprintf("Unsupported method %s.  Supported methods are: %v", r.Method, http.MethodPut),
			http.StatusMethodNotAllowed,
		)
		return
	}

	// Parse the path into the node name, result type, and extension
	node, file := parts[0], parts[1]
	resultType, extension := parseFileName(file)

	logrus.Infof("got %v result from %v\n", resultType, node)

	result := &plugin.Result{
		ResultType: resultType,
		Extension:  extension,
		NodeName:   node,
		Body:       r.Body,
	}

	// Trigger our callback with this checkin record (which should write the file
	// out.) The callback is responsible for doing a 409 conflict if results are
	// given twice for the same node, etc.
	s.ResultsCallback(result, w)
	r.Body.Close()
}

// globalResultsHandler handles requests to post results for the whole cluster. Path must be stripped
// of the /api/v1/results/global prefix, leaving just :type. The only supported
// method is PUT, this does not support reading existing data.
//
// Example: PUT node1.cluster.local/api/v1/results/global/e2e
func (s *Server) globalResultsHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 1 {
		logrus.Warningf("Returning 404 for request to %v", r.URL.Path)
		http.NotFound(w, r)
		return
	}

	// We accept PUT because the client is specifying the resource identifier via
	// the HTTP path. (As opposed to POST, where typically the clients would post
	// to a base URL and the server picks the final resource path.)
	if r.Method != http.MethodPut {
		logrus.Warningf("Got unsupported method %v from request to %v", r.Method, r.URL.Path)
		http.Error(
			w,
			fmt.Sprintf("Unsupported method %s.  Supported methods are: %v", r.Method, http.MethodPut),
			http.StatusMethodNotAllowed,
		)
		return
	}

	resultType, extension := parseFileName(parts[0])
	logrus.Infof("got %v result\n", resultType)

	result := &plugin.Result{
		NodeName:   "",
		ResultType: resultType,
		Extension:  extension,
		Body:       r.Body,
	}

	// Trigger our callback with this checkin record (which should write the file
	// out.) The callback is responsible for doing a 409 conflict if results are
	// given twice for the same node, etc.
	s.ResultsCallback(result, w)
	r.Body.Close()
}

// given an uploaded filename, parse it into its base name and extension.  If
// there are no "." characters, the extension will be blank and the name will
// be set to the filename as-is
func parseFileName(file string) (name string, extension string) {
	filenameParts := strings.SplitN(file, ".", 2)

	if len(filenameParts) == 2 {
		return filenameParts[0], "." + filenameParts[1]
	}

	return file, ""
}
