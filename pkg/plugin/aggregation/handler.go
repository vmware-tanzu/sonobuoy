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

package aggregation

import (
	"encoding/json"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
)

const (
	// we're using /api/v1 right now but aren't doing anything intelligent, if we
	// have an /api/v2 later we'll figure out a good strategy for splitting up the
	// handling.

	// PathResultsByNode is the path for node-specific results to be PUT to. Callers should
	// add two path elements as a suffix to this to specify the node and plugin (e.g. `<path>/node/plugin`)
	PathResultsByNode = "/api/v1/results/by-node"

	// PathResultsGlobal is the path for global (non-node-specific) results to be PUT to. Callers should
	// add one path element as a suffix to this to specify the plugin name (e.g. `<path>/plugin`)
	PathResultsGlobal = "/api/v1/results/global"

	// PathProgressByNode is the path for node-specific progress updates to be POSTed to. Callers should
	// add two path elements as a suffix to this to specify the node and plugin (e.g. `<path>/node/plugin`)
	PathProgressByNode = "api/v1/progress/by-node"

	// PathProgressGlobal is the path for progress updates to be POSTed to for global (non node-specific) plugins.
	// Callers should add one path element as a suffix to this to specify the plugin name (e.g. `<path>/plugin`)
	PathProgressGlobal = "/api/v1/progress/global"

	// resultsGlobal is the path for node-specific results to be PUT
	resultsByNode = PathResultsByNode + "/{node}/{plugin}"

	// resultsGlobal is the path for global (non-node-specific) results to be PUT
	resultsGlobal = PathResultsGlobal + "/{plugin}"

	// progressByNode is the path for progress updates to be POSTed to for node-specific plugins
	progressByNode = PathProgressByNode + "/{node}/{plugin}"

	// progressGlobal is the path for progress updates to be POSTed to for global (non node-specific) plugins
	progressGlobal = PathProgressGlobal + "/{plugin}"

	// defaultFilename is the name given to the file if no filename is given in the
	// content-disposition header
	defaultFilename = "result"
)

var (
	// Only used for route reversals
	r           = mux.NewRouter()
	nodeRoute   = r.Path(resultsByNode).BuildOnly()
	globalRoute = r.Path(resultsGlobal).BuildOnly()
)

// Handler is a net/http Handler that can handle API requests for aggregation of
// results from nodes, calling the provided callback with the results
type Handler struct {
	mux.Router

	// ResultsCallback is the function that is called when a result is checked in.
	ResultsCallback func(*plugin.Result, http.ResponseWriter)

	// ProgressCallback is the function that is called when a progress update is checked in.
	ProgressCallback func(plugin.ProgressUpdate, http.ResponseWriter)
}

// NewHandler constructs a new aggregation handler which will handler results
// and pass them to the given results callback.
func NewHandler(
	resultsCallback func(*plugin.Result, http.ResponseWriter),
	progressCallback func(plugin.ProgressUpdate, http.ResponseWriter),
) http.Handler {
	handler := &Handler{
		Router:           *mux.NewRouter(),
		ResultsCallback:  resultsCallback,
		ProgressCallback: progressCallback,
	}
	// We accept PUT because the client is specifying the resource identifier via
	// the HTTP path. (As opposed to POST, where typically the clients would post
	// to a base URL and the server picks the final resource path.)
	handler.HandleFunc(resultsByNode, handler.resultsHandler).Methods("PUT")
	handler.HandleFunc(resultsGlobal, handler.resultsHandler).Methods("PUT")

	handler.HandleFunc(progressByNode, handler.progressHandler).Methods("POST")
	handler.HandleFunc(progressGlobal, handler.progressHandler).Methods("POST")
	return handler
}

func resultFromRequest(r *http.Request, muxVars map[string]string) *plugin.Result {
	result := &plugin.Result{
		ResultType: muxVars["plugin"], // will be empty string in global case
		NodeName:   muxVars["node"],
		Body:       r.Body,
		MimeType:   r.Header.Get("content-type"),
		Filename:   filenameFromHeader(r.Header.Get("content-disposition")),
	}

	if result.NodeName == "" {
		result.NodeName = plugin.GlobalResult
	}

	return result
}

func progressFromRequest(r *http.Request, muxVars map[string]string) (plugin.ProgressUpdate, error) {
	var update plugin.ProgressUpdate
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&update)
	update.Node = muxVars["node"]
	if update.Node == "" {
		update.Node = plugin.GlobalResult
	}
	update.PluginName = muxVars["plugin"]
	update.Timestamp = time.Now()
	return update, errors.Wrap(err, "unable to decode body")
}

func (h *Handler) resultsHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	vars := mux.Vars(r)
	result := resultFromRequest(r, vars)

	// Trigger our callback with this checkin record (which should write the file
	// out.) The callback is responsible for doing a 409 conflict if results are
	// given twice for the same node, etc.
	h.ResultsCallback(result, w)
	r.Body.Close()
}

func (h *Handler) progressHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	vars := mux.Vars(r)
	update, err := progressFromRequest(r, vars)
	if err != nil {
		logrus.Errorf("Failed to get progress update from request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Trigger our callback with this checkin record (which should write the file
	// out.) The callback is responsible for doing a 409 conflict if results are
	// given twice for the same node, etc.
	h.ProgressCallback(update, w)
	r.Body.Close()
}

// NodeResultURL is the URL for results for a given node result. Takes the baseURL (http[s]://hostname:port/,
// with trailing slash) nodeName, pluginName, and an optional extension. If multiple
// extensions are provided, only the first one is used.
func NodeResultURL(baseURL, nodeName, pluginName string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get node result URL")
	}
	path, err := nodeRoute.URLPath("node", nodeName, "plugin", pluginName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get node result URL")
	}
	path.Scheme = base.Scheme
	path.Host = base.Host
	return path.String(), nil
}

// GlobalResultURL is the URL that results that are not node-specific. Takes the baseURL (http[s]://hostname:port/,
// with trailing slash) pluginName, and an optional extension. If multiple extensions are provided, only the first one
// is used.
func GlobalResultURL(baseURL, pluginName string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get global result URL ")
	}
	path, err := globalRoute.URLPath("plugin", pluginName)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get global result URL ")
	}
	path.Scheme = base.Scheme
	// Host includes port
	path.Host = base.Host
	return path.String(), nil
}

func logRequest(req *http.Request) {
	vars := mux.Vars(req)
	log := logrus.WithFields(map[string]interface{}{
		"plugin_name": vars["plugin"],
		"url":         req.URL,
		"method":      req.Method,
	})
	if node := vars["node"]; node != "" {
		log = log.WithField("node", node)
	}
	if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
		log = log.WithField("client_cert", req.TLS.PeerCertificates[0].DNSNames)
	}
	log.Info("received request")
}

// filenameFromHeader gets the filename from a content-disposition of the form:
// Content-Disposition: attachment; filename=foo.txt
// If there is an error parsing the string, the empty string is returned.
func filenameFromHeader(contentDisposition string) string {
	_, params, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return defaultFilename
	}
	if params["filename"] != "" {
		return params["filename"]
	}
	return defaultFilename
}
