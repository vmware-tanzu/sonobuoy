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

package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"

	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
)

type statusFlags struct {
	namespace string
	kubecfg   Kubeconfig
	showAll   bool
	json      bool
}

type pluginSummaries []pluginSummary

type pluginSummary struct {
	plugin string
	status string
	result string
	count  int
}

// For sort.Interface
func (p pluginSummaries) Len() int { return len(p) }
func (p pluginSummaries) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p pluginSummaries) Less(i, j int) bool {
	pi, pj := p[i], p[j]
	if pi.plugin == pj.plugin {
		return pi.status < pj.status
	}
	return pi.plugin < pj.plugin
}

func NewCmdStatus() *cobra.Command {
	var f statusFlags
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Gets a summarized status of a sonobuoy run",
		Run:   getStatus(&f),
		Args:  cobra.ExactArgs(0),
	}
	flags := cmd.Flags()

	AddNamespaceFlag(&f.namespace, flags)
	AddKubeconfigFlag(&f.kubecfg, flags)
	flags.BoolVar(
		&f.showAll, "show-all", false,
		"Don't summarize plugin statuses, show results for each node",
	)
	flags.BoolVar(
		&f.json, "json", false,
		"Print the status object as json",
	)

	return cmd
}

// TODO (timothysc) summarize and aggregate daemonset-plugins by status done (24) running (24)
// also --show-all
func getStatus(f *statusFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		sbc, err := getSonobuoyClientFromKubecfg(f.kubecfg)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		status, pod, err := sbc.GetStatusPod(&client.StatusConfig{
			Namespace: f.namespace,
		})
		switch {
		case err != nil && pod == nil:
			err = errors.Wrap(err, "error checking sonobuoy status")
		case err != nil:
			err = printPodInfo(os.Stdout, pod)
		case f.showAll:
			err = printAll(os.Stdout, status)
		case f.json:
			err = printJSON(os.Stdout, status)
		default:
			err = printSummary(os.Stdout, status)
		}

		if err != nil {
			errlog.LogError(err)
			os.Exit(1)
		}
		os.Exit(exitCode(status))
	}
}

func exitCode(status *aggregation.Status) int {
	// Allow status==nil to be non-error path. Explicit errors have been handled
	// before this and we are only supposed to be erroring here if we can tell
	// the status is a failure.
	if status == nil {
		return 0
	}
	switch status.Status {
	case aggregation.FailedStatus:
		return 1
	default:
		return 0
	}
}

func humanReadableStatus(str string) string {
	switch str {
	case aggregation.RunningStatus:
		return "Sonobuoy is still running. Runs can take 60 minutes or more depending on cluster and plugin configuration."
	case aggregation.FailedStatus:
		return "Sonobuoy has failed. You can see what happened with `sonobuoy logs`."
	case aggregation.CompleteStatus:
		return "Sonobuoy has completed. Use `sonobuoy retrieve` to get results."
	case aggregation.PostProcessingStatus:
		return "Sonobuoy plugins have completed. Preparing results for download."
	default:
		return fmt.Sprintf("Sonobuoy is in unknown state %q. Please report a bug at github.com/vmware-tanzu/sonobuoy", str)
	}
}

func printPodInfo(w io.Writer, pod *corev1.Pod) error {
	switch pod.Status.Phase {
	case (corev1.PodRunning):
		fmt.Fprintf(w, "The Sonobuoy aggregator is running but is unable to obtain the plugin status information.\n\nIf this state persists, use kubectl to debug further.\n")
	case (corev1.PodPending):
		switch {
		case len(pod.Status.ContainerStatuses) > 0 &&
			(pod.Status.ContainerStatuses[0].State.Waiting.Reason == "ErrImagePull" ||
				pod.Status.ContainerStatuses[0].State.Waiting.Reason == "ImagePullBackOff"):
			fmt.Fprintf(w, "The Sonobuoy aggregator is in the 'Pending' state and the container seems unable to pull the image.\n\nPod has the attached message: %v\n", pod.Status.ContainerStatuses[0].State.Waiting.Message)
			return fmt.Errorf("unable to pull sonobuoy image: %v", pod.Status.ContainerStatuses[0].State.Waiting.Message)
		default:
			fmt.Fprintf(w, "The Sonobuoy aggregator is in the 'Pending' state. This is normal as the pod is created and begins to run, but if this state persists, use kubectl to debug further.\n")
		}
	case (corev1.PodSucceeded):
		fmt.Fprintf(w, "The Sonobuoy aggregator ran successfully but has stopped. If you didn't intend for this to happen, submit a bug to github.com/vmware-tanzu/sonobuoy/issues.\n")
	case (corev1.PodFailed):
		fmt.Fprintf(w, "The Sonobuoy aggregator is in the 'Failed' state. Use kubectl to debug further.\n")
		return fmt.Errorf("sonobuoy aggregator is in the 'Failed' state")
	case (corev1.PodUnknown):
		fmt.Fprintf(w, "The Sonobuoy aggregator is in the 'Unknown' state. Use kubectl to debug further.\n")
		return fmt.Errorf("sonobuoy aggregator is in the 'Unknown' state")
	default:
		fmt.Println(pod.Status.String())
	}
	return nil
}

func printJSON(w io.Writer, status *aggregation.Status) error {
	enc := json.NewEncoder(w)
	return enc.Encode(status)
}

func printAll(w io.Writer, status *aggregation.Status) error {
	tw := defaultTabWriter(w)

	fmt.Fprintf(tw, "PLUGIN\tNODE\tSTATUS\tRESULT\t\n")
	for _, pluginStatus := range status.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t\n", pluginStatus.Plugin, pluginStatus.Node, pluginStatus.Status, pluginStatus.ResultStatus)
	}

	if err := tw.Flush(); err != nil {
		return errors.Wrap(err, "couldn't write status out")
	}

	fmt.Fprintf(w, "\n%s\n", humanReadableStatus(status.Status))
	return nil
}

func printSummary(w io.Writer, status *aggregation.Status) error {
	tw := defaultTabWriter(w)
	totals := map[string]map[string]int{}

	// Effectively making a pivot chart to count the unique combinations of status/result.
	statusResultKey := func(p aggregation.PluginStatus) string {
		return p.Status + ":" + p.ResultStatus
	}

	for _, pStatus := range status.Plugins {
		if _, ok := totals[pStatus.Plugin]; !ok {
			totals[pStatus.Plugin] = make(map[string]int)
		}
		totals[pStatus.Plugin][statusResultKey(pStatus)]++
	}

	// Sort everything nicely
	summaries := make(pluginSummaries, 0)
	for pluginName, pluginStats := range totals {
		for statusAndResult, count := range pluginStats {
			summaries = append(summaries, pluginSummary{
				plugin: pluginName,
				status: strings.Split(statusAndResult, ":")[0],
				result: strings.Split(statusAndResult, ":")[1],
				count:  count,
			})

		}
	}
	sort.Sort(summaries)
	fmt.Fprintf(tw, "PLUGIN\tSTATUS\tRESULT\tCOUNT\t\n")
	for _, summary := range summaries {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t\n", summary.plugin, summary.status, summary.result, summary.count)
	}

	if err := tw.Flush(); err != nil {
		return errors.Wrap(err, "couldn't write status out")
	}

	fmt.Fprintf(w, "\n%s\n", humanReadableStatus(status.Status))
	return nil
}

func defaultTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 2, 3, ' ', tabwriter.AlignRight)
}
