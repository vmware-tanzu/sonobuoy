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
	"sort"

	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/image"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Number times to retry docker commands before giving up
const (
	numDockerRetries  = 1
	e2ePlugin         = "e2e"
	systemdLogsPlugin = "systemd-logs"
)

type imagesFlags struct {
	e2eRegistryConfig string
	plugins           []string
	kubeconfig        Kubeconfig
	customRegistry    string
	dryRun            bool
	k8sVersion        string
}

func NewCmdImages() *cobra.Command {
	var flags imagesFlags
	// Main command
	cmd := &cobra.Command{
		Use:   "images",
		Short: "Manage images used in a plugin. Supported plugins are: 'e2e'",
		Run: func(cmd *cobra.Command, args []string) {
			var client image.Client
			if flags.dryRun {
				client = image.DryRunClient{}
			} else {
				client = image.NewDockerClient()
			}
			version, err := getClusterVersion(flags.k8sVersion, flags.kubeconfig)
			if err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
			if err := listImages(flags.plugins, version, client); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}

	AddKubeconfigFlag(&flags.kubeconfig, cmd.Flags())
	AddPluginListFlag(&flags.plugins, cmd.Flags())
	AddKubernetesVersionFlag(&flags.k8sVersion, cmd.Flags())

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
			var client image.Client
			if flags.dryRun {
				client = image.DryRunClient{}
			} else {
				client = image.NewDockerClient()
			}
			version, err := getClusterVersion(flags.k8sVersion, flags.kubeconfig)
			if err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
			if errs := pullImages(flags.plugins, flags.e2eRegistryConfig, version, client); len(errs) > 0 {
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
	AddPluginListFlag(&flags.plugins, pullCmd.Flags())
	AddDryRunFlag(&flags.dryRun, pullCmd.Flags())
	AddKubernetesVersionFlag(&flags.k8sVersion, pullCmd.Flags())

	return pullCmd
}

func pushCmd() *cobra.Command {
	var flags imagesFlags
	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Pushes images to docker registry for a specific plugin",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if contains(flags.plugins, e2ePlugin) && len(flags.e2eRegistryConfig) == 0 {
				return fmt.Errorf("required flag %q not set", e2eRegistryConfigFlag)
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			var client image.Client
			if flags.dryRun {
				client = image.DryRunClient{}
			} else {
				client = image.NewDockerClient()
			}
			version, err := getClusterVersion(flags.k8sVersion, flags.kubeconfig)
			if err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
			if errs := pushImages(flags.plugins, flags.customRegistry, flags.e2eRegistryConfig, version, client); len(errs) > 0 {
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
	AddPluginListFlag(&flags.plugins, pushCmd.Flags())
	AddCustomRegistryFlag(&flags.customRegistry, pushCmd.Flags())
	AddDryRunFlag(&flags.dryRun, pushCmd.Flags())
	pushCmd.MarkFlagRequired(customRegistryFlag)
	AddKubernetesVersionFlag(&flags.k8sVersion, pushCmd.Flags())

	return pushCmd
}

func downloadCmd() *cobra.Command {
	var flags imagesFlags
	downloadCmd := &cobra.Command{
		Use:   "download",
		Short: "Saves downloaded images from local docker client to a tar file",
		Run: func(cmd *cobra.Command, args []string) {
			var client image.Client
			if flags.dryRun {
				client = image.DryRunClient{}
			} else {
				client = image.NewDockerClient()
			}
			version, err := getClusterVersion(flags.k8sVersion, flags.kubeconfig)
			if err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
			if err := downloadImages(flags.plugins, flags.e2eRegistryConfig, version, client); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(0),
	}
	AddE2ERegistryConfigFlag(&flags.e2eRegistryConfig, downloadCmd.Flags())
	AddKubeconfigFlag(&flags.kubeconfig, downloadCmd.Flags())
	AddPluginListFlag(&flags.plugins, downloadCmd.Flags())
	AddDryRunFlag(&flags.dryRun, downloadCmd.Flags())
	AddKubernetesVersionFlag(&flags.k8sVersion, downloadCmd.Flags())

	return downloadCmd
}

func deleteCmd() *cobra.Command {
	var flags imagesFlags
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes all images downloaded to local docker client",
		Run: func(cmd *cobra.Command, args []string) {
			var client image.Client
			if flags.dryRun {
				client = image.DryRunClient{}
			} else {
				client = image.NewDockerClient()
			}

			if errs := deleteImages(flags.plugins, flags.kubeconfig, flags.e2eRegistryConfig, flags.k8sVersion, client); len(errs) > 0 {
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
	AddPluginListFlag(&flags.plugins, deleteCmd.Flags())
	AddDryRunFlag(&flags.dryRun, deleteCmd.Flags())
	AddKubernetesVersionFlag(&flags.k8sVersion, deleteCmd.Flags())

	return deleteCmd
}

// getClusterVersion will return either the given string or, if empty, use the kubeconfig
// to reach out to the server and check its version.
func getClusterVersion(k8sVersion string, kubeconfig Kubeconfig) (string, error) {
	if len(k8sVersion) > 0 {
		return k8sVersion, nil
	}

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

func listImages(plugins []string, k8sVersion string, client image.Client) error {
	images, err := collectPluginsImages(plugins, k8sVersion, client)
	if err != nil {
		return errors.Wrap(err, "unable to collect images of plugins")
	}
	sort.Strings(images)
	for _, image := range images {
		fmt.Println(image)
	}
	return nil
}

func pullImages(plugins []string, e2eRegistryConfig, k8sVersion string, client image.Client) []error {
	images, err := collectPluginsImages(plugins, k8sVersion, client)
	if err != nil {
		return []error{err, errors.Errorf("unable to collect images of plugins")}
	}
	if e2eRegistryConfig != "" {
		imagePairs, err := convertImagesToPairs(images, "", e2eRegistryConfig, k8sVersion)
		if err != nil {
			return []error{err}
		}
		images = []string{}
		for _, imagePair := range imagePairs {
			images = append(images, imagePair.Dst)
		}
	}
	return client.PullImages(images, numDockerRetries)
}

func downloadImages(plugins []string, e2eRegistryConfig, k8sVersion string, client image.Client) error {
	images, err := collectPluginsImages(plugins, k8sVersion, client)
	if err != nil {
		return errors.Wrapf(err, "unable to collect images of plugins")
	}
	if e2eRegistryConfig != "" {
		imagePairs, err := convertImagesToPairs(images, "", e2eRegistryConfig, k8sVersion)
		if err != nil {
			return err
		}
		images = []string{}
		for _, imagePair := range imagePairs {
			images = append(images, imagePair.Dst)
		}
	}
	filename, err := client.DownloadImages(images, k8sVersion)
	if err != nil {
		return errors.Wrap(err, "unable to download images")
	}
	fmt.Println(filename)
	return nil
}

func pushImages(plugins []string, customRegistry, e2eRegistryConfig, k8sVersion string, client image.Client) []error {
	images, err := collectPluginsImages(plugins, k8sVersion, client)
	if err != nil {
		return []error{err, errors.Errorf("unable to collect images of plugins")}
	}
	imagePairs, err := convertImagesToPairs(images, customRegistry, e2eRegistryConfig, k8sVersion)
	if err != nil {
		return []error{err}
	}
	return client.PushImages(imagePairs, numDockerRetries)
}

func deleteImages(plugins []string, kubeconfig Kubeconfig, e2eRegistryConfig, k8sVersion string, client image.Client) []error {
	images, err := collectPluginsImages(plugins, k8sVersion, client)
	if err != nil {
		return []error{err, errors.Errorf("unable to collect images of plugins")}
	}
	if e2eRegistryConfig != "" {
		imagePairs, err := convertImagesToPairs(images, "", e2eRegistryConfig, k8sVersion)
		if err != nil {
			return []error{err}
		}
		images = []string{}
		for _, imagePair := range imagePairs {
			images = append(images, imagePair.Dst)
		}
	}
	return client.DeleteImages(images, numDockerRetries)
}

func contains(set []string, val string) bool {
	for _, v := range set {
		if v == val {
			return true
		}
	}
	return false
}
