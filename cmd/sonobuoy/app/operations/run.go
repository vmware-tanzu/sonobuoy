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
	"io"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/heptio/sonobuoy/cmd/sonobuoy/utils/kubeconfig"
)

const bufferSize = 4096

// RunConfig are the input options for running
// TODO: We should expose FOCUS and other options with sane defaults
type RunConfig struct {
	GenConfig
	Kubecfg kubeconfig.Config
}

// Run generates the manifest, then tries to apply it to the cluster.
// returns created resources or an error
func Run(cfg RunConfig) error {
	manifest, err := GenerateManifest(cfg.GenConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't run invalid manifest")
	}

	buf := bytes.NewBuffer(manifest)

	restConfig, err := cfg.Kubecfg.Get()
	if err != nil {
		return errors.Wrap(err, "couldn't get REST client")
	}

	mapper, err := newMapper(restConfig)
	if err != nil {
		return errors.Wrap(err, "couldn't retrieve API spec from server")
	}

	d := yaml.NewYAMLOrJSONDecoder(buf, bufferSize)

	for {
		ext := runtime.RawExtension{}
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

		obj := unstructured.Unstructured{}
		if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), ext.Raw, &obj); err != nil {
			return errors.Wrap(err, "couldn't decode template")
		}

		err := createObject(restConfig, &obj, mapper)
		if err != nil {
			return errors.Wrap(err, "failed to create object")
		}
	}
	return nil
}

func createObject(cfg *rest.Config, obj *unstructured.Unstructured, mapper meta.RESTMapper) error {
	client, err := getClient(cfg, obj.GroupVersionKind())
	if err != nil {
		return errors.Wrap(err, "could not make kubernetes client")
	}

	mapping, err := mapper.RESTMapping(
		obj.GroupVersionKind().GroupKind(),
		obj.GroupVersionKind().Version,
	)
	if err != nil {
		return errors.Wrap(err, "could not get resource for object")
	}
	resource := mapping.Resource

	name, namespace, err := getNames(obj)
	if err != nil {
		return errors.Wrap(err, "couldn't retrive object metadata")
	}

	_, err = client.Resource(&v1.APIResource{
		Name:       resource,
		Namespaced: namespace != "",
	}, namespace).Create(obj)

	if err != nil {
		return errors.Wrapf(err, "failed to create API resource %s", name)
	}

	logrus.WithFields(logrus.Fields{
		"name":      name,
		"namespace": namespace,
		"resource":  resource,
	}).Info("created object")
	return nil
}

func getClient(cfg *rest.Config, gvk schema.GroupVersionKind) (*dynamic.Client, error) {
	gk := gvk.GroupVersion()

	cfg.GroupVersion = &gk
	if gvk.Group == "" {
		cfg.APIPath = "/api"
	} else {
		cfg.APIPath = "/apis"
	}

	return dynamic.NewClient(cfg)
}

func newMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create discovery client")
	}
	resources, err := discovery.GetAPIGroupResources(client)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve API resources from server")
	}

	return discovery.NewRESTMapper(
		resources,
		unstruturedVersionInterface,
	), nil
}

func getNames(obj runtime.Object) (string, string, error) {
	accessor := meta.NewAccessor()
	name, err := accessor.Name(obj)
	if err != nil {
		return "", "", errors.Wrapf(err, "couldn't get name for object %T", obj)
	}

	namespace, err := accessor.Namespace(obj)
	if err != nil {
		return "", "", errors.Wrapf(err, "couldn't get namespac e for object %s", name)
	}

	return name, namespace, nil
}

// implements meta.VersionInterfacesFunc
func unstruturedVersionInterface(version schema.GroupVersion) (*meta.VersionInterfaces, error) {
	return &meta.VersionInterfaces{
		ObjectConvertor:  &unstructured.UnstructuredObjectConverter{},
		MetadataAccessor: meta.NewAccessor(),
	}, nil
}
