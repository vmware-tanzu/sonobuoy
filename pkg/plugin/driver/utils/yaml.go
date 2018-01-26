package utils

import (
	"errors"

	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

var encoder kuberuntime.Encoder

func init() {
	schema := kuberuntime.NewScheme()
	schema.AddKnownTypes(schemeGroupVersion, &objectContainer{})
	codecs := serializer.NewCodecFactory(schema)

	serializer := json.NewYAMLSerializer(
		json.DefaultMetaFactory,
		&creator{},
		&typer{},
	)

	encoder = codecs.EncoderForVersion(serializer, schemeGroupVersion)
}

// ContainerToYAML abuses APIMachinery to directly serialize a container to YAML
func ContainerToYAML(container *v1.Container) (string, error) {
	oc := newObjectContainer(container)
	b, err := kuberuntime.Encode(encoder, oc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type objectContainer struct {
	v1.Container
	gvk schema.GroupVersionKind
}

func newObjectContainer(c *v1.Container) *objectContainer {
	return &objectContainer{
		*c,
		schemeGroupVersion.WithKind("pod"),
	}
}

var schemeGroupVersion = schema.GroupVersion{Group: "pod", Version: "v0"}
var schemeGroupVersionKind = schemeGroupVersion.WithKind("pod")

func (o *objectContainer) DeepCopyObject() kuberuntime.Object {
	return newObjectContainer(o.DeepCopy())
}
func (o *objectContainer) GetObjectKind() schema.ObjectKind                { return o }
func (o *objectContainer) SetGroupVersionKind(gvk schema.GroupVersionKind) { o.gvk = gvk }
func (o *objectContainer) GroupVersionKind() schema.GroupVersionKind       { return o.gvk }

type creator struct{}

func (c *creator) New(kind schema.GroupVersionKind) (kuberuntime.Object, error) {
	if kind.Kind == "pod" {
		return &objectContainer{}, nil
	}
	return nil, errors.New("only pod")
}

type typer struct{}

func (t *typer) ObjectKinds(obj kuberuntime.Object) ([]schema.GroupVersionKind, bool, error) {
	if _, ok := obj.(*objectContainer); ok {
		return []schema.GroupVersionKind{schemeGroupVersionKind}, true, nil
	}
	return []schema.GroupVersionKind{}, false, errors.New("not a pod")
}

func (t *typer) Recognizes(kind schema.GroupVersionKind) bool {
	return kind.String() == schemeGroupVersionKind.String()
}
