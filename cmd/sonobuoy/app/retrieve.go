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
	"sync"

	"github.com/heptio/sonobuoy/pkg/client"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	prefix        = filepath.Join("tmp", "sonobuoy")
	defaultOutDir = "."
)

var (
	cpKubecfg   Kubeconfig
	cpNamespace string
)

func init() {
	cmd := &cobra.Command{
		Use:   "retrieve [path]",
		Short: "Retrieves the results of a sonobuoy run to a specified path",
		Run:   retrieveResults,
		Args:  cobra.MaximumNArgs(1),
	}

	AddKubeconfigFlag(&cpKubecfg, cmd.Flags())
	AddNamespaceFlag(&cpNamespace, cmd.Flags())

	RootCmd.AddCommand(cmd)
}

// TODO (timothysc) abstract retrieve details into a lower level function.
func retrieveResults(cmd *cobra.Command, args []string) {
	namespace, err := cmd.Flags().GetString("namespace")
	if err != nil {
		errlog.LogError(fmt.Errorf("failed to get namespace flag: %v", err))
		os.Exit(1)
	}

	outDir := defaultOutDir
	if len(args) > 0 {
		outDir = args[0]
	}

	restConfig, err := cpKubecfg.Get()
	if err != nil {
		errlog.LogError(fmt.Errorf("failed to get kubernetes client: %v", err))
		os.Exit(1)
	}
	sbc, err := client.NewSonobuoyClient(restConfig)
	if err != nil {
		errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
		os.Exit(1)
	}

	// TODO(chuckha) try to catch some errors and present user friendly messages.
	// Setup error channel and synchronization so that all errors get reported before exiting.
	errc := make(chan error)
	errcount := 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range errc {
			errcount++
			errlog.LogError(err)
		}
	}()

	cfg := &client.RetrieveConfig{
		Namespace: namespace,
		CmdErr:    os.Stderr,
		Errc:      errc,
	}

	// Get a reader that contains the tar output of the results directory.
	reader := sbc.RetrieveResults(cfg)

	// RetrieveResults bailed early and will report an error.
	if reader == nil {
		close(errc)
		wg.Wait()
		os.Exit(1)
	}

	// Extract the tar output into a local directory under the prefix.
	err = client.UntarAll(reader, outDir, prefix)
	if err != nil {
		close(errc)
		wg.Wait()
		errlog.LogError(fmt.Errorf("error untarring output: %v", err))
		os.Exit(1)
	}

	// Everything has been written from the reader which means we're done.
	close(errc)
	wg.Wait()

	if errcount != 0 {
		os.Exit(1)
	}
}
