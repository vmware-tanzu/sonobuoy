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

	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/image"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var imagesflags imagesFlags

// Number times to retry docker commands before giving up
const (
	numDockerRetries     = 1
	defaultE2ERegistries = ""
)

type imagesFlags struct {
	e2eRegistryConfig string
	plugin            string
	kubeconfig        Kubeconfig
}

func NewCmdImages() *cobra.Command {
	// Main command
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage images used in a plugin. Supported plugins are: 'e2e'",
		Run:   listImages,
		Args:  cobra.ExactArgs(0),
	}

	AddKubeconfigFlag(&imagesflags.kubeconfig, cmd.Flags())
	AddPluginFlag(&imagesflags.plugin, cmd.Flags())

	// Pull command
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pulls images to local docker client for a specific plugin",
		Run:   pullImages,
		Args:  cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&imagesflags.e2eRegistryConfig, pullCmd.Flags())
	AddKubeconfigFlag(&imagesflags.kubeconfig, pullCmd.Flags())
	AddPluginFlag(&imagesflags.plugin, pullCmd.Flags())

	// Download command
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Saves downloaded images from local docker client to a tar file",
		Run:   downloadImages,
		Args:  cobra.ExactArgs(0),
	}
	AddKubeconfigFlag(&imagesflags.kubeconfig, downloadCmd.Flags())
	AddPluginFlag(&imagesflags.plugin, downloadCmd.Flags())

	// Push command
	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Pushes images to docker registry for a specific plugin",
		Run:   pushImages,
		Args:  cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&imagesflags.e2eRegistryConfig, pushCmd.Flags())
	AddKubeconfigFlag(&imagesflags.kubeconfig, pushCmd.Flags())
	AddPluginFlag(&imagesflags.plugin, pushCmd.Flags())
	pushCmd.MarkFlagRequired(e2eRegistryConfigFlag)

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes all images downloaded to local docker client",
		Run:   deleteImages,
		Args:  cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&imagesflags.e2eRegistryConfig, deleteCmd.Flags())
	AddKubeconfigFlag(&imagesflags.kubeconfig, deleteCmd.Flags())
	AddPluginFlag(&imagesflags.plugin, deleteCmd.Flags())

	cmd.AddCommand(pullCmd)
	cmd.AddCommand(pushCmd)
	cmd.AddCommand(downloadCmd)
	cmd.AddCommand(deleteCmd)

	return cmd
}

func listImages(cmd *cobra.Command, args []string) {

	switch imagesflags.plugin {
	case "e2e":

		if len(imagesflags.e2eRegistryConfig) > 0 {
			// Check if the e2e file exists
			if _, err := os.Stat(imagesflags.e2eRegistryConfig); err != nil {
				errlog.LogError(errors.Errorf("file does not exist or cannot be opened: %v", imagesflags.e2eRegistryConfig))
				os.Exit(1)
			}
		}

		sbc, err := getSonobuoyClientFromKubecfg(imagesflags.kubeconfig)
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

func pullImages(cmd *cobra.Command, args []string) {
	switch imagesflags.plugin {
	case "e2e":

		sbc, err := getSonobuoyClientFromKubecfg(imagesflags.kubeconfig)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		version, err := sbc.Version()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get Sonobuoy client"))
			os.Exit(1)
		}

		upstreamImages, err := image.GetImages(imagesflags.e2eRegistryConfig, version)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't init upstream registry list"))
			os.Exit(1)
		}

		// Init client
		imageClient := image.NewImageClient()

		// Pull all images
		errs := imageClient.PullImages(upstreamImages, numDockerRetries)
		for _, err := range errs {
			errlog.LogError(err)
		}

		if len(errs) > 0 {
			os.Exit(1)
		}

	default:
		errlog.LogError(errors.Errorf("Unsupported plugin: %v", imagesflags.plugin))
		os.Exit(1)
	}
}

func downloadImages(cmd *cobra.Command, args []string) {
	switch imagesflags.plugin {
	case "e2e":

		sbc, err := getSonobuoyClientFromKubecfg(imagesflags.kubeconfig)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		version, err := sbc.Version()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get Sonobuoy client"))
			os.Exit(1)
		}

		upstreamImages, err := image.GetImages(defaultE2ERegistries, version)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't init upstream registry list"))
			os.Exit(1)
		}

		images := []string{}
		for _, v := range upstreamImages {
			images = append(images, v.GetE2EImage())
		}

		// Init client
		imageClient := image.NewImageClient()

		fileName, err := imageClient.DownloadImages(images, version)
		if err != nil {
			errlog.LogError(err)
			os.Exit(1)
		}

		fmt.Println(fileName)

	default:
		errlog.LogError(errors.Errorf("Unsupported plugin: %v", imagesflags.plugin))
		os.Exit(1)
	}
}

func pushImages(cmd *cobra.Command, args []string) {

	switch imagesflags.plugin {
	case "e2e":

		if len(imagesflags.e2eRegistryConfig) > 0 {
			// Check if the e2e file exists
			if _, err := os.Stat(imagesflags.e2eRegistryConfig); err != nil {
				errlog.LogError(errors.Errorf("file does not exist or cannot be opened: %v", imagesflags.e2eRegistryConfig))
				os.Exit(1)
			}
		}

		// Check if the e2e file exists
		if _, err := os.Stat(imagesflags.e2eRegistryConfig); os.IsNotExist(err) {
			errlog.LogError(errors.Errorf("file does not exist or cannot be opened: %v", imagesflags.e2eRegistryConfig))
			os.Exit(1)
		}

		sbc, err := getSonobuoyClientFromKubecfg(imagesflags.kubeconfig)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		version, err := sbc.Version()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get Sonobuoy client"))
			os.Exit(1)
		}

		upstreamImages, err := image.GetImages("", version)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't init upstream registry list"))
			os.Exit(1)
		}

		privateImages, err := image.GetImages(imagesflags.e2eRegistryConfig, version)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't init upstream registry list"))
			os.Exit(1)
		}

		// Init client
		imageClient := image.NewImageClient()

		// Push all images
		errs := imageClient.PushImages(upstreamImages, privateImages, numDockerRetries)
		for _, err := range errs {
			errlog.LogError(err)
		}

	default:
		errlog.LogError(errors.Errorf("Unsupported plugin: %v", imagesflags.plugin))
		os.Exit(1)
	}

}

func deleteImages(cmd *cobra.Command, args []string) {
	switch imagesflags.plugin {
	case "e2e":
		sbc, err := getSonobuoyClientFromKubecfg(imagesflags.kubeconfig)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "could not create sonobuoy client"))
			os.Exit(1)
		}

		version, err := sbc.Version()
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't get Sonobuoy client"))
			os.Exit(1)
		}

		images, err := image.GetImages(imagesflags.e2eRegistryConfig, version)
		if err != nil {
			errlog.LogError(errors.Wrap(err, "couldn't init registry list"))
			os.Exit(1)
		}

		// Init client
		imageClient := image.NewImageClient()

		errs := imageClient.DeleteImages(images, numDockerRetries)
		for _, err := range errs {
			errlog.LogError(err)
		}

	default:
		errlog.LogError(errors.Errorf("Unsupported plugin: %v", imagesflags.plugin))
		os.Exit(1)
	}
}
