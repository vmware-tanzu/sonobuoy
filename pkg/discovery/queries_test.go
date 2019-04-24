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

package discovery

import (
	"fmt"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestFilterResources(t *testing.T) {
	// resourceMap is the set all the tests will run against; taken from actual values in a cluster at time of this writing (v1.14.1).
	resourceMap := map[schema.GroupVersion][]v1.APIResource{
		schema.GroupVersion{Group: "", Version: "v1"}:                                  []v1.APIResource{v1.APIResource{Name: "limitranges", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "LimitRange", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"limits"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "pods", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Pod", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"po"}, Categories: []string{"all"}, StorageVersionHash: ""}, v1.APIResource{Name: "events", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Event", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"ev"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "persistentvolumeclaims", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "PersistentVolumeClaim", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"pvc"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "services", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Service", Verbs: v1.Verbs{"create", "delete", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"svc"}, Categories: []string{"all"}, StorageVersionHash: ""}, v1.APIResource{Name: "resourcequotas", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "ResourceQuota", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"quota"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "configmaps", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "ConfigMap", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"cm"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "persistentvolumes", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "PersistentVolume", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"pv"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "podtemplates", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "PodTemplate", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "serviceaccounts", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "ServiceAccount", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"sa"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "nodes", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "Node", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"no"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "bindings", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Binding", Verbs: v1.Verbs{"create"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "endpoints", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Endpoints", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"ep"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "replicationcontrollers", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "ReplicationController", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"rc"}, Categories: []string{"all"}, StorageVersionHash: ""}, v1.APIResource{Name: "componentstatuses", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "ComponentStatus", Verbs: v1.Verbs{"get", "list"}, ShortNames: []string{"cs"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "namespaces", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "Namespace", Verbs: v1.Verbs{"create", "delete", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"ns"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "secrets", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Secret", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "admissionregistration.k8s.io", Version: "v1beta1"}: []v1.APIResource{v1.APIResource{Name: "validatingwebhookconfigurations", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "ValidatingWebhookConfiguration", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "mutatingwebhookconfigurations", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "MutatingWebhookConfiguration", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "apiextensions.k8s.io", Version: "v1beta1"}:         []v1.APIResource{v1.APIResource{Name: "customresourcedefinitions", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "CustomResourceDefinition", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"crd", "crds"}, Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "apiregistration.k8s.io", Version: "v1"}:            []v1.APIResource{v1.APIResource{Name: "apiservices", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "APIService", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "apps", Version: "v1"}:                              []v1.APIResource{v1.APIResource{Name: "statefulsets", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "StatefulSet", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"sts"}, Categories: []string{"all"}, StorageVersionHash: ""}, v1.APIResource{Name: "controllerrevisions", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "ControllerRevision", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "authentication.k8s.io", Version: "v1"}:             []v1.APIResource{v1.APIResource{Name: "tokenreviews", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "TokenReview", Verbs: v1.Verbs{"create"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "authorization.k8s.io", Version: "v1"}:              []v1.APIResource{v1.APIResource{Name: "selfsubjectaccessreviews", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "SelfSubjectAccessReview", Verbs: v1.Verbs{"create"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "subjectaccessreviews", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "SubjectAccessReview", Verbs: v1.Verbs{"create"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "selfsubjectrulesreviews", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "SelfSubjectRulesReview", Verbs: v1.Verbs{"create"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "localsubjectaccessreviews", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "LocalSubjectAccessReview", Verbs: v1.Verbs{"create"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "autoscaling", Version: "v1"}:                       []v1.APIResource{v1.APIResource{Name: "horizontalpodautoscalers", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "HorizontalPodAutoscaler", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"hpa"}, Categories: []string{"all"}, StorageVersionHash: ""}},
		schema.GroupVersion{Group: "batch", Version: "v1"}:                             []v1.APIResource{v1.APIResource{Name: "jobs", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Job", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string{"all"}, StorageVersionHash: ""}},
		schema.GroupVersion{Group: "batch", Version: "v1beta1"}:                        []v1.APIResource{v1.APIResource{Name: "cronjobs", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "CronJob", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"cj"}, Categories: []string{"all"}, StorageVersionHash: ""}},
		schema.GroupVersion{Group: "certificates.k8s.io", Version: "v1beta1"}:          []v1.APIResource{v1.APIResource{Name: "certificatesigningrequests", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "CertificateSigningRequest", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"csr"}, Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "coordination.k8s.io", Version: "v1beta1"}:          []v1.APIResource{v1.APIResource{Name: "leases", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Lease", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "extensions", Version: "v1beta1"}:                   []v1.APIResource{v1.APIResource{Name: "networkpolicies", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "NetworkPolicy", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"netpol"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "ingresses", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Ingress", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"ing"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "replicasets", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "ReplicaSet", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"rs"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "daemonsets", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "DaemonSet", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"ds"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "podsecuritypolicies", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "PodSecurityPolicy", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"psp"}, Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "deployments", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Deployment", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"deploy"}, Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "policy", Version: "v1beta1"}:                       []v1.APIResource{v1.APIResource{Name: "poddisruptionbudgets", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "PodDisruptionBudget", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"pdb"}, Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "rbac.authorization.k8s.io", Version: "v1"}:         []v1.APIResource{v1.APIResource{Name: "rolebindings", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "RoleBinding", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "clusterroles", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "ClusterRole", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "roles", SingularName: "", Namespaced: true, Group: "", Version: "", Kind: "Role", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "clusterrolebindings", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "ClusterRoleBinding", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "scheduling.k8s.io", Version: "v1beta1"}:            []v1.APIResource{v1.APIResource{Name: "priorityclasses", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "PriorityClass", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"pc"}, Categories: []string(nil), StorageVersionHash: ""}},
		schema.GroupVersion{Group: "storage.k8s.io", Version: "v1"}:                    []v1.APIResource{v1.APIResource{Name: "volumeattachments", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "VolumeAttachment", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string(nil), Categories: []string(nil), StorageVersionHash: ""}, v1.APIResource{Name: "storageclasses", SingularName: "", Namespaced: false, Group: "", Version: "", Kind: "StorageClass", Verbs: v1.Verbs{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"}, ShortNames: []string{"sc"}, Categories: []string(nil), StorageVersionHash: ""}},
	}

	tcs := []struct {
		desc          string
		ns            bool
		wantResources []string

		// To test some things we may not want to worry about the whole list
		// and instead worry about some custom concern, hence the customCheck.
		expect      []schema.GroupVersionResource
		customCheck func([]schema.GroupVersionResource) error
	}{
		{
			desc:          "Bad filter will lead to no resources",
			wantResources: []string{"foo"},
			expect:        []schema.GroupVersionResource{},
		}, {
			desc:          "Specific filter will grab one item (namespaced)",
			wantResources: []string{"pods"},
			ns:            true,
			expect: []schema.GroupVersionResource{
				schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			},
		}, {
			desc:          "Filter one item (non-namespaced)",
			wantResources: []string{"storageclasses"},
			expect: []schema.GroupVersionResource{
				schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
			},
		}, {
			desc:          "Filters out namespaced values",
			wantResources: []string{"pods"},
			expect:        []schema.GroupVersionResource{},
		}, {
			desc:          "Filters non-namespaced values",
			wantResources: []string{"storageclasses"},
			ns:            true,
			expect:        []schema.GroupVersionResource{},
		}, {
			desc:          "Empty namespace gets non-namespaced values",
			wantResources: []string{"storageclasses"},
			ns:            false,
			expect: []schema.GroupVersionResource{
				schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
			},
		}, {
			desc:          "Filters listable",
			wantResources: []string{"bindings"},
			ns:            false,
			expect:        []schema.GroupVersionResource{},
		}, {
			desc:          "Filters secrets when querying everything implicitly",
			wantResources: []string{},
			ns:            true,
			customCheck: func(gvrList []schema.GroupVersionResource) error {
				for _, gvr := range gvrList {
					if gvr.Resource == "secrets" {
						return fmt.Errorf("Expected secrets to be missing but found gvr: %#v", gvr)
					}
				}
				return nil
			},
		}, {
			desc:          "Can get secrets explicitly",
			wantResources: []string{"secrets"},
			ns:            true,
			expect: []schema.GroupVersionResource{
				schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			out := filterResources(resourceMap, tc.ns, tc.wantResources)
			if tc.customCheck != nil {
				err := tc.customCheck(out)
				if err != nil {
					t.Fatal(err)
				}
				return
			}

			if diff := pretty.Compare(tc.expect, out); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
