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
	"path/filepath"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	prefix        = filepath.Join("tmp", "sonobuoy")
	defaultOutDir = "."
)

type receiveFlags struct {
	namespace string
	kubecfg   Kubeconfig
}

var rcvFlags receiveFlags

func init() {
	cmd := &cobra.Command{
		Use:   "retrieve [path]",
		Short: "Retrieves the results of a sonobuoy run to a specified path",
		Run:   retrieveResults,
		Args:  cobra.MaximumNArgs(1),
	}

	AddKubeconfigFlag(&rcvFlags.kubecfg, cmd.Flags())
	AddNamespaceFlag(&rcvFlags.namespace, cmd.Flags())

	RootCmd.AddCommand(cmd)
}

func retrieveResults(cmd *cobra.Command, args []string) {
	outDir := defaultOutDir
	if len(args) > 0 {
		outDir = args[0]
	}

	restConfig, err := rcvFlags.kubecfg.Get()
	if err != nil {
		errlog.LogError(fmt.Errorf("failed to get kubernetes client: %v", err))
		os.Exit(1)
	}
	sbc, err := client.NewSonobuoyClient(restConfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	// Get a reader that contains the tar output of the results directory.
	reader, err := sbc.RetrieveResults(&client.RetrieveConfig{Namespace: rcvFlags.namespace})
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	// Extract the tar output into a local directory under the prefix.
	err = client.UntarAll(reader, outDir, prefix)
	if err != nil {
		os.Exit(1)
	}
}
