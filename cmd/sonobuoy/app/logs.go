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

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
)

const (
	bufSize = 2048
)

var logConfig client.LogConfig
var logsKubecfg Kubeconfig

func NewCmdLogs() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Dumps the logs of the currently running sonobuoy containers for diagnostics",
		Run:   getLogs,
		Args:  cobra.ExactArgs(0),
	}

	cmd.Flags().BoolVarP(
		&logConfig.Follow, "follow", "f", false,
		"Specify if the logs should be streamed.",
	)
	logConfig.Out = os.Stdout
	AddKubeconfigFlag(&logsKubecfg, cmd.Flags())
	AddNamespaceFlag(&logConfig.Namespace, cmd.Flags())
	return cmd
}

func getLogs(cmd *cobra.Command, args []string) {
	cfg, err := logsKubecfg.Get()
	if err != nil {
		errlog.LogError(errors.Wrap(err, "failed to get rest config"))
		os.Exit(1)
	}
	sbc, err := getSonobuoyClient(cfg)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}
	logreader, err := sbc.LogReader(&logConfig)
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
