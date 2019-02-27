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
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
)

var statusFlags struct {
	namespace string
	kubecfg   Kubeconfig
	showAll   bool
}

func NewCmdStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Gets a summarized status of a sonobuoy run",
		Run:   getStatus,
		Args:  cobra.ExactArgs(0),
	}
	flags := cmd.Flags()

	AddNamespaceFlag(&statusFlags.namespace, flags)
	AddKubeconfigFlag(&statusFlags.kubecfg, flags)
	flags.BoolVar(
		&statusFlags.showAll, "show-all", false,
		"Don't summarize plugin statuses, show all individually",
	)

	return cmd
}

// TODO (timothysc) summarize and aggregate daemonset-plugins by status done (24) running (24)
// also --show-all
func getStatus(cmd *cobra.Command, args []string) {
	cfg, err := statusFlags.kubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get kubernetes config"))
		os.Exit(1)
	}
	sbc, err := getSonobuoyClient(cfg)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	status, err := sbc.GetStatus(statusFlags.namespace)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}

	if statusFlags.showAll {
		err = printAll(os.Stdout, status)
	} else {
		err = printSummary(os.Stdout, status)
	}

	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
	os.Exit(exitCode(status))
}

func exitCode(status *aggregation.Status) int {
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
		return "Sonobuoy is still running. Runs can take up to 60 minutes."
	case aggregation.FailedStatus:
		return "Sonobuoy has failed. You can see what happened with `sonobuoy logs`."
	case aggregation.CompleteStatus:
		return "Sonobuoy has completed. Use `sonobuoy retrieve` to get results."
	case aggregation.PostProcessingStatus:
		return "Sonobuoy plugins have completed. Preparing results for download."
	default:
		return fmt.Sprintf("Sonobuoy is in unknown state %q. Please report a bug at github.com/heptio/sonobuoy", str)
	}
}

func printAll(w io.Writer, status *aggregation.Status) error {
	tw := tabwriter.NewWriter(w, 1, 8, 1, '\t', tabwriter.AlignRight)

	fmt.Fprintf(tw, "PLUGIN\tNODE\tSTATUS\n")
	for _, pluginStatus := range status.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", pluginStatus.Plugin, pluginStatus.Node, pluginStatus.Status)
	}

	if err := tw.Flush(); err != nil {
		return errors.Wrap(err, "couldn't write status out")
	}

	fmt.Fprintf(w, "\n%s\n", humanReadableStatus(status.Status))
	return nil
}

func printSummary(w io.Writer, status *aggregation.Status) error {
	tw := tabwriter.NewWriter(w, 1, 8, 1, '\t', tabwriter.AlignRight)
	totals := map[string]map[string]int{}
	for _, plugin := range status.Plugins {
		if _, ok := totals[plugin.Plugin]; !ok {
			totals[plugin.Plugin] = make(map[string]int)
		}
		totals[plugin.Plugin][plugin.Status]++
	}

	// sort everything nicely
	summaries := make(pluginSummaries, 0)
	for pluginName, pluginStats := range totals {
		for status, count := range pluginStats {
			summaries = append(summaries, pluginSummary{
				plugin: pluginName,
				status: status,
				count:  count,
			})
		}
	}
	sort.Sort(summaries)
	fmt.Fprintf(tw, "PLUGIN\tSTATUS\tCOUNT\n")
	for _, summary := range summaries {
		fmt.Fprintf(tw, "%s\t%s\t%d\n", summary.plugin, summary.status, summary.count)
	}

	if err := tw.Flush(); err != nil {
		return errors.Wrap(err, "couldn't write status out")
	}

	fmt.Fprintf(w, "\n%s\n", humanReadableStatus(status.Status))
	return nil
}

type pluginSummaries []pluginSummary

type pluginSummary struct {
	plugin string
	status string
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
