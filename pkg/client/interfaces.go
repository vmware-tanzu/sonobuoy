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

	"k8s.io/client-go/rest"
)

// LogConfig is the options passed to GetLogs
type LogConfig struct {
	Follow    *bool
	Namespace string
}

// GenConfig is the input options for generating a Sonobuoy manifest
type GenConfig struct {
	ModeName  Mode
	Image     string
	Namespace string
}

// RunConfig is the input options for running Sonobuoy
type RunConfig struct {
	GenConfig
}

// CopyConfig is the options passed to CopyConfig.
type CopyConfig struct {
	Namespace  string
	RestConfig *rest.Config
	CmdErr     io.Writer
	Errc       chan error
}

// SonobuoyClient is a high-level interface to Sonobuoy operations.
type SonobuoyClient struct{}

// NewSonobuoyClient creates a new SonobuoyClient
func NewSonobuoyClient() *SonobuoyClient {
	return &SonobuoyClient{}
}

// Make sure SonobuoyClient implements the interface
var _ Interface = &SonobuoyClient{}

// NewClient creates a new sonobuoy client.

// Interface is the main contract that we will give to external consumers of this library
// This will provide a consistent look/feel to upstream and allow us to expose sonobuoy behavior
// to other automation systems.
type Interface interface {
	// functions that are exposed for consumption
	Run(cfg *RunConfig, restConfig *rest.Config) error
	GenerateManifest(cfg *GenConfig) ([]byte, error)
	// CopyResults(cfg *CopyConfig) io.Reader
	// GetStatus(namespace string, client kubernetes.Interface) (*aggregation.Status, error)
	// GetLogs(cfg *LogConfig, client kubernetes.Interface) error
}
