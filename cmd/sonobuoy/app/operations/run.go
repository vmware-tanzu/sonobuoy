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

package operations

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/kubectl/resource"
)

// RunConfig are the input options for running
// TODO: We should expose FOCUS and other options with sane defaults
type RunConfig struct {
	GenConfig
}

// Run generates the manifest, then tries to apply it to the cluster.
// returns created resources or an error
func Run(cfg RunConfig) error {
	yaml, err := GenerateManifest(cfg.GenConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't run invalid manifest")
	}
	buf := bytes.NewBuffer(yaml)

	restConfig, err := cmdutil.NewClientAccessFactory(nil).ClientConfig()
	if err != nil {
		return errors.Wrap(err, "couldn't get REST client")
	}

	err = cmdutil.NewFactory(nil).
		NewBuilder().
		Unstructured().
		NamespaceParam(cfg.Namespace.Get()).
		RequireNamespace().
		Stream(buf, fmt.Sprintf("%s.yaml", cfg.ModeName.String())).
		Flatten().
		Do().
		Visit(func(info *resource.Info, err error) error {
			if err != nil {
				return err
			}

			*restConfig.GroupVersion = info.ResourceMapping().GroupVersionKind.GroupVersion()
			if info.ResourceMapping().GroupVersionKind.Group == "" {
				restConfig.APIPath = "/api"
			} else {
				restConfig.APIPath = "/apis"
			}

			client, err := rest.RESTClientFor(restConfig)
			if err != nil {
				return errors.Wrapf(err, "couldn't make rest client for %s", info.Name)
			}

			helper := resource.NewHelper(client, info.ResourceMapping())

			_, err = helper.Create(
				cfg.Namespace.Get(),
				false, // don't overwrite existing resources
				info.AsUnstructured(),
			)
			if err != nil {
				return errors.Wrapf(err, "couldn't create resource %v (%v)", info.Name, info.ResourceMapping().Resource)
			}
			logrus.WithFields(logrus.Fields{
				"name":      info.Name,
				"namespace": info.Namespace,
				"resource":  info.ResourceMapping().Resource,
			}).Info("created object")
			return nil
		})

	return errors.Wrap(err, "failed to apply resource")
}
