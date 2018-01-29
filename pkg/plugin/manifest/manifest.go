package manifest

import (
	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SonobuoyConfig is the Sonobuoy metadata that plugins all supply
type SonobuoyConfig struct {
	Driver     string `json:"driver"`
	PluginName string `json:"plugin-name"`
	ResultType string `json:"result-type"`
	objectKind
}

// DeepCopy makes a deep copy (needed by DeepCopyObject)
func (s *SonobuoyConfig) DeepCopy() *SonobuoyConfig {
	return &SonobuoyConfig{
		Driver:     s.Driver,
		PluginName: s.PluginName,
		ResultType: s.ResultType,
		objectKind: objectKind{s.objectKind.gvk},
	}
}

// Manifest is the high-level manifest for a plugin
type Manifest struct {
	SonobuoyConfig SonobuoyConfig `json:"sonobuoy-config"`
	Spec           Container      `json:"spec"`
	objectKind
}

// DeepCopyObject is required by runtime.Object
func (p *Manifest) DeepCopyObject() kuberuntime.Object {
	return &Manifest{
		SonobuoyConfig: *p.SonobuoyConfig.DeepCopy(),
		Spec:           *p.Spec.DeepCopy(),
		objectKind:     objectKind{p.gvk},
	}
}

// GetObjectKind is required by runtime.Object
func (p *Manifest) GetObjectKind() schema.ObjectKind { return p }

// Container is a thin wrapper around coreV1.Container that supplies DeepCopyObject and GetObjectKind
type Container struct {
	v1.Container
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
