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
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"github.com/heptio/sonobuoy/pkg/errlog"
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

	status, err := ops.GetStatus(statusNamespace, client)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}

	tw := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', tabwriter.AlignRight)

	fmt.Fprintf(tw, "PLUGIN\tNODE\tSTATUS\n")
	for _, pluginStatus := range status.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", pluginStatus.Plugin, pluginStatus.Node, pluginStatus.Status)
	}
	fmt.Fprintf(tw, "\t\t%s\n", strings.ToUpper(status.Status))
	if err := tw.Flush(); err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't write status out"))
		os.Exit(1)
	}

	os.Exit(0)
}
