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

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

var deleteFlags struct {
	kubeconfig Kubeconfig
	namespace  string
}

func init() {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "cleans up a sonobuoy run",
		Run:   deleteSonobuoyRun,
		Args:  cobra.ExactArgs(0),
	}

	AddKubeconfigFlag(&deleteFlags.kubeconfig, cmd)
	AddNamespaceFlag(&deleteFlags.namespace, cmd)

	RootCmd.AddCommand(cmd)
}

func deleteSonobuoyRun(cmd *cobra.Command, args []string) {
	cfg, err := deleteFlags.kubeconfig.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get kubernetes config"))
		os.Exit(1)
	}

	kubeclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "couldn't get kubernetes client"))
		os.Exit(1)
	}

	if err := client.NewSonobuoyClient().Delete(deleteFlags.namespace, kubeclient); err != nil {
		errlog.LogError(errors.Wrap(err, "failed to delete sonobuoy resources"))
		os.Exit(1)
	}

}
