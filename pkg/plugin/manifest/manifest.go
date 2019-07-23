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

package manifest

import (
	corev1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SonobuoyConfig is the Sonobuoy metadata that plugins all supply
type SonobuoyConfig struct {
	// Driver is the way in which this plugin is run. Either 'Job' or 'Daemonset'.
	Driver string `json:"driver"`

	// Name is the user-facing name for the plugin.
	PluginName string `json:"plugin-name"`

	// ResultType should uniquely identify the plugin and is used by the aggregator
	// to track what results should be reported back to it.
	ResultType string `json:"result-type"`

	// SkipCleanup informs Sonobuoy to leave the pods created for this plugin running,
	// after the run completes instead of deleting them as part of default, cleanup behavior.
	SkipCleanup bool `json:"skip-cleanup,omitempty"`

	objectKind
}

// DeepCopy makes a deep copy (needed by DeepCopyObject)
func (s *SonobuoyConfig) DeepCopy() *SonobuoyConfig {
	return &SonobuoyConfig{
		Driver:      s.Driver,
		PluginName:  s.PluginName,
		ResultType:  s.ResultType,
		SkipCleanup: s.SkipCleanup,
		objectKind:  objectKind{s.objectKind.gvk},
	}
}

// Manifest is the high-level manifest for a plugin
type Manifest struct {
	SonobuoyConfig SonobuoyConfig `json:"sonobuoy-config"`
	Spec           Container      `json:"spec"`
	ExtraVolumes   []Volume       `json:"extra-volumes,omitempty"`
	objectKind
}

// DeepCopyObject is required by runtime.Object
func (m *Manifest) DeepCopyObject() kuberuntime.Object {
	return &Manifest{
		SonobuoyConfig: *m.SonobuoyConfig.DeepCopy(),
		Spec:           *m.Spec.DeepCopy(),
		objectKind:     objectKind{m.gvk},
	}
}

// GetObjectKind is required by runtime.Object
func (m *Manifest) GetObjectKind() schema.ObjectKind { return m }

// Container is a thin wrapper around coreV1.Container that supplies DeepCopyObject and GetObjectKind
type Container struct {
	corev1.Container
	objectKind
}

// DeepCopy wraps Container.DeepCopy, copying the objectKind as well.
func (c *Container) DeepCopy() *Container {
	return &Container{
		Container:  *c.Container.DeepCopy(),
		objectKind: objectKind{c.gvk},
	}
}

// DeepCopyObject is just DeepCopy, needed for runtime.Object
func (c *Container) DeepCopyObject() kuberuntime.Object { return c.DeepCopy() }

// GetObjectKind returns the underlying objectKind, needed for runtime.Object
func (c *Container) GetObjectKind() schema.ObjectKind { return c }

// Volume is a thin wrapper around coreV1.Volume that supplies DeepCopyObject and GetObjectKind
type Volume struct {
	corev1.Volume
	objectKind
}

// DeepCopy wraps Volume.DeepCopy, copying the objectKind as well.
func (v *Volume) DeepCopy() *Volume {
	return &Volume{
		Volume:     *v.Volume.DeepCopy(),
		objectKind: objectKind{v.gvk},
	}
}

// DeepCopyObject is just DeepCopy, needed for runtime.Object
func (v *Volume) DeepCopyObject() kuberuntime.Object { return v.DeepCopy() }

// GetObjectKind returns the underlying objectKind, needed for runtime.Object
func (v *Volume) GetObjectKind() schema.ObjectKind { return v }
