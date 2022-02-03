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
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	pluginaggregation "github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
	"github.com/vmware-tanzu/sonobuoy/pkg/tarball"

	"github.com/pkg/errors"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	pluginDefinitionFilename = "defintion.json"

	// MetaLocation is the place under which snapshot metadata (query times, config) is stored
	// Also stored in query.go
	MetaLocation = "meta"
)

type RunInfo struct {
	LoadedPlugins []string `json:"plugins,omitempty"`
}

// timeout is an interface to identify if an error is due to a timeout or not.
type timeout interface {
	Timeout() bool
}

// Run is the main entrypoint for discovery.
func Run(restConf *rest.Config, cfg *config.Config) (errCount int) {
	kubeClient, err := kubernetes.NewForConfig(restConf)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create kubernetes client"))
		return errCount + 1
	}

	t := time.Now()

	// 1. Create the directory which will store the results, including the
	// `meta` directory inside it (which we always need regardless of
	// config)
	outpath := filepath.Join(config.AggregatorResultsPath, cfg.UUID)
	metapath := filepath.Join(outpath, MetaLocation)
	err = os.MkdirAll(metapath, 0755)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create directory to store results"))
		return errCount + 1
	}

	// Write logs to the configured results location. All log levels
	// should write to the same log file
	pathmap := make(lfshook.PathMap)
	logfile := filepath.Join(metapath, "run.log")
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

	// Set initial annotation stating the pod is running. Ensures the annotation
	// exists sooner for user/polling consumption and prevents issues were we try
	// to patch a non-existant status later.
	trackErrorsFor("setting initial pod status")(
		setPodStatusAnnotation(kubeClient, cfg.Namespace,
			&pluginaggregation.Status{
				Status: pluginaggregation.RunningStatus,
			}),
	)

	// 3. Dump the config.json we used to run our test
	if blob, err := json.Marshal(cfg); err == nil {
		logrus.Trace("Recording the marshalled Sonobuoy config")
		if err = ioutil.WriteFile(filepath.Join(metapath, "config.json"), blob, 0644); err != nil {
			errlog.LogError(errors.Wrap(err, "could not write config.json file"))
			return errCount + 1
		}
	} else {
		errlog.LogError(errors.Wrap(err, "error marshalling Sonobuoy config"))
	}

	// runInfo is for dumping additional information to help enable processing of the resulting tarball.
	runInfo := RunInfo{
		LoadedPlugins: []string{},
	}

	// 4. Run the plugin aggregator. Save this error for clear logging later.
	runErr := pluginaggregation.Run(kubeClient, cfg.LoadedPlugins, cfg.Aggregation, cfg.ProgressUpdatesPort, cfg.ResultsDir, cfg.Namespace, outpath)
	trackErrorsFor("running plugins")(runErr)

	// Add a log line at the end of the run for clarity. Common problem in timeout situations is that
	// users do not find the timeout message in the middle of the run logs. Can't just add it with a `defer`
	// since we'd also like this to appear in the podlogs that get put into the tarball.
	if tErr, ok := runErr.(timeout); ok && tErr.Timeout() {
		logrus.Errorf("Timeout occurred when running plugins. Inspect logs further for details.")
	}

	logrus.Infof("Log lines after this point will not appear in the downloaded tarball.")

	// 7. Clean up after the plugins
	pluginaggregation.Cleanup(kubeClient, cfg.LoadedPlugins)

	// Postprocessing before we create the tarball.
	for _, p := range cfg.LoadedPlugins {
		logrus.WithField("plugin", p.GetName()).Trace("Post-processing")
		item, errs := results.PostProcessPlugin(p, outpath)
		for _, e := range errs {
			logrus.Errorf("Error processing plugin %v: %v", p.GetName(), e)
		}

		// Save results object regardless of errors; it is our best effort to understand the results.
		if err := results.SaveProcessedResults(p.GetName(), outpath, item); err != nil {
			logrus.Errorf("Unable to save results for plugin %v: %v", p.GetName(), err)
		}

		// Update the plugin status with this post-processed information.
		if err := updatePluginStatus(kubeClient, cfg.Namespace, p.GetName(), item); err != nil {
			logrus.Errorf("Failed to update status for plugin %v: %v", p.GetName(), err)
		}
	}

	// Saving plugin definitions in their respective folders for easy reference.
	for _, p := range cfg.LoadedPlugins {
		logrus.
			WithField("plugin", p.GetName()).
			Tracef("Saving plugin definition")
		runInfo.LoadedPlugins = append(runInfo.LoadedPlugins, p.GetName())
		trackErrorsFor("saving plugin info")(
			dumpPlugin(p, outpath),
		)
	}

	// Dump extra metadata that may be useful to postprocessors or analysis.
	blob, err := json.Marshal(runInfo)
	trackErrorsFor("marshalling run info")(err)
	if err == nil {
		err = ioutil.WriteFile(filepath.Join(metapath, results.InfoFile), blob, 0644)
		trackErrorsFor("saving" + results.InfoFile)(err)
	}

	// Run queries.
	trackErrorsFor("running queries")(
		QueryCluster(restConf, cfg),
	)

	// Add health metadata
	trackErrorsFor("adding health metadata")(
		SaveHealthSummary(outpath),
	)

	// 8. tarball up results YYYYMMDDHHMM_sonobuoy_UID.tar.gz
	filename := fmt.Sprintf("%v_sonobuoy_%v.tar.gz", t.Format("200601021504"), cfg.UUID)
	tb := filepath.Join(config.AggregatorResultsPath, filename)
	err = tarball.DirToTarball(outpath, tb, true)
	if err == nil {
		defer os.RemoveAll(outpath)
	}
	trackErrorsFor("assembling results tarball")(err)

	tarInfo, err := getFileInfo(tb)
	trackErrorsFor("recording tarball info")(err)

	// 9. Mark final annotation stating the results are available and status is completed.
	trackErrorsFor("updating pod status")(
		updateStatus(
			kubeClient,
			cfg.Namespace,
			pluginaggregation.CompleteStatus,
			&tarInfo,
		),
	)

	logrus.Infof("Results available at %v", tb)

	return errCount
}

func statusCounts(item *results.Item, startingCounts map[string]int) {
	if item == nil {
		return
	}

	if len(item.Items) > 0 {
		for _, v := range item.Items {
			statusCounts(&v, startingCounts)
		}
		return
	}
	startingCounts[item.Status]++
}

func getFileInfo(path string) (pluginaggregation.TarInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return pluginaggregation.TarInfo{}, err
	}

	f, err := os.Open(path)
	if err != nil {
		return pluginaggregation.TarInfo{}, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return pluginaggregation.TarInfo{}, err
	}

	return pluginaggregation.TarInfo{
		Name:      filepath.Base(path),
		Size:      fi.Size(),
		SHA256:    fmt.Sprintf("%x", h.Sum(nil)),
		CreatedAt: time.Now(),
	}, nil
}

// dumpPlugin will marshal the plugin to the appropriate location in the outputDir:
// plugins/<name>/definition.json. This makes the data more clear for any consumer
// looking at the tarball about what was.
func dumpPlugin(p plugin.Interface, outputDir string) error {
	b, err := json.Marshal(p)
	if err != nil {
		return errors.Wrapf(err, "encoding plugin %v definition to yaml", p.GetName())
	}

	err = ioutil.WriteFile(
		filepath.Join(outputDir, results.PluginsDir, p.GetName(), pluginDefinitionFilename),
		b,
		os.FileMode(0644),
	)
	return errors.Wrapf(err, "writing plugin %v definition to yaml", p.GetName())
}

// updateStatus changes the summary status of the sonobuoy pod in order to
// effect the finalized status the user sees. This does not change the
// status of individual plugins.
func updateStatus(client kubernetes.Interface, namespace string, status string, tarInfo *pluginaggregation.TarInfo) error {
	podStatus, _, err := pluginaggregation.GetStatus(client, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get the existing status")
	}

	// Update status
	podStatus.Status = status
	if tarInfo != nil {
		podStatus.Tarball = *tarInfo
	}
	return setPodStatusAnnotation(client, namespace, podStatus)
}

func updatePluginStatus(client kubernetes.Interface, namespace string, pluginName string, item results.Item) error {
	logrus.WithField("plugin", pluginName).Trace("Updating plugin status")
	podStatus, _, err := pluginaggregation.GetStatus(client, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get the existing status")
	}

	integrateResultsIntoStatus(podStatus, pluginName, &item)

	return setPodStatusAnnotation(client, namespace, podStatus)
}

func integrateResultsIntoStatus(podStatus *pluginaggregation.Status, pluginName string, item *results.Item) {
	for i := range podStatus.Plugins {
		if podStatus.Plugins[i].Plugin != pluginName {
			continue
		}
		var itemForNode *results.Item
		if podStatus.Plugins[i].Node == plugin.GlobalResult {
			itemForNode = item
		} else {
			itemForNode = item.GetSubTreeByName(podStatus.Plugins[i].Node)
		}

		if itemForNode == nil {
			logrus.WithField("plugin", pluginName).Trace("No results for node in this result")
			return
		}

		statusInfo := map[string]int{}
		statusCounts(itemForNode, statusInfo)
		podStatus.Plugins[i].ResultStatus = itemForNode.Status
		podStatus.Plugins[i].ResultStatusCounts = statusInfo
		logrus.
			WithField("plugin", pluginName).
			WithField("node", podStatus.Plugins[i].Node).
			Tracef("Updating with status %v", itemForNode.Status)
	}
}

// setPodStatusAnnotation sets the status on the pod via an annotation. It will overwrite the
// existing status.
func setPodStatusAnnotation(client kubernetes.Interface, namespace string, status *pluginaggregation.Status) error {
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

	// Determine sonobuoy pod name
	podName, err := pluginaggregation.GetAggregatorPodName(client, namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get the name of the aggregator pod to set the status on")
	}

	_, err = client.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	return err
}
