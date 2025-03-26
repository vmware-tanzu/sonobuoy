/*
Copyright 2021 Sonobuoy Contributors

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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/dynamic"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Query cluster runs multiple queries against the cluster in order to obtain debug
// information.
func QueryCluster(restConf *rest.Config, cfg *config.Config) error {
	// Adjust QPS/Burst so that the queries execute as quickly as possible.
	restConf.QPS = float32(cfg.QPS)
	restConf.Burst = cfg.Burst

	apiHelper, err := dynamic.NewAPIHelperFromRESTConfig(restConf)
	if err != nil {
		return errors.Wrap(err, "failed to get APIHelper from REST config")
	}

	kubeClient, err := kubernetes.NewForConfig(restConf)
	if err != nil {
		return errors.Wrap(err, "could not create kubernetes client")
	}

	// Create the directory which will store the results, including the
	// `meta` directory inside it (which we always need regardless of
	// config)
	outpath := cfg.QueryOutputDir()
	metapath := filepath.Join(outpath, MetaLocation)
	err = os.MkdirAll(metapath, 0o755)
	if err != nil {
		return errors.Wrap(err, "could not create directory to store results")
	}

	// Run the queries
	recorder := NewQueryRecorder()
	clusterResources, nsResources, err := getAllFilteredResources(apiHelper, cfg.Resources)
	if err != nil {
		return errors.Wrap(err, "unable to filter resources")
	}

	if err := QueryHostData(kubeClient, recorder, cfg); err != nil {
		logrus.Errorf("Failed to query host data: %v", err)
	}

	if err := QueryResources(apiHelper, recorder, clusterResources, nil, cfg); err != nil {
		logrus.Errorf("Failed to query cluster resources: %v", err)
	}

	if err := QueryServerData(kubeClient, recorder, cfg); err != nil {
		logrus.Errorf("Failed to query server data: %v", err)
	}

	// Get the list of namespaces and apply the regex filter on the namespace
	logrus.Infof("Filtering namespaces based on the following regex:%s", cfg.Filters.Namespaces)
	nslist, err := FilterNamespaces(kubeClient, cfg.Filters.Namespaces, cfg.Namespace)
	if err != nil {
		logrus.Errorf("could not filter namespaces, will not query every namespace: %v", err)
	}

	for _, ns := range nslist {
		if err := QueryResources(apiHelper, recorder, nsResources, &ns, cfg); err != nil {
			logrus.Errorf("Failed to query resources for namespace %q: %v", ns, err)
		}
	}

	// Query pod logs
	if cfg.Resources == nil || sliceContains(cfg.Resources, "podlogs") {
		// Eliminate duplicate pods when query by namespaces and query by fieldSelectors
		visitedPods := make(map[string]struct{})

		// Query by namespace if there is a filter.
		nsFilter := getPodLogNamespaceFilter(cfg)
		if len(nsFilter) > 0 {
			var nsListLogs []string
			if cfg.Limits.PodLogs.SonobuoyNamespace != nil && *cfg.Limits.PodLogs.SonobuoyNamespace {
				nsListLogs, _ = FilterNamespaces(kubeClient, nsFilter, cfg.Namespace)
			} else {
				nsListLogs, _ = FilterNamespaces(kubeClient, nsFilter)
			}
			for _, ns := range nsListLogs {
				logrus.Info("querying pod logs under namespace " + ns)
				if err := QueryPodLogs(kubeClient, recorder, ns, cfg, visitedPods); err != nil {
					logrus.Errorf("Failed to query pod logs for namespace %q: %v", ns, err)
				}
			}
		}

		// Now query by field selectors by not providing a namespace.
		// visitedPods will prevent us from duplicating logs.
		if err := QueryPodLogs(kubeClient, recorder, "", cfg, visitedPods); err != nil {
			logrus.Errorf("Failed to query pod logs: %v", err)
		}
	} else {
		logrus.Infof("podlogs not specified in non-nil Resources, skipping getting podlogs")
	}

	// Dump the query times
	logrus.Infof("recording query times at %v", filepath.Join(metapath, "query-time.json"))
	if err := recorder.DumpQueryData(filepath.Join(metapath, "query-time.json")); err != nil {
		logrus.Errorf("Failed to write query times: %v", err)
	}

	return nil
}

// FilterNamespaces filter the list of namespaces according to the filter string. If unable
// to query all the namespaces, an optional defaultNS can be provided which will be added
// to the returned list if it matches the filter.
func FilterNamespaces(kubeClient kubernetes.Interface, filter string, defaultNS ...string) ([]string, error) {
	var validns []string
	re := regexp.MustCompile(filter)
	nslist, err := kubeClient.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		// Even if we can't get all namespaces, check the current one and add it to the list.
		if len(defaultNS) > 0 && re.MatchString(defaultNS[0]) {
			validns = append(validns, defaultNS[0])
		}
		return validns, errors.WithStack(err)
	}

	for _, ns := range nslist.Items {
		logrus.Infof("Namespace %v Matched=%v", ns.Name, re.MatchString(ns.Name))
		if re.MatchString(ns.Name) {
			validns = append(validns, ns.Name)
		}
	}
	return validns, nil
}

const (
	// NSResourceLocation is the place under which namespaced API resources (pods, etc) are stored
	NSResourceLocation = "resources/ns"
	// ClusterResourceLocation is the place under which non-namespaced API resources (nodes, etc) are stored
	ClusterResourceLocation = "resources/cluster"
	// HostsLocation is the place under which host information (configz, healthz) is stored
	HostsLocation = "hosts"
	// listVerb is the API verb we ensure resources respond to in order to try and call List()
	listVerb = "list"
	// secretResourceName is the value of the Name field on Secrets. We will implicitly filter those if the user
	// tries to just query everything by not specifying a Resource list.
	secretResourceName = "secrets"
)

type (
	listQuery func() (*unstructured.UnstructuredList, error)
	objQuery  func() (interface{}, error)
)

// timedListQuery performs a list query and serialize the results
func timedListQuery(outpath string, file string, f listQuery) (time.Duration, error) {
	start := time.Now()
	list, err := f()
	duration := time.Since(start)
	if err != nil {
		return duration, err
	}

	if len(list.Items) > 0 {
		err = errors.WithStack(SerializeObj(list, outpath, file))
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
// <outputDir>/resources/ns/<ns>/*.json or <outputDir>/resources/cluster/*.json.
func QueryResources(
	client *dynamic.APIHelper,
	recorder *QueryRecorder,
	resources []schema.GroupVersionResource,
	ns *string,
	cfg *config.Config,
) error {
	// Early exit; avoid forming query or creating output directories.
	if len(resources) == 0 {
		return nil
	}

	if ns != nil {
		logrus.Infof("Running ns query (%v)", *ns)
	} else {
		logrus.Info("Running cluster queries")
	}

	// Create the parent directory we will use to store the results
	outdir := filepath.Join(cfg.QueryOutputDir(), ClusterResourceLocation)
	if ns != nil {
		outdir = filepath.Join(cfg.QueryOutputDir(), NSResourceLocation, *ns)
	}

	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return errors.WithStack(err)
	}

	// Setup label filter if there is one.
	opts := metav1.ListOptions{}
	if len(cfg.Filters.LabelSelector) > 0 {
		if _, err := labels.Parse(cfg.Filters.LabelSelector); err != nil {
			logrus.Warningf("Labelselector %v failed to parse with error %v", cfg.Filters.LabelSelector, err)
		} else {
			opts.LabelSelector = cfg.Filters.LabelSelector
		}
	}

	// Execute the query
	for _, gvr := range resources {
		lister := func() (*unstructured.UnstructuredList, error) {
			resourceClient := client.Client.Resource(gvr)
			if ns != nil && len(*ns) > 0 {
				// Use a namespaced query rather than using a metadata.namespace filter to
				// avoid permissions issues.
				objs, err := resourceClient.Namespace(*ns).List(context.TODO(), opts)
				return objs, errors.Wrapf(err, "listing resource %v", gvr)
			} else {
				objs, err := resourceClient.List(context.TODO(), opts)
				return objs, errors.Wrapf(err, "listing resource %v", gvr)
			}
		}

		// The core group is just the empty string but for clarity and consistency, refer to it as core.
		groupText := gvr.Group
		if groupText == "" {
			groupText = "core"
		}

		query := func() (time.Duration, error) {
			return timedListQuery(
				outdir,
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
// If namespace is not provided, get pod logs using field selectors.
// VisitedPods will eliminate duplicate pods when execute overlapping queries,
// e.g. query by namespaces and query by fieldSelectors.
func QueryPodLogs(kubeClient kubernetes.Interface, recorder *QueryRecorder, ns string, cfg *config.Config,
	visitedPods map[string]struct{},
) error {
	start := time.Now()

	opts := metav1.ListOptions{}
	if len(cfg.Limits.PodLogs.LabelSelector) > 0 {
		if _, err := labels.Parse(cfg.Limits.PodLogs.LabelSelector); err != nil {
			logrus.Warningf("Labelselector %v failed to parse with error %v", cfg.Limits.PodLogs.LabelSelector, err)
		} else {
			opts.LabelSelector = cfg.Limits.PodLogs.LabelSelector
		}
	}

	if len(ns) > 0 {
		logrus.Infof("Collecting Pod Logs by namespace (%v)", ns)
		err := gatherPodLogs(kubeClient, ns, opts, cfg, visitedPods)
		if err != nil {
			return err
		}
	} else {
		logrus.Infof("Collecting Pod Logs by FieldSelectors: %q", cfg.Limits.PodLogs.FieldSelectors)
		for _, fieldSelector := range cfg.Limits.PodLogs.FieldSelectors {
			opts.FieldSelector = fieldSelector
			err := gatherPodLogs(kubeClient, ns, opts, cfg, visitedPods)
			if err != nil {
				return err
			}
		}
	}

	duration := time.Since(start)
	recorder.RecordQuery("PodLogs", ns, duration, nil)
	return nil
}

// QueryHostData gets the host data and records it.
func QueryHostData(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	if cfg.Resources != nil && !sliceContains(cfg.Resources, "nodes") {
		logrus.Info("nodes not specified in non-nil Resources. Skipping host data gathering.")
		return nil
	}

	start := time.Now()

	nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
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
	logrus.Infof("Querying server version and API Groups")
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
		return timedObjectQuery(cfg.QueryOutputDir(), "serverversion.json", objqry)
	}
	timedQuery(recorder, "serverversion", "", query)

	return nil
}

func queryServerGroups(kubeClient kubernetes.Interface, recorder *QueryRecorder, cfg *config.Config) error {
	if cfg.Resources != nil && !sliceContains(cfg.Resources, "servergroups") {
		logrus.Info("servergroups not specified in non-nil Resources. Skipping servergroups query.")
		return nil
	}
	objqry := func() (interface{}, error) { return kubeClient.Discovery().ServerGroups() }
	query := func() (time.Duration, error) {
		return timedObjectQuery(cfg.QueryOutputDir(), "servergroups.json", objqry)
	}
	timedQuery(recorder, "servergroups", "", query)

	return nil
}

// QueryRecorder records a sequence of queries
type QueryRecorder struct {
	queries []*QueryData
}

// NewQueryRecorder returns a new empty QueryRecorder
func NewQueryRecorder() *QueryRecorder {
	return &QueryRecorder{
		queries: make([]*QueryData, 0),
	}
}

// QueryData captures the results of the run for post-processing
type QueryData struct {
	QueryObj    string `json:"queryobj,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ElapsedTime string `json:"time,omitempty"`
	Error       error  `json:"error,omitempty"`
}

// RecordQuery transcribes a query by name, namespace, duration and error
func (q *QueryRecorder) RecordQuery(name string, namespace string, duration time.Duration, recerr error) {
	if recerr != nil {
		errlog.LogError(errors.Wrapf(recerr, "error querying %v", name))
	}
	summary := &QueryData{
		QueryObj:    name,
		Namespace:   namespace,
		ElapsedTime: duration.String(),
		Error:       recerr,
	}

	q.queries = append(q.queries, summary)
}

// DumpQueryData writes query information out to a file at the give filepath
func (q *QueryRecorder) DumpQueryData(filepath string) error {
	// Format the query data as JSON
	data, err := json.Marshal(q.queries)
	if err != nil {
		return err
	}

	// Ensure the leading path is created
	if err := os.MkdirAll(path.Dir(filepath), 0o755); err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0o755)
}

const (
	// PodLogsLocation is the location within the results tarball where pod
	// information is stored.
	PodLogsLocation = "podlogs"
)

// mapping to k8s.io/api/core/v1/PodLogOptions.
func getPodLogOptions(cfg *config.Config) *v1.PodLogOptions {
	podLogLimits := &cfg.Limits.PodLogs

	options := &v1.PodLogOptions{
		Previous:     podLogLimits.Previous,
		SinceSeconds: podLogLimits.SinceSeconds,
		SinceTime:    podLogLimits.SinceTime,
		Timestamps:   podLogLimits.Timestamps,
		TailLines:    podLogLimits.TailLines,
		LimitBytes:   podLogLimits.LimitBytes,
	}

	return options
}

// gatherPodLogs will loop through collecting pod logs and placing them into a directory tree
// If ns is not provided,  meaning candidate pods can come from all namespaces
// visitedPods will eliminate duplicate pods when execute overlapping queries,
// e.g. query by namespaces and query by fieldSelectors
func gatherPodLogs(kubeClient kubernetes.Interface, ns string, opts metav1.ListOptions, cfg *config.Config,
	visitedPods map[string]struct{},
) error {
	// 1 - Collect the list of pods
	podlist, err := kubeClient.CoreV1().Pods(ns).List(context.TODO(), opts)
	if err != nil {
		return errors.WithStack(err)
	}

	podLogOptions := getPodLogOptions(cfg)

	// 2 - Foreach pod, dump each of its containers' logs in a tree in the following location:
	//   pods/:podname/logs/:containername.txt
	for _, pod := range podlist.Items {
		if _, ok := visitedPods[string(pod.UID)]; ok {
			logrus.
				WithField("pod.UID", pod.UID).WithField("pod.Name", pod.Name).
				Tracef("Skipping pod pod logs since we have already visited it before")
			continue
		}
		visitedPods[string(pod.UID)] = struct{}{}

		if pod.Status.Phase == v1.PodFailed && pod.Status.Reason == "Evicted" {
			logrus.WithField("podName", pod.Name).Trace("Skipping evicted pod.")
			continue
		}
		for _, container := range pod.Spec.Containers {
			podLogOptions.Container = container.Name
			body, err := kubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions).DoRaw(context.TODO())
			if err != nil {
				return errors.WithStack(err)
			}
			outdir := path.Join(cfg.QueryOutputDir(), PodLogsLocation, pod.Namespace, pod.Name, "logs")
			if err = os.MkdirAll(outdir, 0o755); err != nil {
				return errors.WithStack(err)
			}

			outfile := path.Join(outdir, container.Name) + ".txt"
			if err = os.WriteFile(outfile, body, 0o644); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	return nil
}

// getNodeEndpoint returns the response from pinging a node endpoint
func getNodeEndpoint(client rest.Interface, nodeName, endpoint string) (rest.Result, error) {
	// TODO(chuckha) make this timeout configurable
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(30*time.Second))
	defer cancel()
	req := client.
		Get().
		Resource("nodes").
		Name(nodeName).
		SubResource("proxy").
		Suffix(endpoint)

	result := req.Do(ctx)
	if result.Error() != nil {
		logrus.Warningf("Could not get %v endpoint for node %v: %v", endpoint, nodeName, result.Error())
	}
	return result, result.Error()
}

// gatherNodeData collects non-resource information about a node through the
// kubernetes API.  That is, its `healthz` and `configz` endpoints, which are
// not "resources" per se, although they are accessible through the apiserver.
func gatherNodeData(nodeNames []string, restclient rest.Interface, cfg *config.Config) error {
	logrus.Info("Collecting Node Configuration and Health...")

	for _, name := range nodeNames {
		// Create the output for each node
		out := path.Join(cfg.QueryOutputDir(), HostsLocation, name)
		logrus.Infof("Creating host results for %v under %v\n", name, out)
		if err := os.MkdirAll(out, 0o755); err != nil {
			return err
		}

		_, err := timedObjectQuery(out, "configz.json", func() (interface{}, error) {
			data := make(map[string]interface{})
			result, err := getNodeEndpoint(restclient, name, "configz")
			if err != nil {
				return data, err
			}

			resultBytes, err := result.Raw()
			if err != nil {
				return data, err
			}
			err = json.Unmarshal(resultBytes, &data)
			return data, err
		})
		if err != nil {
			return err
		}

		_, err = timedObjectQuery(out, "healthz.json", func() (interface{}, error) {
			data := make(map[string]interface{})
			result, err := getNodeEndpoint(restclient, name, "healthz")
			if err != nil {
				return data, err
			}
			var healthstatus int
			result.StatusCode(&healthstatus)
			data["status"] = healthstatus
			return data, nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// Targeted namespaces will be specified by cfg.Limits.PodLogs.Namespaces OR cfg.Limits.PodLogs.SonobuoyNamespace.
func getPodLogNamespaceFilter(cfg *config.Config) string {
	nsfilter := cfg.Limits.PodLogs.Namespaces

	if cfg.Limits.PodLogs.SonobuoyNamespace != nil && *cfg.Limits.PodLogs.SonobuoyNamespace {
		if len(nsfilter) > 0 {
			nsfilter = fmt.Sprintf("%s|%s", nsfilter, cfg.Namespace)
		} else {
			nsfilter = cfg.Namespace
		}
	}
	return nsfilter
}

// SerializeObj will write out an object
func SerializeObj(obj interface{}, outpath string, file string) error {
	var err error
	if err = os.MkdirAll(outpath, 0o755); err != nil {
		return errors.WithStack(err)
	}

	var b []byte
	switch t := obj.(type) {
	case *unstructured.UnstructuredList:
		for _, v := range t.Items {
			v.SetManagedFields(nil)
		}
		b, err = t.MarshalJSON()
	default:
		b, err = json.Marshal(obj)
	}
	if err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.WriteFile(filepath.Join(outpath, file), b, 0o644))
}
