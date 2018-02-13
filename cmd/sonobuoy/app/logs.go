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

	ops "github.com/heptio/sonobuoy/cmd/sonobuoy/app/operations"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Dumps the logs of the currently running sonobuoy containers for diagnostics",
		Run:   getLogs,
	}
	cmd.Flags().BoolP("follow", "f", false, "Specify if the logs should be streamed.")
	RootCmd.AddCommand(cmd)
}

func getLogs(cmd *cobra.Command, args []string) {
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		errlog.LogError(errors.Wrap(err, "error getting follow flag"))
		os.Exit(1)
	}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		errlog.LogError(fmt.Errorf("failed to get rest config: %v", err))
		os.Exit(1)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		errlog.LogError(fmt.Errorf("failed to get kubernetes client: %v", err))
		os.Exit(1)
	}
	if err := ops.GetLogs(kubeClient, config.DefaultPluginNamespace, follow); err != nil {
		errlog.LogError(errors.Wrap(err, "error attempting to get sonobuoy logs"))
		os.Exit(1)
	}
}
