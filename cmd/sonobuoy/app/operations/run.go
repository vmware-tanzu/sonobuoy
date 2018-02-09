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
	"io"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
)

// RunConfig are the input options for running
// TODO: We should expose FOCUS and other options with sane defaults
type RunConfig struct {
	GenConfig
}

// Run generates the manifest, then tries to apply it to the cluster.
// returns created resources or an error
func Run(cfg RunConfig) error {
	manifest, err := GenerateManifest(cfg.GenConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't run invalid manifest")
	}
	buf := bytes.NewBuffer(manifest)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err := kubeConfig.ClientConfig()

	if err != nil {
		return errors.Wrap(err, "couldn't get REST client")
	}

	d := yaml.NewYAMLOrJSONDecoder(buf, 4096)

	mapper := legacyscheme.Registry
	for {
		ext := runtime.RawExtension{}
		obj := unstructured.Unstructured{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "couldn't decode template")
		}

		// Skip over empty or partial objects
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), ext.Raw, &obj); err != nil {
			return errors.Wrap(err, "couldn't decode template")
		}

		fmt.Printf("%+v\n", obj)

		gvk := obj.GroupVersionKind()

		*restConfig.GroupVersion = gvk.GroupVersion()
		if gvk.Group == "" {
			restConfig.APIPath = "/api"
		} else {
			restConfig.APIPath = "/apis"
		}

		client, err := dynamic.NewClient(restConfig)
		if err != nil {
			return errors.Wrapf(err, "couldn't make rest client")
		}

		mapping, err := mapper.RESTMapper(gvk.GroupVersion()).RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return errors.Wrapf(err, "couldn't get resource")
		}

		resource := mapping.Resource
		name, err := mapping.MetadataAccessor.Name(&obj)
		if err != nil {
			return errors.Wrapf(err, "couldn't get name for resource %s", resource)
		}

		namespace, err := mapping.MetadataAccessor.Namespace(&obj)
		if err != nil {
			return errors.Wrapf(err, "couldn't get namespace for object %s", name)
		}

		_, err = client.Resource(&v1.APIResource{
			Name:       resource,
			Namespaced: namespace != "",
		}, namespace).Create(&obj)

		if err != nil {
			return errors.Wrapf(err, "couldn't create resource %s", name)
		}

		logrus.WithFields(logrus.Fields{
			"name":      name,
			"namespace": namespace,
			"resource":  resource,
		}).Info("created object")

	}
	return nil
}
