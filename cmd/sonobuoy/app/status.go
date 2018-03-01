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
	"os"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	ops "github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin/aggregation"
)

var (
	statusNamespace  string
	statusKubeconfig Kubeconfig
)

func init() {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Gets a summarizes status of a sonobuoy run",
		Run:   getStatus,
		Args:  cobra.ExactArgs(0),
	}

	AddNamespaceFlag(&statusNamespace, cmd)
	AddKubeconfigFlag(&statusKubeconfig, cmd)

	RootCmd.AddCommand(cmd)
}

// TODO (timothysc) summarize and aggregate daemonset-plugins by status done (24) running (24)
// also --show-all
func getStatus(cmd *cobra.Command, args []string) {
	config, err := statusKubeconfig.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get kubernetes config"))
		os.Exit(1)
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't initialise kubernete client"))
		os.Exit(1)
	}

	status, err := ops.NewSonobuoyClient().GetStatus(statusNamespace, client)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}

	tw := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', tabwriter.AlignRight)

	fmt.Fprintf(tw, "PLUGIN\tNODE\tSTATUS\n")
	for _, pluginStatus := range status.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", pluginStatus.Plugin, pluginStatus.Node, pluginStatus.Status)
	}

	if err := tw.Flush(); err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't write status out"))
		os.Exit(1)
	}
	fmt.Printf("\n%s\n", humanReadableStatus(status.Status))
}

func humanReadableStatus(str string) string {
	switch str {
	case aggregation.RunningStatus:
		return "Sonobuoy is still running. Runs can take up to 60 minutes."
	case aggregation.FailedStatus:
		return "Sonobuoy has failed. You can see what happened with `sonobuoy logs`."
	case aggregation.CompleteStatus:
		return "Sonobuoy has completed. Use `sonobuoy retrieve` to get results."
	default:
		return fmt.Sprintf("Sonobuoy is in unknown state %q. Please report a bug at github.com/heptio/sonobuoy", str)
	}
}
