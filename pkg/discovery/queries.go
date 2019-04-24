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

package discovery

import (
	"os"
	"path"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/heptio/sonobuoy/pkg/dynamic"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// NSResourceLocation is the place under which namespaced API resources (pods, etc) are stored
	NSResourceLocation = "resources/ns"
	// ClusterResourceLocation is the place under which non-namespaced API resources (nodes, etc) are stored
	ClusterResourceLocation = "resources/cluster"
	// HostsLocation is the place under which host information (configz, healthz) is stored
	HostsLocation = "hosts"
	// MetaLocation is the place under which snapshot metadata (query times, config) is stored
	MetaLocation = "meta"
	// listVerb is the API verb we ensure resources respond to in order to try and call List()
	listVerb = "list"
	// secretResourceName is the value of the Name field on Secrets. We will implicitly filter those if the user
	// tries to just query everything by not specifying a Resource list.
	secretResourceName = "secrets"
)

type listQuery func() (*unstructured.UnstructuredList, error)
type objQuery func() (interface{}, error)

// timedListQuery performs a list query and serialize the results
func timedListQuery(outpath string, file string, f listQuery) (time.Duration, error) {
	start := time.Now()
	list, err := f()
	duration := time.Since(start)
	if err != nil {
		return duration, err
	}

	if len(list.Items) > 0 {
		err = errors.WithStack(SerializeObj(list.Items, outpath, file))
	}
	return duration, err
}

func timedObjectQuery(outpath string, file string, f objQuery) (time.Duration, error) {
	start := time.Now()
	obj, err := f()
	duration := time.Since(start)
	if err != nil {
		return duration, err
	}

	return duration, errors.WithStack(SerializeObj(obj, outpath, file))
}

// timedQuery Wraps the execution of the function with a recorded timed snapshot
func timedQuery(recorder *QueryRecorder, name string, ns string, fn func() (time.Duration, error)) {
	duration, fnErr := fn()
	recorder.RecordQuery(name, ns, duration, fnErr)
}

func sliceContains(set []string, val string) bool {
	for _, v := range set {
		if v == val {
			return true
		}
	}
	return false
}

// given the filter options and a query against the given ns; what resources should we query? resourceNameList being empty means all. Only kept that for backwards compat.
func getResources(client *dynamic.APIHelper) (map[schema.GroupVersion][]metav1.APIResource, error) {
	resourceMap, err := client.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, err
	}

	// Some resources are ambiguously set in two or more groups. As kubectl
	// does, we should just prefer the first one returned by discovery.
	resourcesSeen := map[string]struct{}{}
	versionResourceMap := map[schema.GroupVersion][]metav1.APIResource{}
	for _, apiResourceList := range resourceMap {
		version, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrap(err, "parsing schema")
		}

		resources := []metav1.APIResource{}
		for _, apiResource := range apiResourceList.APIResources {
			// If we've seen the resource already, skip it.
			if _, ok := resourcesSeen[apiResource.Name]; ok {
				continue
			}
			resources = append(resources, apiResource)
			resourcesSeen[apiResource.Name] = struct{}{}
			continue
		}
		versionResourceMap[version] = resources
	}

	return versionResourceMap, nil
}

// QueryResources will query all the intended resources. If given a non-nil namespace
// it queries only namespaced objects; non-namespaced otherwise. Writing them out to
// <resultsdir>/resources/ns/<ns>/*.json or <resultsdir>/resources/cluster/*.json.
func QueryResources(
	client *dynamic.APIHelper,
	recorder *QueryRecorder,
	resources []schema.GroupVersionResource,
	ns *string,
	cfg *config.Config) error {

	if ns != nil {
		logrus.Infof("Running ns query (%v)", *ns)
	} else {
		logrus.Info("Running cluster queries")
	}

	// 1. Create the parent directory we will use to store the results
	outdir := path.Join(cfg.OutputDir(), ClusterResourceLocation)
	if ns != nil {
		outdir = path.Join(cfg.OutputDir(), NSResourceLocation, *ns)
	}

	if err := os.MkdirAll(outdir, 0755); err != nil {
		return errors.WithStack(err)
	}

	// 2. Setup label filter if there is one.
	opts := metav1.ListOptions{}
	if len(cfg.Filters.LabelSelector) > 0 {
		if _, err := labels.Parse(cfg.Filters.LabelSelector); err != nil {
			logrus.Warningf("Labelselector %v failed to parse with error %v", cfg.Filters.LabelSelector, err)
		} else {
			opts.LabelSelector = cfg.Filters.LabelSelector
		}
	}
	if ns != nil && len(*ns) > 0 {
		opts.FieldSelector = "metadata.namespace=" + *ns
	}

	// 3. Execute the query
	for _, gvr := range resources {
		lister := func() (*unstructured.UnstructuredList, error) {
			resourceClient := client.Client.Resource(gvr)
			resources, err := resourceClient.List(opts)

			return resources, errors.Wrapf(err, "listing resource %v", gvr)
		}

		// The core group is just the empty string but for clarity and consistency, refer to it as core.
		groupText := gvr.Group
		if groupText == "" {
			groupText = "core"
		}

		query := func() (time.Duration, error) {
			return timedListQuery(
				outdir+"/",
				groupText+"_"+gvr.Version+"_"+gvr.Resource+".json",
				lister,
			)
		}

		// Get the pretty-print namespace and avoid dereference issues.
		nsVal := ""
		if ns != nil {
			nsVal = *ns
		}

		timedQuery(recorder, gvr.Resource, nsVal, query)
	}

	return nil
}

// getAllFilteredResources figure out which resources we want to query for based on the filter list and whether
// or not we are considering namespaced objects or not.
func getAllFilteredResources(client *dynamic.APIHelper, wantResources []string) (clusterResources, nsResources []schema.GroupVersionResource, retErr error) {
	groupResources, err := getResources(client)
	if err != nil {
		return nil, nil, errors.Wrap(err, "choosing resources to gather")
	}
	return filterResources(groupResources, false, wantResources),
		filterResources(groupResources, true, wantResources),
		nil
}

func filterResources(gvrs map[schema.GroupVersion][]metav1.APIResource, namespaced bool, wantResources []string) []schema.GroupVersionResource {
	results := []schema.GroupVersionResource{}
	for gv, resources := range gvrs {
		for _, res := range resources {
			// Get either namespaced or non-namespaced resources.
			if namespaced != res.Namespaced {
				continue
			}

			// Double check the resources are listable.
			listable := false
			for _, v := range res.Verbs {
				if v == listVerb {
					listable = true
					break
				}
			}
			if !listable {
				continue
			}

			// Filter for explicit values
			if wantResources != nil {
				if !sliceContains(wantResources, res.Name) {
					logrus.Infof("%v not specified in non-nil Resources. Skipping %v query.", res.Name, res.Name)
					continue
				}
			} else {
				// Filter out secrets by default to avoid accidental exposure.
				if res.Name == secretResourceName {
					logrus.Infof("Resources is not set explicitly implying query all resources, but skipping %v for safety. Specify the value explicitly in Resources to gather this data.", res.Name)
					continue
				}
			}

			results = append(results, gv.WithResource(res.Name))
		}
	}
	return results
}

// QueryPodLogs gets the pod logs for each pod in the given namespace.
func QueryPodLogs(kubeClient kubernetes.Interface, recorder *QueryRecorder, ns string, cfg *config.Config) error {
	// Force podlogs gathering in the namespace Sonobuoy is in. Optional otherwise and
	// based on the Resources value.
	if cfg.Resources != nil &&
		!sliceContains(cfg.Resources, "podlogs") &&
		ns != cfg.Namespace {
		logrus.Infof("podlogs not specified in non-nil Resources, only getting podlogs for namespace %v", cfg.Namespace)
		return nil
	}

	start := time.Now()

	opts := metav1.ListOptions{}
	if len(cfg.Filters.LabelSelector) > 0 {
		if _, err := labels.Parse(cfg.Filters.LabelSelector); err != nil {
			logrus.Warningf("Labelselector %v failed to parse with error %v", cfg.Filters.LabelSelector, err)
		} else {
			opts.LabelSelector = cfg.Filters.LabelSelector
		}
	}

	err := gatherPodLogs(kubeClient, ns, opts, cfg)
	if err != nil {
		return err
	}
	duration := time.Since(start)
	recorder.RecordQuery("PodLogs", ns, duration, err)
	return nil
}

// QueryHostData gets the host data and records it.
func QueryHostData(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	if cfg.Resources != nil && !sliceContains(cfg.Resources, "nodes") {
		logrus.Info("nodes not specified in non-nil Resources. Skipping host data gathering.")
		return nil
	}

	start := time.Now()

	// TODO(chuckha) look at FieldSelector for list options{}
	nodeList, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get node list")
	}
	nodeNames := make([]string, len(nodeList.Items))
	for i, node := range nodeList.Items {
		nodeNames[i] = node.Name
	}
	err = gatherNodeData(nodeNames, kubeClient.CoreV1().RESTClient(), cfg)
	duration := time.Since(start)
	recorder.RecordQuery("Nodes", "", duration, err)

	return nil
}

// QueryServerData gets the server version and server group data and records it.
func QueryServerData(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	if err := queryServerVersion(kubeClient, recorder, cfg); err != nil {
		return err
	}

	if err := queryServerGroups(kubeClient, recorder, cfg); err != nil {
		return err
	}

	return nil
}

func queryServerVersion(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	if cfg.Resources != nil && !sliceContains(cfg.Resources, "serverversion") {
		logrus.Info("serverversion not specified in non-nil Resources. Skipping serverversion query.")
		return nil
	}

	objqry := func() (interface{}, error) { return kubeClient.Discovery().ServerVersion() }
	query := func() (time.Duration, error) {
		return timedObjectQuery(cfg.OutputDir(), "serverversion.json", objqry)
	}
	timedQuery(recorder, "serverversion", "", query)

	return nil
}

func queryServerGroups(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	if cfg.Resources != nil && !sliceContains(cfg.Resources, "servergroups") {
		logrus.Info("servergroups not specified in non-nil Resources. Skipping servergroups query.")
		return nil
	}
	objqry := func() (interface{}, error) { return kubeClient.Discovery().ServerVersion() }
	query := func() (time.Duration, error) {
		return timedObjectQuery(cfg.OutputDir(), "serverversion.json", objqry)
	}
	timedQuery(recorder, "serverversion", "", query)

	return nil
}
