package dynamic_test

import (
	"errors"
	"testing"

	sonodynamic "github.com/heptio/sonobuoy/pkg/dynamic"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

type testMapper struct{}

func (t *testMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	if gk.Kind == "fail-rest-mapping" {
		return nil, errors.New("some error")
	}
	if gk.Kind == "fail-resource" {
		return &meta.RESTMapping{
			Resource: schema.GroupVersionResource{
				Group:    gk.Group,
				Version:  versions[0],
				Resource: "fail-resource",
			},
		}, nil
	}
	return &meta.RESTMapping{}, nil
}

type testMetadataAccessor struct{}

func (t *testMetadataAccessor) Namespace(obj runtime.Object) (string, error) {
	if obj.GetObjectKind().GroupVersionKind().Kind == "fail-namespace" {
		return "", errors.New("namespace error")
	}
	return "", nil
}
func (t *testMetadataAccessor) Name(obj runtime.Object) (string, error) {
	if obj.GetObjectKind().GroupVersionKind().Kind == "fail-name" {
		return "", errors.New("name error")
	}
	return "", nil
}
func (t *testMetadataAccessor) ResourceVersion(obj runtime.Object) (string, error) { return "", nil }

type testDyanmicInterface struct{}

func (t *testDyanmicInterface) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	if resource.String() == "testing/v1alpha1, Resource=fail-resource" {
		return nil
	}
	return &testNamespaceableResourceInterface{}
}

type testResourceInterface struct{}
type testNamespaceableResourceInterface struct {
	testResourceInterface
}

func (t *testNamespaceableResourceInterface) Namespace(string) dynamic.ResourceInterface {
	return &testResourceInterface{}
}
func (t *testResourceInterface) Create(obj *unstructured.Unstructured, subresources ...string) (*unstructured.Unstructured, error) {
	return obj, nil
}
func (t *testResourceInterface) Update(obj *unstructured.Unstructured, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (t *testResourceInterface) UpdateStatus(obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (t *testResourceInterface) Delete(name string, options *metav1.DeleteOptions, subresources ...string) error {
	return nil
}
func (t *testResourceInterface) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return nil
}
func (t *testResourceInterface) Get(name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}
func (t *testResourceInterface) List(opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return nil, nil
}
func (t *testResourceInterface) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return nil, nil
}
func (t *testResourceInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (*unstructured.Unstructured, error) {
	return nil, nil
}

func TestCreateObject(t *testing.T) {
	testcases := []struct {
		name        string
		kind        string
		expectError bool
	}{
		{
			name:        "simple passing test",
			expectError: false,
		},
		{
			name:        "there should be an error if restmapping fails",
			kind:        "fail-rest-mapping",
			expectError: true,
		},
		{
			name:        "there should be an error if the namespace accessor fails",
			kind:        "fail-namespace",
			expectError: true,
		},
		{
			name:        "there should be an error if name accessor fails",
			kind:        "fail-name",
			expectError: true,
		},
		{
			name:        "there should be an error if Resource fails to return a NamespacableResourceInterface",
			kind:        "fail-resource",
			expectError: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			helper, err := sonodynamic.NewAPIHelper(&testDyanmicInterface{}, &testMapper{}, &testMetadataAccessor{})
			if err != nil {
				t.Fatalf("could not create apihelper: %v", err)
			}
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "testing/v1alpha1",
					"kind":       tc.kind,
				},
			}
			out, err := helper.CreateObject(obj)
			if tc.expectError && err == nil {
				t.Fatalf("expected an error but got nil")
			}
			// return early if we got what we wanted
			if tc.expectError && err != nil {
				return
			}
			if err != nil {
				t.Fatalf("failed to create object: %v", err)
			}
			if out == nil {
				t.Fatalf("out should not be nil")
			}
		})

	}
}
