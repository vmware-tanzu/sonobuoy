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
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/args"
	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

var statusOpts struct {
	namespace args.Namespace
	config    args.Kubeconfig
}

func init() {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Gets a summarizes status of a sonobuoy run",
		Run:   getStatus,
		Args:  cobra.ExactArgs(0),
	}

	args.AddNamespaceFlag(&statusOpts.namespace, cmd)
	args.AddKubeconfigFlag(&statusOpts.config, cmd)

	RootCmd.AddCommand(cmd)
}

func getStatus(cmd *cobra.Command, args []string) {
	config, err := statusOpts.config.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get kubernetes config"))
		os.Exit(1)
	}
	namespace := statusOpts.namespace.Get()
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't initialise kubernete client"))
		os.Exit(1)
	}

	status, err := ops.GetStatus(namespace, client)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to run sonobuoy"))
		os.Exit(1)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"plugin", "node", "status"})
	for _, pluginStatus := range status.Plugins {
		table.Append([]string{pluginStatus.Plugin, pluginStatus.Node, pluginStatus.Status})
	}
	table.SetFooter([]string{"", "", status.Status})
	table.Render()

	os.Exit(0)
}
