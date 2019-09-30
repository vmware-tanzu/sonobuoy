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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

const (
	bufSize = 2048
)

var logFlags struct {
	namespace  string
	follow     bool
	plugin     string
	kubeconfig Kubeconfig
}

func NewCmdLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Dumps the logs of the currently running sonobuoy containers for diagnostics",
		Run:   getLogs,
		Args:  cobra.ExactArgs(0),
	}

	cmd.Flags().BoolVarP(
		&logFlags.follow, "follow", "f", false,
		"Specify if the logs should be streamed.",
	)
	AddKubeconfigFlag(&logFlags.kubeconfig, cmd.Flags())
	AddNamespaceFlag(&logFlags.namespace, cmd.Flags())
	cmd.Flags().StringVarP(&logFlags.plugin, pluginFlag, "p", "", "Show logs for a specific plugin")
	return cmd
}

func getLogs(cmd *cobra.Command, args []string) {
	sbc, err := getSonobuoyClientFromKubecfg(logFlags.kubeconfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	logConfig := client.NewLogConfig()
	logConfig.Namespace = logFlags.namespace
	logConfig.Follow = logFlags.follow
	logConfig.Plugin = logFlags.plugin

	logreader, err := sbc.LogReader(logConfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not build a log reader"))
		os.Exit(1)
	}
	b := make([]byte, bufSize)
	for {
		n, err := logreader.Read(b)
		if err != nil && err != io.EOF {
			errlog.LogError(errors.Wrap(err, "error reading logs"))
			os.Exit(1)
		}
		fmt.Fprint(logConfig.Out, string(b[:n]))
		if err == io.EOF {
			return
		}
	}
}
