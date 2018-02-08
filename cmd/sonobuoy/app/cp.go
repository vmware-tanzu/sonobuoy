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
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var path string

func init() {
	cmd := &cobra.Command{
		Use:   "cp",
		Short: "Copies the results to a specified path",
		Run:   copyResults,
	}
	cmd.PersistentFlags().StringVar(
		&path, "path", "./",
		"TBD: location to output",
	)
	RootCmd.AddCommand(cmd)
}

func copyResults(cmd *cobra.Command, args []string) {
	f := util.NewClientAccessFactory(nil)
	cfg, err := f.ClientConfig()
	if err != nil {
		errlog.LogError(fmt.Errorf("could not get cfg: %v", err))
		os.Exit(1)
	}
	clientset, err := f.ClientSet()
	if err != nil {
		errlog.LogError(fmt.Errorf("could not get clientset: %v", err))
		os.Exit(1)
	}
	src := ops.FileSpec{
		PodNamespace: "heptio-sonobuoy",
		PodName:      "sonobuoy",
		File:         "/tmp/sonobuoy",
	}
	dst := ops.FileSpec{
		File: "./archive",
	}
	errc := make(chan error)
	go ops.CopyResults(cfg, clientset, os.Stderr, src, dst, errc)
	errorCount := 0
	for err := range errc {
		errorCount++
		errlog.LogError(fmt.Errorf("error during coyping: %v", err))
	}
	if errorCount > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}
