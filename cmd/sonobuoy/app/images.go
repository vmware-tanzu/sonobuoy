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
	var flags imagesFlags
	// Main command
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage images used in a plugin. Supported plugins are: 'e2e'",
		Run: func(cmd *cobra.Command, args []string) {
			if err := listImages(flags.plugin, flags.kubeconfig); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}

	AddKubeconfigFlag(&flags.kubeconfig, cmd.Flags())
	AddPluginFlag(&flags.plugin, cmd.Flags())

	cmd.AddCommand(pullCmd())
	cmd.AddCommand(pushCmd())
	cmd.AddCommand(downloadCmd())
	cmd.AddCommand(deleteCmd())

	return cmd
}

func pullCmd() *cobra.Command {
	var flags imagesFlags
	pullCmd := &cobra.Command{
		Use:   "pull",
		Short: "Pulls images to local docker client for a specific plugin",
		Run: func(cmd *cobra.Command, args []string) {
			if errs := pullImages(flags.plugin, flags.kubeconfig, flags.e2eRegistryConfig); len(errs) > 0 {
				for _, err := range errs {
					errlog.LogError(err)
				}
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&flags.e2eRegistryConfig, pullCmd.Flags())
	AddKubeconfigFlag(&flags.kubeconfig, pullCmd.Flags())
	AddPluginFlag(&flags.plugin, pullCmd.Flags())

	return pullCmd
}

func pushCmd() *cobra.Command {
	var flags imagesFlags
	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Pushes images to docker registry for a specific plugin",
		Run: func(cmd *cobra.Command, args []string) {
			if errs := pushImages(flags.plugin, flags.kubeconfig, flags.e2eRegistryConfig); len(errs) > 0 {
				for _, err := range errs {
					errlog.LogError(err)
				}
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&flags.e2eRegistryConfig, pushCmd.Flags())
	AddKubeconfigFlag(&flags.kubeconfig, pushCmd.Flags())
	AddPluginFlag(&flags.plugin, pushCmd.Flags())
	// TODO(bridget): This won't be required when dealing with other plugins
	pushCmd.MarkFlagRequired(e2eRegistryConfigFlag)

	return pushCmd
}

func downloadCmd() *cobra.Command {
	var flags imagesFlags
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Saves downloaded images from local docker client to a tar file",
		Run: func(cmd *cobra.Command, args []string) {
			if err := downloadImages(flags.plugin, flags.kubeconfig, flags.e2eRegistryConfig); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&flags.e2eRegistryConfig, downloadCmd.Flags())
	AddKubeconfigFlag(&flags.kubeconfig, downloadCmd.Flags())
	AddPluginFlag(&flags.plugin, downloadCmd.Flags())
	return downloadCmd
}

func deleteCmd() *cobra.Command {
	var flags imagesFlags
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes all images downloaded to local docker client",
		Run: func(cmd *cobra.Command, args []string) {
			if errs := deleteImages(flags.plugin, flags.kubeconfig, flags.e2eRegistryConfig); len(errs) > 0 {
				for _, err := range errs {
					errlog.LogError(err)
				}
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&flags.e2eRegistryConfig, deleteCmd.Flags())
	AddKubeconfigFlag(&flags.kubeconfig, deleteCmd.Flags())
	AddPluginFlag(&flags.plugin, deleteCmd.Flags())
	return deleteCmd
}

func getClusterVersion(kubeconfig Kubeconfig) (string, error) {
	sbc, err := getSonobuoyClientFromKubecfg(kubeconfig)
	if err != nil {
		return "", errors.Wrap(err, "couldn't create sonobuoy client")
	}

	version, err := sbc.Version()
	if err != nil {
		return "", errors.Wrap(err, "couldn't get Sonobuoy client")
	}

	return version, nil
}

func listImages(plugin string, kubeconfig Kubeconfig) error {
	switch plugin {
	case "e2e":
		version, err := getClusterVersion(kubeconfig)
		if err != nil {
			return errors.Wrap(err, "failed to get cluster version")
		}

		defaultImages, err := image.GetE2EImages(defaultE2ERegistries, version)
		if err != nil {
			return errors.Wrap(err, "couldn't get images")
		}

		for _, image := range defaultImages {
			fmt.Println(image)
		}
	default:
		return errors.Errorf("Unsupported plugin: %v", plugin)
	}

	return nil
}

func pullImages(plugin string, kubeconfig Kubeconfig, e2eRegistryConfig string) []error {
	switch plugin {
	case "e2e":
		version, err := getClusterVersion(kubeconfig)
		if err != nil {
			return []error{errors.Wrap(err, "failed to get cluster version")}
		}

		images, err := image.GetE2EImages(e2eRegistryConfig, version)
		if err != nil {
			return []error{errors.Wrap(err, "couldn't get images")}
		}

		imageClient := image.NewImageClient()

		return imageClient.PullImages(images, numDockerRetries)
	default:
		return []error{errors.Errorf("Unsupported plugin: %v", plugin)}
	}

}

func downloadImages(plugin string, kubeconfig Kubeconfig, e2eRegistryConfig string) error {
	switch plugin {
	case "e2e":
		version, err := getClusterVersion(kubeconfig)
		if err != nil {
			return errors.Wrap(err, "failed to get cluster version")
		}

		images, err := image.GetE2EImages(e2eRegistryConfig, version)
		if err != nil {
			return errors.Wrap(err, "couldn't get images")
		}

		imageClient := image.NewImageClient()

		fileName, err := imageClient.DownloadImages(images, version)
		if err != nil {
			return err
		}

		fmt.Println(fileName)

	default:
		return errors.Errorf("Unsupported plugin: %v", plugin)
	}

	return nil
}

func pushImages(plugin string, kubeconfig Kubeconfig, e2eRegistryConfig string) []error {
	switch plugin {
	case "e2e":
		version, err := getClusterVersion(kubeconfig)
		if err != nil {
			return []error{errors.Wrap(err, "failed to get cluster version")}
		}

		defaultImages, err := image.GetE2EImages(defaultE2ERegistries, version)
		if err != nil {
			return []error{errors.Wrap(err, "couldn't get images")}
		}

		privateImages, err := image.GetE2EImages(e2eRegistryConfig, version)
		if err != nil {
			return []error{errors.Wrap(err, "couldn't get images")}
		}

		imageClient := image.NewImageClient()

		return imageClient.PushImages(defaultImages, privateImages, numDockerRetries)
	default:
		return []error{errors.Errorf("Unsupported plugin: %v", plugin)}
	}
}

func deleteImages(plugin string, kubeconfig Kubeconfig, e2eRegistryConfig string) []error {
	switch plugin {
	case "e2e":
		version, err := getClusterVersion(kubeconfig)
		if err != nil {
			return []error{errors.Wrap(err, "failed to get cluster version")}
		}

		images, err := image.GetE2EImages(e2eRegistryConfig, version)
		if err != nil {
			return []error{errors.Wrap(err, "couldn't get images")}
		}

		imageClient := image.NewImageClient()

		return imageClient.DeleteImages(images, numDockerRetries)

	default:
		return []error{errors.Errorf("Unsupported plugin: %v", plugin)}
	}
}
