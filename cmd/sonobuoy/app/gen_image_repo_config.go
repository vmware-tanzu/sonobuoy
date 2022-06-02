/*
Copyright the Sonobuoy contributors 2019

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
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// NewCmdGenImageRepoConfig creates the `default-image-config` subcommand for `gen`
// which will print out the default image registry config for the E2E tests which is
// used with the `images` and `run` command.
func NewCmdGenImageRepoConfig() *cobra.Command {
	var cfg Kubeconfig
	cmd := &cobra.Command{
		Use:   "default-image-config",
		Short: "Generates the default image registry config for the e2e plugin",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := defaultImageRegConfig(&cfg)
			if err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
			fmt.Println(string(s))
		},
		Args: cobra.NoArgs,
	}

	genPluginSet := pflag.NewFlagSet("default-image-config", pflag.ExitOnError)
	AddKubeconfigFlag(&cfg, genPluginSet)
	cmd.Flags().AddFlagSet(genPluginSet)

	return cmd
}

func defaultImageRegConfig(cfg *Kubeconfig) ([]byte, error) {
	sbc, err := getSonobuoyClientFromKubecfg(*cfg)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not create sonobuoy client")
	}

	version, err := sbc.Version()
	if err != nil {
		return []byte{}, errors.Wrap(err, "couldn't get Kubernetes version")
	}

	registries, err := image.GetDefaultImageRegistries(version)
	if err != nil {
		return []byte{}, errors.Wrap(err, "couldn't get image registries for version")
	}

	d, err := yaml.Marshal(&registries)
	if err != nil {
		return []byte{}, errors.Wrap(err, "couldn't marshal registry information")
	}
	return d, nil
}
