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

package client

import (
	"io"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// LogConfig are the options passed to GetLogs.
type LogConfig struct {
	// Follow determines if the logs should be followed or not (tail -f).
	Follow *bool
	// Namespace is the namespace the sonobuoy aggregator is running in.
	Namespace string
}

// GenConfig are the input options for generating a Sonobuoy manifest.
type GenConfig struct {
	E2EConfig  *E2EConfig
	Config     *config.Config
	Image      string
	Namespace  string
	EnableRBAC bool
}

// E2EConfig is the configuration of the E2E test.
type E2EConfig struct {
	Focus string
	Skip  string
}

// RunConfig are the input options for running Sonobuoy.
type RunConfig struct {
	GenConfig
	// SkipPreflight means don't run any checks before kicking off the Sonobuoy run.
	SkipPreflight bool
}

// RetrieveConfig are the options passed to RetrieveResults.
type RetrieveConfig struct {
	// CmdErr is the place to write errors to.
	CmdErr io.Writer
	// Errc reports errors from go routines that retrieve may spawn.
	Errc chan error
	// Namespace is the namespace the sonobuoy aggregator is running in.
	Namespace string
}

// SonobuoyClient is a high-level interface to Sonobuoy operations.
type SonobuoyClient struct{}

// NewSonobuoyClient creates a new SonobuoyClient
func NewSonobuoyClient() *SonobuoyClient {
	return &SonobuoyClient{}
}

// Make sure SonobuoyClient implements the interface
var _ Interface = &SonobuoyClient{}

// Interface is the main contract that we will give to external consumers of this library
// This will provide a consistent look/feel to upstream and allow us to expose sonobuoy behavior
// to other automation systems.
type Interface interface {
	// Run generates the manifest, then tries to apply it to the cluster.
	// returns created resources or an error
	Run(cfg *RunConfig, restConfig *rest.Config) error
	// GenerateManifest fills in a template with a Sonobuoy config
	GenerateManifest(cfg *GenConfig) ([]byte, error)
	// RetrieveResults copies results from a sonobuoy run into a Reader in tar format.
	RetrieveResults(cfg *RetrieveConfig, restConfig *rest.Config) io.Reader
	// GetStatus determines the status of the sonobuoy run in order to assist the user.
	GetStatus(namespace string, client kubernetes.Interface) (*aggregation.Status, error)
	// GetLogs streams logs from the sonobuoy pod by default to stdout.
	GetLogs(cfg *LogConfig, client kubernetes.Interface) error
	// Delete removes a sonobuoy run, namespace, and all associated tests.
	Delete(namespace string, client kubernetes.Interface) error
}
