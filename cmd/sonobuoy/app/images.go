/*
Copyright 2019 Heptio Inc.

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

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/image"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var imagesflags imagesFlags

type imagesFlags struct {
	e2eRegistryConfig string
	plugin            string
	kubeconfig        Kubeconfig

	imagesflags *pflag.FlagSet
}

func ImagesFlagSet(cfg *imagesFlags) *pflag.FlagSet {
	flagset := pflag.NewFlagSet("images", pflag.ExitOnError)
	AddPluginFlag(&cfg.plugin, flagset)
	AddKubeconfigFlag(&cfg.kubeconfig, flagset)
	cfg.imagesflags = flagset
	return flagset
}

func NewCmdImages() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage images used in a plugin. Supported plugins are: 'e2e'",
		Run:   getImages,
		Args:  cobra.ExactArgs(0),
	}

	cmd.Flags().AddFlagSet(ImagesFlagSet(&imagesflags))
	return cmd
}

func getImages(cmd *cobra.Command, args []string) {

	switch imagesflags.plugin {
	case "e2e":

		cfg, err := imagesflags.kubeconfig.Get()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get REST client"))
			os.Exit(1)
		}

		sbc, err := getSonobuoyClient(cfg)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		version, err := sbc.Version()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get Sonobuoy client"))
			os.Exit(1)
		}

		// Get list of images that match the version
		registry, err := image.NewRegistryList("", version)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't init Registry List"))
			os.Exit(1)
		}

		images, err := registry.GetImageConfigs()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get images for version"))
			os.Exit(1)
		}

		for _, v := range images {
			fmt.Println(v.GetE2EImage())
		}
	default:
		errlog.LogError(errors.Errorf("Unsupported plugin: %v", imagesflags.plugin))
		os.Exit(1)
	}
}
