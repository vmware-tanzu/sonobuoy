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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/dynamic"
	"github.com/heptio/sonobuoy/pkg/errlog"

	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/viniciuschiele/tarx"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	pluginaggregation "github.com/heptio/sonobuoy/pkg/plugin/aggregation"
)

// Run is the main entrypoint for discovery.
func Run(restConf *rest.Config, cfg *config.Config) (errCount int) {
	// Adjust QPS/Burst so that the queries execute as quickly as possible.
	restConf.QPS = float32(cfg.QPS)
	restConf.Burst = cfg.Burst

	apiHelper, err := dynamic.NewAPIHelperFromRESTConfig(restConf)
	if err != nil {
		errlog.LogError(err)
		return errCount + 1
	}

	kubeClient, err := kubernetes.NewForConfig(restConf)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create kubernetes client"))
		return errCount + 1
	}

	t := time.Now()

	// 1. Create the directory which will store the results, including the
	// `meta` directory inside it (which we always need regardless of
	// config)
	outpath := path.Join(cfg.ResultsDir, cfg.UUID)
	metapath := path.Join(outpath, MetaLocation)
	err = os.MkdirAll(metapath, 0755)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create directory to store results"))
		return errCount + 1
	}

	// Write logs to the configured results location. All log levels
	// should write to the same log file
	pathmap := make(lfshook.PathMap)
	logfile := path.Join(metapath, "run.log")
	for _, level := range logrus.AllLevels {
		pathmap[level] = logfile
	}

	hook := lfshook.NewHook(pathmap, &logrus.JSONFormatter{})

	logrus.AddHook(hook)

	// Unset all hooks as we exit the Run function
	defer func() {
		logrus.StandardLogger().Hooks = make(logrus.LevelHooks)
	}()
	// closure used to collect and report errors.
	trackErrorsFor := func(action string) func(error) {
		return func(err error) {
			if err != nil {
				errCount++
				errlog.LogError(errors.Wrapf(err, "error %v", action))
			}
		}
	}

	// Determine sonobuoy pod name
	pluginaggregation.SetStatusPodName(kubeClient, cfg.Namespace)

	// Set initial annotation stating the pod is running. Ensures the annotation
	// exists sooner for user/polling consumption and prevents issues were we try
	// to patch a non-existant status later.
	trackErrorsFor("setting initial pod status")(
		setStatus(kubeClient, cfg.Namespace,
			&pluginaggregation.Status{
				Status: pluginaggregation.RunningStatus,
			}),
	)

	// 2. Get the list of namespaces and apply the regex filter on the namespace
	logrus.Infof("Filtering namespaces based on the following regex:%s",  cfg.Filters.Namespaces)
	nslist, err := FilterNamespaces(kubeClient,  cfg.Filters.Namespaces)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not filter namespaces"))
		return errCount + 1
	}

	// 3. Dump the config.json we used to run our test
	if blob, err := json.Marshal(cfg); err == nil {
		if err = ioutil.WriteFile(path.Join(metapath, "config.json"), blob, 0644); err != nil {
			errlog.LogError(errors.Wrap(err, "could not write config.json file"))
			return errCount + 1
		}
	}

	// 4. Run the plugin aggregator
	trackErrorsFor("running plugins")(
		pluginaggregation.Run(kubeClient, cfg.LoadedPlugins, cfg.Aggregation, cfg.Namespace, outpath),
	)

	// 5. Run the queries
	recorder := NewQueryRecorder()
	clusterResources, nsResources, err := getAllFilteredResources(apiHelper, cfg.Resources)

	trackErrorsFor("querying cluster resources")(
		QueryHostData(kubeClient, recorder, cfg),
	)

	trackErrorsFor("querying cluster resources")(
		QueryResources(apiHelper, recorder, clusterResources, nil, cfg),
	)

	trackErrorsFor("querying server info")(
		QueryServerData(kubeClient, recorder, cfg),
	)

	for _, ns := range nslist {
		trackErrorsFor("querying resources under namespace " + ns)(
			QueryResources(apiHelper, recorder, nsResources, &ns, cfg),
		)
	}

	// query pod logs
	if cfg.Resources == nil || sliceContains(cfg.Resources, "podlogs") {

		// Eliminate duplicate pods when query by namespaces and query by fieldSelectors
		visitedPods := make(map[string]struct{})

		nsFilter := getPodLogNamespaceFilter(cfg)
		if len(nsFilter) > 0 {
			nsListLogs, _ := FilterNamespaces(kubeClient, nsFilter)
			for _, ns := range nsListLogs {
				trackErrorsFor("querying pod logs under namespace " + ns)(
					QueryPodLogs(kubeClient, recorder, ns, cfg, visitedPods),
				)
			}
		}
		trackErrorsFor("querying pod logs by field selectors")(
			QueryPodLogs(kubeClient, recorder, "", cfg, visitedPods),
		)
	} else {
		logrus.Infof("podlogs not specified in non-nil Resources, skipping getting podlogs")
	}

	// 6. Dump the query times
	trackErrorsFor("recording query times")(
		recorder.DumpQueryData(path.Join(metapath, "query-time.json")),
	)

	// 7. Clean up after the plugins
	pluginaggregation.Cleanup(kubeClient, cfg.LoadedPlugins)

	// 8. tarball up results YYYYMMDDHHMM_sonobuoy_UID.tar.gz
	tb := cfg.ResultsDir + "/" + t.Format("200601021504") + "_sonobuoy_" + cfg.UUID + ".tar.gz"
	err = tarx.Compress(tb, outpath, &tarx.CompressOptions{Compression: tarx.Gzip})
	if err == nil {
		defer os.RemoveAll(outpath)
	}
	trackErrorsFor("assembling results tarball")(err)

	// 9. Mark final annotation stating the results are available and status is completed.
	trackErrorsFor("updating pod status")(
		updateStatus(kubeClient, cfg.Namespace, pluginaggregation.CompleteStatus),
	)

	logrus.Infof("Results available at %v", tb)

	return errCount
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

// updateStatus changes the summary status of the sonobuoy pod in order to
// effect the finalized status the user sees. This does not change the status
// of individual plugins.
func updateStatus(client kubernetes.Interface, namespace string, status string) error {
	podStatus, err := pluginaggregation.GetStatus(client, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get the existing status")
	}

	// Update status
	podStatus.Status = status
	return setStatus(client, namespace, podStatus)
}

// setStatus sets the status on the pod via an annotation. It will overwrite the
// existing status.
func setStatus(client kubernetes.Interface, namespace string, status *pluginaggregation.Status) error {
	// Marshal back into json, inject into the patch, then serialize again.
	statusBytes, err := json.Marshal(status)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the status")
	}

	patch := pluginaggregation.GetPatch(string(statusBytes))
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return errors.Wrap(err, "failed to marshal the patch")
	}

	_, err = client.CoreV1().Pods(namespace).Patch(pluginaggregation.StatusPodName, types.MergePatchType, patchBytes)
	return err
}
