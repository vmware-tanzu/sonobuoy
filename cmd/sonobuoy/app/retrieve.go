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
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/util/exec"
)

const (
	defaultOutDir = "."
)

type retrieveFlags struct {
	namespace      string
	kubecfg        Kubeconfig
	extract        bool
	outputLocation string
}

func NewCmdRetrieve() *cobra.Command {
	rcvFlags := retrieveFlags{}
	cmd := &cobra.Command{
		Use:   "retrieve [path]",
		Short: "Retrieves the results of a sonobuoy run to a specified path",
		Run:   retrieveResultsCmd(&rcvFlags),
		Args:  cobra.MaximumNArgs(1),
	}

	AddKubeconfigFlag(&rcvFlags.kubecfg, cmd.Flags())
	AddNamespaceFlag(&rcvFlags.namespace, cmd.Flags())
	AddExtractFlag(&rcvFlags.extract, cmd.Flags())
	return cmd
}

func retrieveResultsCmd(opts *retrieveFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		opts.outputLocation = defaultOutDir
		if len(args) > 0 {
			opts.outputLocation = args[0]
		}

		sbc, err := getSonobuoyClientFromKubecfg(opts.kubecfg)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		// Get a reader that contains the tar output of the results directory.
		reader, ec, err := sbc.RetrieveResults(&client.RetrieveConfig{Namespace: opts.namespace})
		if err != nil {
			errlog.LogError(err)
			os.Exit(1)
		}

		err = retrieveResults(*opts, reader, ec)
		if _, ok := err.(exec.CodeExitError); ok {
			fmt.Fprintln(os.Stderr, "Results not ready yet. Check `sonobuoy status` for status.")
			os.Exit(1)
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "error retrieving results: %v\n", err)
			os.Exit(2)
		}
	}
}

func retrieveResults(opts retrieveFlags, r io.Reader, ec <-chan error) error {
	eg := &errgroup.Group{}
	eg.Go(func() error { return <-ec })
	eg.Go(func() error {
		// This untars the request itself, which is tar'd as just part of the API request, not the sonobuoy logic.
		filesCreated, err := client.UntarAll(r, opts.outputLocation, "")
		if err != nil {
			return err
		}
		if !opts.extract {
			// Only print the filename if not extracting. Allows capturing the filename for scripting.
			for _, name := range filesCreated {
				fmt.Println(name)
			}
			return nil
		} else {
			for _, filename := range filesCreated {
				err := client.UntarFile(filename, opts.outputLocation, true)
				if err != nil {
					// Just log errors if it is just not cleaning up the file.
					re, ok := err.(*client.DeletionError)
					if ok {
						errlog.LogError(re)
					} else {
						return err
					}
				}
			}

			return err
		}
	})

	return eg.Wait()
}
