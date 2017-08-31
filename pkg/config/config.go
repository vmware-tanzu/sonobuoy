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

package config

import (
	"path"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/satori/go.uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterResources is the list of API resources that are scoped to the entire
// cluster (ie. not to any particular namespace)
var ClusterResources = []string{
	"CertificateSigningRequests",
	"ClusterRoleBindings",
	"ClusterRoles",
	"ComponentStatuses",
	"Nodes",
	"PersistentVolumes",
	"PodSecurityPolicies",
	"StorageClasses",
	"ThirdPartyResources",
}

// NamespacedResources is the list of API resources that are scoped to a
// kubernetes namespace.
var NamespacedResources = []string{
	"ConfigMaps",
	//"CronJobs",
	"DaemonSets",
	"Deployments",
	"Endpoints",
	"Events",
	"HorizontalPodAutoscalers",
	"Ingresses",
	"Jobs",
	"LimitRanges",
	"PersistentVolumeClaims",
	"PodDisruptionBudgets",
	"PodPresets",
	"PodTemplates",
	"Pods",
	"ReplicaSets",
	"ReplicationControllers",
	"ResourceQuotas",
	"RoleBindings",
	"Roles",
	"Secrets",
	"ServiceAccounts",
	"Services",
	"StatefulSets",
}

// SpecialResources are resources that aren't queried (or stored) the same was
// as the rest, so need special casing for querying them.
var SpecialResources = []string{
	"PodLogs",
	"ServerGroups",
	"ServerVersion",
}

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
	Kubeconfig  string `json:"Kubeconfig" mapstructure:"Kubeconfig"`

	///////////////////////////////////////////////
	// Data collection options
	///////////////////////////////////////////////
	Resources []string `json:"Resources" mapstructure:"Resources"`

	///////////////////////////////////////////////
	// Filtering options
	///////////////////////////////////////////////
	Filters FilterOptions `json:"Filters" mapstructure:"Filters"`

	Limits LimitConfig `json:"Limits" mapstructure:"Limits"`

	///////////////////////////////////////////////
	// plugin configurations settings
	///////////////////////////////////////////////
	Aggregation      plugin.AggregationConfig `json:"Server" mapstructure:"Server"`
	PluginSelections []plugin.Selection       `json:"Plugins" mapstructure:"Plugins"`
	PluginSearchPath []string                 `json:"PluginSearchPath" mapstructure:"PluginSearchPath"`
	PluginNamespace  string                   `json:"PluginNamespace" mapstructure:"PluginNamespace"`
	LoadedPlugins    []plugin.Interface       // this is assigned when plugins are loaded.
}

// LimitConfig is a configuration on the limits of sizes of various responses.
type LimitConfig struct {
	PodLogs SizeOrTimeLimitConfig `json:"PodLogs" mapstructure:"PodLogs"`
}

// SizeOrTimeLimitConfig represents configuration that limits the size of
// something either by a total disk size, or by a length of time.
type SizeOrTimeLimitConfig struct {
	LimitSize string `json:"LimitSize" mapstructure:"LimitSize"`
	LimitTime string `json:"LimitTime" mapstructure:"LimitTime"`
}

// FilterResources is a utility function used to parse Resources
func (cfg *Config) FilterResources(filter []string) map[string]bool {
	results := make(map[string]bool)

	for _, felement := range filter {
		for _, check := range cfg.Resources {
			if felement == check {
				results[felement] = true
			}
		}
	}
	return results
}

// OutputDir returns the directory under the ResultsDir containing the
// UUID for this run.
func (cfg *Config) OutputDir() string {
	return path.Join(cfg.ResultsDir, cfg.UUID)
}

// SizeLimitBytes returns how many bytes the configuration is set to limit,
// returning defaultVal if not set.
func (c SizeOrTimeLimitConfig) SizeLimitBytes(defaultVal int64) int64 {
	val, defaulted, err := c.sizeLimitBytes()

	// Ignore error, since we should have already caught it in validation
	if err != nil || defaulted {
		return defaultVal
	}

	return val
}

func (c SizeOrTimeLimitConfig) sizeLimitBytes() (val int64, defaulted bool, err error) {
	str := c.LimitSize
	if str == "" {
		return 0, true, nil
	}

	var bs datasize.ByteSize
	err = bs.UnmarshalText([]byte(str))
	return int64(bs.Bytes()), false, err
}

// TimeLimitDuration returns the duration the configuration is set to limit, returning defaultVal if not set.
func (c SizeOrTimeLimitConfig) TimeLimitDuration(defaultVal time.Duration) time.Duration {
	val, defaulted, err := c.timeLimitDuration()

	// Ignore error, since we should have already caught it in validation
	if err != nil || defaulted {
		return defaultVal
	}

	return val
}

func (c SizeOrTimeLimitConfig) timeLimitDuration() (val time.Duration, defaulted bool, err error) {
	str := c.LimitTime
	if str == "" {
		return 0, true, nil
	}

	val, err = time.ParseDuration(str)
	return val, false, err
}

// NewWithDefaults returns a newly-constructed Config object with default values.
func NewWithDefaults() *Config {
	var cfg Config
	cfg.UUID = uuid.NewV4().String()
	cfg.Description = "DEFAULT"
	cfg.ResultsDir = "./results"
	cfg.Version = buildinfo.Version

	cfg.Filters.Namespaces = ".*"

	cfg.Resources = ClusterResources
	cfg.Resources = append(cfg.Resources, NamespacedResources...)
	cfg.Resources = append(cfg.Resources, SpecialResources...)

	cfg.PluginNamespace = metav1.NamespaceSystem

	cfg.Aggregation.BindAddress = "0.0.0.0"
	cfg.Aggregation.BindPort = 8080
	cfg.Aggregation.TimeoutSeconds = 1800 // 30 minutes

	cfg.PluginSearchPath = []string{
		"./plugins.d",
		"/etc/sonobuoy/plugins.d",
		"~/sonobuoy/plugins.d",
	}

	return &cfg
}

// addPlugin adds a (configured, initialized) plugin to the config object so
// that it can be executed.
func (cfg *Config) addPlugin(plugin plugin.Interface) {
	cfg.LoadedPlugins = append(cfg.LoadedPlugins, plugin)
}

// getPlugins gets the list of plugins selected for this configuration.
func (cfg *Config) getPlugins() []plugin.Interface {
	return cfg.LoadedPlugins
}
