package manifest

import (
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// Encoder is a runtime.Encoder for Sonobuoy's manifest objects
var Encoder kuberuntime.Encoder

// Decoder is a runtime.Decoder for Sonobuoy's manifest objects
var Decoder kuberuntime.Decoder

// GroupVersion is the schema groupVersion for Sonobuoy
var GroupVersion = schema.GroupVersion{Group: "sonobuoy", Version: "v0"}

func init() {
	schema := kuberuntime.NewScheme()
	schema.AddKnownTypes(GroupVersion,
		&Container{},
		&Manifest{},
	)
	codecs := serializer.NewCodecFactory(schema)

	serializer := json.NewYAMLSerializer(
		json.DefaultMetaFactory,
		&creator{},
		&typer{},
	)

	Encoder = codecs.EncoderForVersion(serializer, GroupVersion)
	Decoder = codecs.DecoderToVersion(serializer, GroupVersion)
}

// ContainerToYAML abuses APIMachinery to directly serialize a container to YAML
func ContainerToYAML(container *v1.Container) (string, error) {
	oc := &Container{Container: *container}
	b, err := kuberuntime.Encode(Encoder, oc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

type objectKind struct {
	gvk schema.GroupVersionKind
}

func (o *objectKind) SetGroupVersionKind(gvk schema.GroupVersionKind) { o.gvk = gvk }
func (o *objectKind) GroupVersionKind() schema.GroupVersionKind       { return o.gvk }

type creator struct{}

func (c *creator) New(kind schema.GroupVersionKind) (kuberuntime.Object, error) {
	if GroupVersion != kind.GroupVersion() {
		return nil, fmt.Errorf("unrecognised group version %s", kind.GroupVersion().String())
	}
	switch kind.Kind {
	case "container":
		return &Container{}, nil
	case "manifest":
		return &Manifest{}, nil
	default:
		return nil, fmt.Errorf("unrecognised kind %v", kind.Kind)
	}
}

type typer struct{}

func (t *typer) ObjectKinds(obj kuberuntime.Object) ([]schema.GroupVersionKind, bool, error) {
	switch obj.(type) {
	case (*Container):
		return []schema.GroupVersionKind{GroupVersion.WithKind("container")}, true, nil
	case (*Manifest):
		return []schema.GroupVersionKind{GroupVersion.WithKind("manifest")}, true, nil
	default:
		return []schema.GroupVersionKind{}, false, errors.New("not a pod")
	}
}

func (t *typer) Recognizes(kind schema.GroupVersionKind) bool {
	if GroupVersion != kind.GroupVersion() {
		return false
	}
	switch kind.Kind {
	case "pod":
		return true
	default:
		return false
	}
}
