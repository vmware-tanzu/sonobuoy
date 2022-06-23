/*
Copyright 2018 Heptio Inc.

Copyright 2022 the Sonobuoy Project contributors

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

package config

import (
	"fmt"
	"os"
	"path"

	"github.com/vmware-tanzu/sonobuoy/pkg/buildinfo"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultNamespace is the namespace where the aggregator and plugin workers will run (but not necessarily the pods created by the plugin workers).
	DefaultNamespace = "sonobuoy"

	// UpstreamKubeConformanceImageURL is the URL of the docker image to run for
	// the kube conformance tests which is maintained by upstream Kubernetes.
	UpstreamKubeConformanceImageURL = "k8s.gcr.io/conformance"
	// DefaultAggregationServerBindPort is the default port for the aggregation server to bind to.
	DefaultAggregationServerBindPort = 8080
	// DefaultAggregationServerBindAddress is the default address for the aggregation server to bind to.
	DefaultAggregationServerBindAddress = "0.0.0.0"
	// DefaultAggregationServerTimeoutSeconds is the default amount of time the aggregation server will wait for all plugins to complete.
	DefaultAggregationServerTimeoutSeconds = 21600 // 360 min
	// AggregatorPodName is the name of the main pod that runs plugins and collects results.
	AggregatorPodName = "sonobuoy"
	// AggregatorContainerName is the name of the main container in the aggregator pod.
	AggregatorContainerName = "kube-sonobuoy"
	// AggregatorResultsPath is the location in the main container of the aggregator pod where results will be archived.
	// It is different from the default results path for plugins (/tmp/sonobuoy/results).
	AggregatorResultsPath = "/tmp/sonobuoy"
	// DefaultSonobuoyPullPolicy is the default pull policy used in the Sonobuoy config.
	DefaultSonobuoyPullPolicy = "IfNotPresent"
	// DefaultQueryQPS is the number of queries per second Sonobuoy will make when gathering data.
	DefaultQueryQPS = 30
	// DefaultQueryBurst is the peak number of queries per second Sonobuoy will make when gathering data.
	DefaultQueryBurst = 50
	// DefaultProgressUpdatesPort is the port on which the Sonobuoy worker will listen for status updates from its plugin.
	DefaultProgressUpdatesPort = "8099"

	// DefaultDNSNamespace is the namespace where the DNS pods for the cluster are found.
	DefaultDNSNamespace = "kube-system"

	// DefaultSystemdLogsImage is the URL for the docker image used by the systemd-logs plugin
	DefaultSystemdLogsImage = "sonobuoy/systemd-logs:v0.4"

	// DefaultSystemdLogsImageNameAndTag is the name/tag for systemd-logs plugin without the registry.
	DefaultSystemdLogsImageNameAndTag = "systemd-logs:v0.4"

	// DefaultSecurityContextMode is a shorthand for common security context values. Default is nonroot which sets
	// runAsUser, runAsGroup, and fsGroup. 'none' skips setting those it entirely since Windows does not support them.
	DefaultSecurityContextMode = "nonroot"

	AggregatorPermissionsClusterAdmin   = "clusterAdmin"
	AggregatorPermissionsClusterRead    = "clusterRead"
	AggregatorPermissionsNamespaceAdmin = "namespaceAdmin"
	DefaultAggregatorPermissions        = AggregatorPermissionsClusterAdmin

	// DefaultServiceAccountName is the default Service Account name that should be used if no customization is provided
	DefaultServiceAccountName = "sonobuoy-serviceaccount"
)

var (
	// DefaultImage is the URL of the docker image to run for the aggregator and workers
	DefaultImage = "sonobuoy/sonobuoy:" + buildinfo.Version

	// DefaultResources is the default set of resources which are queried for after plugins run. The strings
	// are compared against the resource.Name given by the client-go discovery client. The non-standard values
	// that are included here are: podlogs, servergroups, serverversion. The value 'nodes', although a crawlable
	// API value, also is used to query against the healthz and configz endpoints on the node.
	// By providing the nil value we query them all (except secrets).
	DefaultResources = []string(nil)

	// DefaultDNSPodLabels are the label selectors that are used to locate the DNS pods in the cluster.
	DefaultDNSPodLabels = []string{
		"k8s-app=kube-dns",
		"k8s-app=coredns",
	}
)

// FilterOptions allow operators to select sets to include in a report
type FilterOptions struct {
	Namespaces    string `json:"Namespaces"`
	LabelSelector string `json:"LabelSelector"`
}

// Config is the input struct used to determine what data to collect.
type Config struct {
	// NOTE: viper uses "mapstructure" as the tag for config
	// serialization, *NOT* "json".  mapstructure is a separate library
	// that converts maps to structs, and has its own syntax for tagging
	// fields. The only documentation on this is in the mapstructure docs:
	// https://godoc.org/github.com/mitchellh/mapstructure#example-Decode--Tags
	//
	// To be safe we annotate with both json and mapstructure tags.

	///////////////////////////////////////////////
	// Meta-Data collection options
	///////////////////////////////////////////////
	Description string `json:"Description" mapstructure:"Description"`
	UUID        string `json:"UUID" mapstructure:"UUID"`
	Version     string `json:"Version" mapstructure:"Version"`
	ResultsDir  string `json:"ResultsDir" mapstructure:"ResultsDir"`
	QueryDir    string `json:"QueryDir,omitempty" mapstructure:"QueryDir"`

	///////////////////////////////////////////////
	// Query options
	///////////////////////////////////////////////
	Resources []string      `json:"Resources" mapstructure:"Resources"`
	Filters   FilterOptions `json:"Filters" mapstructure:"Filters"`
	Limits    LimitConfig   `json:"Limits" mapstructure:"Limits"`
	QPS       float32       `json:"QPS,omitempty" mapstructure:"QPS"`
	Burst     int           `json:"Burst,omitempty" mapstructure:"Burst"`

	///////////////////////////////////////////////
	// Plugin configurations settings
	///////////////////////////////////////////////
	Aggregation      plugin.AggregationConfig `json:"Server" mapstructure:"Server"`
	PluginSelections []plugin.Selection       `json:"Plugins" mapstructure:"Plugins"`
	PluginSearchPath []string                 `json:"PluginSearchPath" mapstructure:"PluginSearchPath"`
	Namespace        string                   `json:"Namespace" mapstructure:"Namespace"`
	SkipPlugins      bool                     `json:"SkipPlugins,omitempty"`
	LoadedPlugins    []plugin.Interface       `json:"-"` // this is assigned when plugins are loaded.

	///////////////////////////////////////////////
	// Sonobuoy configuration
	///////////////////////////////////////////////
	WorkerImage            string            `json:"WorkerImage" mapstructure:"WorkerImage"`
	ImagePullPolicy        string            `json:"ImagePullPolicy" mapstructure:"ImagePullPolicy"`
	ForceImagePullPolicy   bool              `json:"ForceImagePullPolicy,omitempty" mapstructure:"ForceImagePullPolicy"`
	ImagePullSecrets       string            `json:"ImagePullSecrets" mapstructure:"ImagePullSecrets"`
	CustomAnnotations      map[string]string `json:"CustomAnnotations,omitempty" mapstructure:"CustomAnnotations"`
	AggregatorPermissions  string            `json:"AggregatorPermissions" mapstructure:"AggregatorPermissions"`
	ServiceAccountName     string            `json:"ServiceAccountName" mapstructure:"ServiceAccountName"`
	ExistingServiceAccount bool              `json:"ExistingServiceAccount,omitempty" mapstructure:"ExistingServiceAccount,omitempty"`

	// ProgressUpdatesPort is the port on which the Sonobuoy worker will listen for status updates from its plugin.
	ProgressUpdatesPort string `json:"ProgressUpdatesPort,omitempty" mapstructure:"ProgressUpdatesPort"`

	// SecurityCtx allows uers to specify the security context for the aggregator pod. This is particularly
	// important for two cases: local security prevents running without specifying non-root users and Windows
	// nodes do not currently support runAsUser/Group settings so if we default so something we need to allow
	// users to opt out easily.
	SecurityContextMode string `json:"SecurityContextMode" mapstructure:"SecurityContextMode"`
}

// LimitConfig is a configuration on the limits of various responses, such as limits of sizes
type LimitConfig struct {
	PodLogs PodLogLimits `json:"PodLogs" mapstructure:"PodLogs"`
}

// PodLogLimits limits the scope of response when getting logs from pods.
// The scope of pods is defined by (Namespaces OR SonobuoyNamespace OR FieldSelectors) AND LabelSelector
// Then for each pod, the format and size of logs is defined by other fields, e.g. SinceSeconds
type PodLogLimits struct {
	// A regular expression for the targeted namespaces, follows the format of Golang Regexp
	Namespaces string `json:"Namespaces" mapstructure:"Namespaces"`

	// If set to true, take into account the pod logs from Sonobuoy's namespace, i.e. Config.Namespace
	// Use a pointer here to differentiate an empty input or an input of false.
	// Default value will be true, empty input could be overwritten by default value
	SonobuoyNamespace *bool `json:"SonobuoyNamespace" mapstructure:"SonobuoyNamespace"`

	// Filtering candidate pods by their fields
	// This is an array of FieldSelector, each of them follows the same format as:
	// k8s.io/apimachinery/pkg/apis/meta/v1/types/ListOptions/FieldSelector
	//
	// One field selector contains one or more chained operators, with AND logic
	// e.g. FieldSelectors = ["status.phase==Running,spec.restartPolicy=Always"] means
	// status.phase==Running AND spec.restartPolicy=Always
	//
	// Multiple field selectors can be provided, with OR logic:
	// e.g. FieldSelectors = ["status==Pending", "status.phase==Running,spec.restartPolicy=Always"] means
	// status==Pending OR ( status.phase==Running AND spec.restartPolicy=Always )
	FieldSelectors []string `json:"FieldSelectors" mapstructure:"FieldSelectors"`

	// Filtering candidate pods by their labels
	// using the same format as k8s.io/apimachinery/pkg/apis/meta/v1/types/ListOptions/LabelSelector.
	// The input should follow this form:
	//
	//  <selector-syntax>         ::= <requirement> | <requirement> "," <selector-syntax>
	//  <requirement>             ::= [!] KEY [ <set-based-restriction> | <exact-match-restriction> ]
	//  <set-based-restriction>   ::= "" | <inclusion-exclusion> <value-set>
	//  <inclusion-exclusion>     ::= <inclusion> | <exclusion>
	//  <exclusion>               ::= "notin"
	//  <inclusion>               ::= "in"
	//  <value-set>               ::= "(" <values> ")"
	//  <values>                  ::= VALUE | VALUE "," <values>
	//  <exact-match-restriction> ::= ["="|"=="|"!="] VALUE
	//
	// KEY is a sequence of one or more characters following [ DNS_SUBDOMAIN "/" ] DNS_LABEL. Max length is 63 characters.
	// VALUE is a sequence of zero or more characters "([A-Za-z0-9_-\.])". Max length is 63 characters.
	// Delimiter is white space: (' ', '\t')
	//
	// Example of valid syntax:
	//  "x in (foo,,baz),y,z notin ()"
	//
	LabelSelector string `json:"LabelSelector" mapstructure:"LabelSelector"`

	// Return previous terminated container logs. Defaults to false.
	// +optional
	Previous bool `json:"Previous" mapstructure:"Previous"`

	// A relative time in seconds before the current time from which to show logs. If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of sinceSeconds or sinceTime may be specified.
	// +optional
	SinceSeconds *int64 `json:"SinceSeconds" mapstructure:"SinceSeconds"`

	// An RFC3339 timestamp from which to show logs, e.g. "2019-06-28T00:34:00Z" . If this value
	// precedes the time a pod was started, only logs since the pod start will be returned.
	// If this value is in the future, no logs will be returned.
	// Only one of LimitTime or sinceTime may be specified.
	// +optional
	SinceTime *metav1.Time `json:"SinceTime" mapstructure:"SinceTime"`

	// If true, add an RFC3339 or RFC3339Nano timestamp at the beginning of every line
	// of log output. Defaults to false.
	// +optional
	Timestamps bool `json:"Timestamps" mapstructure:"Timestamps"`

	// If set, the number of lines from the end of the logs to show. If not specified,
	// logs are shown from the creation of the container or sinceSeconds or sinceTime
	// +optional
	TailLines *int64 `json:"TailLines" mapstructure:"TailLines"`

	// If set, the number of bytes to read from the server before terminating the
	// log output. This may not display a complete final line of logging, and may return
	// slightly more or slightly less than the specified limit.
	// +optional
	LimitBytes *int64 `json:"LimitBytes" mapstructure:"LimitBytes"`
}

// FilterResources is a utility function used to parse Resources
func (cfg *Config) FilterResources(filter []string) []string {
	var results []string
	for _, felement := range filter {
		for _, check := range cfg.Resources {
			if felement == check {
				results = append(results, felement)
			}
		}
	}
	return results
}

// OutputDir returns the AggregatorResultsPath/:UUID. Hard-coded aggregator results path
// to avoid miscommunication between the local host and aggregator during retrieve.
func (cfg *Config) OutputDir() string {
	return path.Join(AggregatorResultsPath, cfg.UUID)
}

// QueryOutputDir returns the QueryDir if set and falls back to the
// AggregatorResultsPath/:UUID to work on the aggregator by default.
func (cfg *Config) QueryOutputDir() string {
	// Override option for when running via CLI.
	if len(cfg.QueryDir) > 0 {
		return cfg.QueryDir
	}
	// Default for when running via aggregator.
	return cfg.OutputDir()
}

// New returns a newly-constructed Config object with default values.
func New() *Config {
	var cfg Config
	cfg.Description = "DEFAULT"
	// Note: The resultsDir here is just for the plugins, not the aggregator.
	cfg.ResultsDir = plugin.ResultsDir
	cfg.Version = buildinfo.Version

	cfg.Filters.Namespaces = ".*"

	cfg.QPS = DefaultQueryQPS
	cfg.Burst = DefaultQueryBurst
	cfg.Resources = DefaultResources

	cfg.Namespace = DefaultNamespace
	cfg.Limits.PodLogs.Namespaces = "kube-system"
	cfg.Limits.PodLogs.SonobuoyNamespace = new(bool)
	*cfg.Limits.PodLogs.SonobuoyNamespace = true
	cfg.Limits.PodLogs.FieldSelectors = []string{}

	cfg.Aggregation.BindAddress = DefaultAggregationServerBindAddress
	cfg.Aggregation.BindPort = DefaultAggregationServerBindPort
	cfg.Aggregation.TimeoutSeconds = DefaultAggregationServerTimeoutSeconds

	cfg.PluginSearchPath = []string{
		"./plugins.d",
		"/etc/sonobuoy/plugins.d",
		"~/sonobuoy/plugins.d",
	}

	switch devRepo := os.Getenv("SONOBUOY_DEV_REPO"); {
	case len(devRepo) > 0:
		cfg.WorkerImage = fmt.Sprintf("%v/sonobuoy:%v", devRepo, buildinfo.Version)
	default:
		cfg.WorkerImage = DefaultImage
	}

	cfg.ImagePullPolicy = DefaultSonobuoyPullPolicy

	cfg.ProgressUpdatesPort = DefaultProgressUpdatesPort

	cfg.SecurityContextMode = DefaultSecurityContextMode

	cfg.AggregatorPermissions = DefaultAggregatorPermissions

	cfg.ServiceAccountName = DefaultServiceAccountName

	cfg.ExistingServiceAccount = false

	return &cfg
}

// addPlugin adds a (configured, initialized) plugin to the config object so
// that it can be executed.
func (cfg *Config) addPlugin(plugin plugin.Interface) {
	cfg.LoadedPlugins = append(cfg.LoadedPlugins, plugin)
}
