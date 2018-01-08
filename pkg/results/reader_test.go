package results_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/api/core/v1"

	"github.com/heptio/sonobuoy/pkg/results"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sver "k8s.io/apimachinery/pkg/version"
)

type version struct {
	major int
	minor int
}

func (v *version) path() string {
	return fmt.Sprintf("test_data/results-%v.%v.tar.gz", v.major, v.minor)
}
func (v *version) String() string {
	return fmt.Sprintf("v%v.%v", v.major, v.minor)
}

func MustGetReader(path string, t *testing.T) *results.Reader {
	t.Helper()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read tarball data: %v", err)
	}
	r := bytes.NewReader(data)
	gzipr, err := gzip.NewReader(r)
	if err != nil {
		t.Fatalf("failed to get a gzip reader: %v", err)
	}
	v, err := results.DiscoverVersion(gzipr)
	if err != nil {
		t.Fatalf("failed to discover version: %v", err)
	}
	_, err = r.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal("Could not seek to 0")
	}
	err = gzipr.Reset(r)
	if err != nil {
		t.Fatalf("failed to reset the gzip reader: %v", err)
	}
	a := results.NewReaderWithVersion(gzipr, v)
	if err != nil {
		t.Fatalf("Failed to open Reader: %v", err)
	}
	return a
}

var versions = []*version{
	&version{0, 8},
	&version{0, 9},
	&version{0, 10},
}

func TestServerGroup(t *testing.T) {
	testCases := []struct {
		testName       string
		version        *version
		expectedGroups int
	}{
		{
			testName:       "extract api group versions (not available)",
			version:        &version{0, 8},
			expectedGroups: 0,
		},
		{
			testName:       "extract api group versions",
			version:        &version{0, 9},
			expectedGroups: 14,
		},
		{
			testName:       "extract version",
			version:        &version{0, 10},
			expectedGroups: 14,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName+" "+tc.version.String(), func(t *testing.T) {
			reader := MustGetReader(tc.version.path(), t)
			groups := metav1.APIGroupList{}
			err := reader.WalkFiles(func(path string, info os.FileInfo, err error) error {
				return results.ExtractFileIntoStruct(reader.ServerGroupsFile(), path, info, &groups)
			})
			if err != nil {
				t.Fatalf("Got an error looking for server groups: %v", err)
			}
			if len(groups.Groups) != tc.expectedGroups {
				t.Fatalf("Expected %v groups but found %v", tc.expectedGroups, len(groups.Groups))
			}
		})
	}
}

func TestServerVersion(t *testing.T) {
	testCases := []struct {
		testName        string
		version         *version
		expectedVersion string
	}{
		{
			testName:        "extract version",
			version:         &version{0, 8},
			expectedVersion: "1.7",
		},
		{
			testName:        "extract version",
			version:         &version{0, 9},
			expectedVersion: "1.8",
		},
		{
			testName:        "extract version",
			version:         &version{0, 10},
			expectedVersion: "1.8",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testName+" "+tc.version.String(), func(t *testing.T) {
			reader := MustGetReader(tc.version.path(), t)
			k8sInfo := k8sver.Info{}
			err := reader.WalkFiles(func(path string, info os.FileInfo, err error) error {
				return results.ExtractFileIntoStruct(reader.ServerVersionFile(), path, info, &k8sInfo)
			})
			if err != nil {
				t.Fatalf("got an error walking files: %v", err)
			}
			actual := k8sInfo.Major + "." + k8sInfo.Minor
			if actual != tc.expectedVersion {
				t.Fatalf("Versions don't match. Expected: %v, actual: %v", tc.expectedVersion, actual)
			}
		})
	}
}

func TestSonobuoyVersion(t *testing.T) {
	testCases := []struct {
		testName        string
		version         *version
		expectedVersion string
	}{
		{
			testName:        "extract version",
			version:         &version{0, 8},
			expectedVersion: "v0.8",
		},
		{
			testName:        "extract version",
			version:         &version{0, 9},
			expectedVersion: "v0.9",
		},
		{
			testName:        "extract version",
			version:         &version{0, 10},
			expectedVersion: "v0.10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName+" "+tc.version.String(), func(t *testing.T) {
			reader := MustGetReader(tc.version.path(), t)
			if tc.expectedVersion != reader.Version {
				t.Fatalf("Expected %v but found %v", tc.expectedVersion, reader.Version)
			}
		})
	}
}

func TestNonNamespacedResources(t *testing.T) {
	testCases := []struct {
		testName string
		version  *version
		expected []string
	}{
		{
			testName: "nodes",
			version:  &version{0, 8},
			expected: []string{
				"ip-10-0-15-53.us-west-2.compute.internal",
				"ip-10-0-19-88.us-west-2.compute.internal",
				"ip-10-0-27-65.us-west-2.compute.internal",
			},
		},
		{
			testName: "nodes",
			version:  &version{0, 9},
			expected: []string{
				"ip-10-0-15-53.us-west-2.compute.internal",
				"ip-10-0-19-88.us-west-2.compute.internal",
				"ip-10-0-27-65.us-west-2.compute.internal",
			},
		},
		{
			testName: "nodes",
			version:  &version{0, 10},
			expected: []string{
				"ip-10-0-26-239.us-west-2.compute.internal",
				"ip-10-0-9-16.us-west-2.compute.internal",
				"ip-10-0-9-206.us-west-2.compute.internal",
			},
		},
	}

	// Run the test for every version
	for _, tc := range testCases {
		t.Run(tc.testName+" "+tc.version.String(), func(t *testing.T) {
			reader := MustGetReader(tc.version.path(), t)
			nodes := make([]*v1.Node, 0)
			err := reader.WalkFiles(func(path string, info os.FileInfo, err error) error {
				return results.ExtractFileIntoStruct(reader.NodesFile(), path, info, &nodes)
			})
			for _, n := range nodes {
				fmt.Println(n.Name)
			}
			if err != nil {
				t.Fatalf("Got an error walking files: %v", err)
			}
			if len(nodes) != len(tc.expected) {
				t.Fatalf("expected %v nodes found %v", len(tc.expected), len(nodes))
			}
		})
	}
}

func TestNamespacedResources(t *testing.T) {
	testCases := []struct {
		testName         string
		namespace        string
		version          *version
		expectedServices []string
	}{
		{
			testName:         "no services in the default namespace",
			namespace:        "default",
			version:          &version{0, 8},
			expectedServices: []string{},
		},
		{
			testName:         "no services in the kube-system namespace",
			namespace:        "kube-system",
			version:          &version{0, 8},
			expectedServices: []string{},
		},
		{
			testName:         "no services in the default namespace",
			namespace:        "default",
			version:          &version{0, 9},
			expectedServices: []string{"kubernetes"},
		},
		{
			testName:         "no services in the kube-system namespace",
			namespace:        "kube-system",
			version:          &version{0, 9},
			expectedServices: []string{"calico-etcd", "kube-dns", "kubernetes-dashboard"},
		},
		{
			testName:         "no services in the default namespace",
			namespace:        "default",
			version:          &version{0, 10},
			expectedServices: []string{"kubernetes"},
		},
		{
			testName:         "some services in the kube-system namespace",
			namespace:        "kube-system",
			version:          &version{0, 10},
			expectedServices: []string{"kube-dns", "kube-proxy", "kube-apiserver"},
		},
	}

	// Run the test for every version
	for _, tc := range testCases {
		t.Run(tc.testName+" "+tc.version.String(), func(t *testing.T) {
			reader := MustGetReader(tc.version.path(), t)
			svcs := make([]*v1.Service, 0)
			err := reader.WalkFiles(func(path string, info os.FileInfo, err error) error {
				return results.ExtractFileIntoStruct(filepath.Join(reader.NamespacedResources(), tc.namespace, "Services.json"), path, info, &svcs)
			})
			if err != nil {
				t.Fatalf("Got an error walking files: %v", err)
			}
			if len(svcs) != len(tc.expectedServices) {
				t.Fatalf("expected %v services found %v", len(tc.expectedServices), len(svcs))
			}
		})
	}
}

func TestExtractBytes(t *testing.T) {
	testCases := []struct {
		testName       string
		version        *version
		expectedOutput string
	}{
		{
			testName:       "Extract config.json from an archive",
			version:        &version{0, 10},
			expectedOutput: `{"Description":"EXAMPLE","UUID":"0659bf2a-80db-48ca-935b-8d30dbefa14e","Version":"v0.10.0-4-g8100397","ResultsDir":"/tmp/sonobuoy","Kubeconfig":"","Resources":["CertificateSigningRequests","ClusterRoleBindings","ClusterRoles","ComponentStatuses","CustomResourceDefinitions","Nodes","PersistentVolumes","PodSecurityPolicies","ServerVersion","StorageClasses","ConfigMaps","DaemonSets","Deployments","Endpoints","Events","HorizontalPodAutoscalers","Ingresses","Jobs","LimitRanges","PersistentVolumeClaims","Pods","PodDisruptionBudgets","PodTemplates","ReplicaSets","ReplicationControllers","ResourceQuotas","RoleBindings","Roles","ServerGroups","ServiceAccounts","Services","StatefulSets"],"Filters":{"Namespaces":".*","LabelSelector":""},"Limits":{"PodLogs":{"LimitSize":"","LimitTime":""}},"Server":{"bindaddress":"0.0.0.0","bindport":8080,"advertiseaddress":"sonobuoy-master:8080","timeoutseconds":5400},"Plugins":[{"name":"systemd_logs"},{"name":"e2e"}],"PluginSearchPath":["./plugins.d","/etc/sonobuoy/plugins.d","~/sonobuoy/plugins.d"],"PluginNamespace":"heptio-sonobuoy","LoadedPlugins":[{"Definition":{"Driver":"DaemonSet","Name":"systemd_logs","ResultType":"systemd_logs","Template":{"Name":"plugin","ParseName":"plugin","Root":{"NodeType":11,"Pos":0,"Nodes":[{"NodeType":0,"Pos":0,"Text":"YXBpVmVyc2lvbjogZXh0ZW5zaW9ucy92MWJldGExCmtpbmQ6IERhZW1vblNldAptZXRhZGF0YToKICBhbm5vdGF0aW9uczoKICAgIHNvbm9idW95LWRyaXZlcjogRGFlbW9uU2V0CiAgICBzb25vYnVveS1wbHVnaW46IHN5c3RlbWRfbG9ncwogICAgc29ub2J1b3ktcmVzdWx0LXR5cGU6IHN5c3RlbWRfbG9ncwogIGxhYmVsczoKICAgIGNvbXBvbmVudDogc29ub2J1b3kKICAgIHNvbm9idW95LXJ1bjogJw=="},{"NodeType":1,"Pos":231,"Line":10,"Pipe":{"NodeType":14,"Pos":231,"Line":10,"Decl":null,"Cmds":[{"NodeType":4,"Pos":231,"Args":[{"NodeType":8,"Pos":231,"Ident":["SessionID"]}]}]}},{"NodeType":0,"Pos":243,"Text":"JwogICAgdGllcjogYW5hbHlzaXMKICBuYW1lOiBzeXN0ZW1kLWxvZ3MKICBuYW1lc3BhY2U6ICc="},{"NodeType":1,"Pos":301,"Line":13,"Pipe":{"NodeType":14,"Pos":301,"Line":13,"Decl":null,"Cmds":[{"NodeType":4,"Pos":301,"Args":[{"NodeType":8,"Pos":301,"Ident":["Namespace"]}]}]}},{"NodeType":0,"Pos":313,"Text":"JwpzcGVjOgogIHNlbGVjdG9yOgogICAgbWF0Y2hMYWJlbHM6CiAgICAgIHNvbm9idW95LXJ1bjogJw=="},{"NodeType":1,"Pos":373,"Line":17,"Pipe":{"NodeType":14,"Pos":373,"Line":17,"Decl":null,"Cmds":[{"NodeType":4,"Pos":373,"Args":[{"NodeType":8,"Pos":373,"Ident":["SessionID"]}]}]}},{"NodeType":0,"Pos":385,"Text":"JwogIHRlbXBsYXRlOgogICAgbWV0YWRhdGE6CiAgICAgIGxhYmVsczoKICAgICAgICBjb21wb25lbnQ6IHNvbm9idW95CiAgICAgICAgc29ub2J1b3ktcnVuOiAn"},{"NodeType":1,"Pos":480,"Line":22,"Pipe":{"NodeType":14,"Pos":480,"Line":22,"Decl":null,"Cmds":[{"NodeType":4,"Pos":480,"Args":[{"NodeType":8,"Pos":480,"Ident":["SessionID"]}]}]}},{"NodeType":0,"Pos":492,"Text":"JwogICAgICAgIHRpZXI6IGFuYWx5c2lzCiAgICBzcGVjOgogICAgICBjb250YWluZXJzOgogICAgICAtIGNvbW1hbmQ6CiAgICAgICAgLSBzaAogICAgICAgIC0gLWMKICAgICAgICAtIC9nZXRfc3lzdGVtZF9sb2dzLnNoICYmIHNsZWVwIDM2MDAKICAgICAgICBlbnY6CiAgICAgICAgLSBuYW1lOiBOT0RFX05BTUUKICAgICAgICAgIHZhbHVlRnJvbToKICAgICAgICAgICAgZmllbGRSZWY6CiAgICAgICAgICAgICAgZmllbGRQYXRoOiBzcGVjLm5vZGVOYW1lCiAgICAgICAgLSBuYW1lOiBSRVNVTFRTX0RJUgogICAgICAgICAgdmFsdWU6IC90bXAvcmVzdWx0cwogICAgICAgIC0gbmFtZTogQ0hST09UX0RJUgogICAgICAgICAgdmFsdWU6IC9ub2RlCiAgICAgICAgaW1hZ2U6IGdjci5pby9oZXB0aW8taW1hZ2VzL3Nvbm9idW95LXBsdWdpbi1zeXN0ZW1kLWxvZ3M6bGF0ZXN0CiAgICAgICAgaW1hZ2VQdWxsUG9saWN5OiBBbHdheXMKICAgICAgICBuYW1lOiBzb25vYnVveS1zeXN0ZW1kLWxvZ3MtY29uZmlnLQ=="},{"NodeType":1,"Pos":1011,"Line":41,"Pipe":{"NodeType":14,"Pos":1011,"Line":41,"Decl":null,"Cmds":[{"NodeType":4,"Pos":1011,"Args":[{"NodeType":8,"Pos":1011,"Ident":["SessionID"]}]}]}},{"NodeType":0,"Pos":1023,"Text":"CiAgICAgICAgc2VjdXJpdHlDb250ZXh0OgogICAgICAgICAgcHJpdmlsZWdlZDogdHJ1ZQogICAgICAgIHZvbHVtZU1vdW50czoKICAgICAgICAtIG1vdW50UGF0aDogL3RtcC9yZXN1bHRzCiAgICAgICAgICBuYW1lOiByZXN1bHRzCiAgICAgICAgICByZWFkT25seTogZmFsc2UKICAgICAgICAtIG1vdW50UGF0aDogL25vZGUKICAgICAgICAgIG5hbWU6IHJvb3QKICAgICAgICAgIHJlYWRPbmx5OiBmYWxzZQogICAgICAtIGNvbW1hbmQ6CiAgICAgICAgLSBzaAogICAgICAgIC0gLWMKICAgICAgICAtIC9zb25vYnVveSB3b3JrZXIgc2luZ2xlLW5vZGUgLXYgNSAtLWxvZ3Rvc3RkZXJyICYmIHNsZWVwIDM2MDAKICAgICAgICBlbnY6CiAgICAgICAgLSBuYW1lOiBOT0RFX05BTUUKICAgICAgICAgIHZhbHVlRnJvbToKICAgICAgICAgICAgZmllbGRSZWY6CiAgICAgICAgICAgICAgZmllbGRQYXRoOiBzcGVjLm5vZGVOYW1lCiAgICAgICAgLSBuYW1lOiBSRVNVTFRTX0RJUgogICAgICAgICAgdmFsdWU6IC90bXAvcmVzdWx0cwogICAgICAgIC0gbmFtZTogTUFTVEVSX1VSTAogICAgICAgICAgdmFsdWU6ICc="},{"NodeType":1,"Pos":1597,"Line":63,"Pipe":{"NodeType":14,"Pos":1597,"Line":63,"Decl":null,"Cmds":[{"NodeType":4,"Pos":1597,"Args":[{"NodeType":8,"Pos":1597,"Ident":["MasterAddress"]}]}]}},{"NodeType":0,"Pos":1613,"Text":"JwogICAgICAgIC0gbmFtZTogUkVTVUxUX1RZUEUKICAgICAgICAgIHZhbHVlOiBzeXN0ZW1kX2xvZ3MKICAgICAgICBpbWFnZTogZ2NyLmlvL2hlcHRpby1pbWFnZXMvc29ub2J1b3k6bWFzdGVyCiAgICAgICAgaW1hZ2VQdWxsUG9saWN5OiBBbHdheXMKICAgICAgICBuYW1lOiBzb25vYnVveS13b3JrZXIKICAgICAgICB2b2x1bWVNb3VudHM6CiAgICAgICAgLSBtb3VudFBhdGg6IC90bXAvcmVzdWx0cwogICAgICAgICAgbmFtZTogcmVzdWx0cwogICAgICAgICAgcmVhZE9ubHk6IGZhbHNlCiAgICAgIGRuc1BvbGljeTogQ2x1c3RlckZpcnN0V2l0aEhvc3ROZXQKICAgICAgaG9zdElQQzogdHJ1ZQogICAgICBob3N0TmV0d29yazogdHJ1ZQogICAgICBob3N0UElEOiB0cnVlCiAgICAgIHRvbGVyYXRpb25zOgogICAgICAtIGVmZmVjdDogTm9TY2hlZHVsZQogICAgICAgIGtleTogbm9kZS1yb2xlLmt1YmVybmV0ZXMuaW8vbWFzdGVyCiAgICAgICAgb3BlcmF0b3I6IEV4aXN0cwogICAgICAtIGtleTogQ3JpdGljYWxBZGRvbnNPbmx5CiAgICAgICAgb3BlcmF0b3I6IEV4aXN0cwogICAgICB2b2x1bWVzOgogICAgICAtIGVtcHR5RGlyOiB7fQogICAgICAgIG5hbWU6IHJlc3VsdHMKICAgICAgLSBob3N0UGF0aDoKICAgICAgICAgIHBhdGg6IC8KICAgICAgICBuYW1lOiByb290Cg=="}]}}},"DfnTemplateData":{"SessionID":"a0cf5d7168f24710","MasterAddress":"http://sonobuoy-master:8080/api/v1/results/by-node","Namespace":"heptio-sonobuoy"}},{"Definition":{"Driver":"Job","Name":"e2e","ResultType":"e2e","Template":{"Name":"plugin","ParseName":"plugin","Root":{"NodeType":11,"Pos":0,"Nodes":[{"NodeType":0,"Pos":0,"Text":"YXBpVmVyc2lvbjogdjEKa2luZDogUG9kCm1ldGFkYXRhOgogIGFubm90YXRpb25zOgogICAgc29ub2J1b3ktZHJpdmVyOiBKb2IKICAgIHNvbm9idW95LXBsdWdpbjogZTJlCiAgICBzb25vYnVveS1yZXN1bHQtdHlwZTogZTJlCiAgbGFiZWxzOgogICAgY29tcG9uZW50OiBzb25vYnVveQogICAgc29ub2J1b3ktcnVuOiAn"},{"NodeType":1,"Pos":185,"Line":10,"Pipe":{"NodeType":14,"Pos":185,"Line":10,"Decl":null,"Cmds":[{"NodeType":4,"Pos":185,"Args":[{"NodeType":8,"Pos":185,"Ident":["SessionID"]}]}]}},{"NodeType":0,"Pos":197,"Text":"JwogICAgdGllcjogYW5hbHlzaXMKICBuYW1lOiBzb25vYnVveS1lMmUtam9iLQ=="},{"NodeType":1,"Pos":245,"Line":12,"Pipe":{"NodeType":14,"Pos":245,"Line":12,"Decl":null,"Cmds":[{"NodeType":4,"Pos":245,"Args":[{"NodeType":8,"Pos":245,"Ident":["SessionID"]}]}]}},{"NodeType":0,"Pos":257,"Text":"CiAgbmFtZXNwYWNlOiAn"},{"NodeType":1,"Pos":274,"Line":13,"Pipe":{"NodeType":14,"Pos":274,"Line":13,"Decl":null,"Cmds":[{"NodeType":4,"Pos":274,"Args":[{"NodeType":8,"Pos":274,"Ident":["Namespace"]}]}]}},{"NodeType":0,"Pos":286,"Text":"JwpzcGVjOgogIGNvbnRhaW5lcnM6CiAgLSBlbnY6CiAgICAtIG5hbWU6IEUyRV9GT0NVUwogICAgICB2YWx1ZTogUG9kcyBzaG91bGQgYmUgc3VibWl0dGVkIGFuZCByZW1vdmVkCiAgICBpbWFnZTogZ2NyLmlvL2hlcHRpby1pbWFnZXMva3ViZS1jb25mb3JtYW5jZTpsYXRlc3QKICAgIGltYWdlUHVsbFBvbGljeTogQWx3YXlzCiAgICBuYW1lOiBlMmUKICAgIHZvbHVtZU1vdW50czoKICAgIC0gbW91bnRQYXRoOiAvdG1wL3Jlc3VsdHMKICAgICAgbmFtZTogcmVzdWx0cwogICAgICByZWFkT25seTogZmFsc2UKICAtIGNvbW1hbmQ6CiAgICAtIHNoCiAgICAtIC1jCiAgICAtIC9zb25vYnVveSB3b3JrZXIgZ2xvYmFsIC12IDUgLS1sb2d0b3N0ZGVycgogICAgZW52OgogICAgLSBuYW1lOiBOT0RFX05BTUUKICAgICAgdmFsdWVGcm9tOgogICAgICAgIGZpZWxkUmVmOgogICAgICAgICAgZmllbGRQYXRoOiBzcGVjLm5vZGVOYW1lCiAgICAtIG5hbWU6IFJFU1VMVFNfRElSCiAgICAgIHZhbHVlOiAvdG1wL3Jlc3VsdHMKICAgIC0gbmFtZTogTUFTVEVSX1VSTAogICAgICB2YWx1ZTogJw=="},{"NodeType":1,"Pos":847,"Line":38,"Pipe":{"NodeType":14,"Pos":847,"Line":38,"Decl":null,"Cmds":[{"NodeType":4,"Pos":847,"Args":[{"NodeType":8,"Pos":847,"Ident":["MasterAddress"]}]}]}},{"NodeType":0,"Pos":863,"Text":"JwogICAgLSBuYW1lOiBSRVNVTFRfVFlQRQogICAgICB2YWx1ZTogZTJlCiAgICBpbWFnZTogZ2NyLmlvL2hlcHRpby1pbWFnZXMvc29ub2J1b3k6bWFzdGVyCiAgICBpbWFnZVB1bGxQb2xpY3k6IEFsd2F5cwogICAgbmFtZTogc29ub2J1b3ktd29ya2VyCiAgICB2b2x1bWVNb3VudHM6CiAgICAtIG1vdW50UGF0aDogL3RtcC9yZXN1bHRzCiAgICAgIG5hbWU6IHJlc3VsdHMKICAgICAgcmVhZE9ubHk6IGZhbHNlCiAgcmVzdGFydFBvbGljeTogTmV2ZXIKICBzZXJ2aWNlQWNjb3VudE5hbWU6IHNvbm9idW95LXNlcnZpY2VhY2NvdW50CiAgdG9sZXJhdGlvbnM6CiAgLSBlZmZlY3Q6IE5vU2NoZWR1bGUKICAgIGtleTogbm9kZS1yb2xlLmt1YmVybmV0ZXMuaW8vbWFzdGVyCiAgICBvcGVyYXRvcjogRXhpc3RzCiAgLSBrZXk6IENyaXRpY2FsQWRkb25zT25seQogICAgb3BlcmF0b3I6IEV4aXN0cwogIHZvbHVtZXM6CiAgLSBlbXB0eURpcjoge30KICAgIG5hbWU6IHJlc3VsdHMK"}]}}},"DfnTemplateData":{"SessionID":"b803642b9d884c42","MasterAddress":"http://sonobuoy-master:8080/api/v1/results/global","Namespace":"heptio-sonobuoy"}}]}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName+" "+tc.version.String(), func(t *testing.T) {
			reader := MustGetReader(tc.version.path(), t)
			var buf bytes.Buffer
			err := reader.WalkFiles(func(path string, info os.FileInfo, err error) error {
				return results.ExtractBytes(filepath.Join(reader.Metadata(), "config.json"), path, info, &buf)
			})
			if err != nil {
				t.Fatalf("Got an error walking files: %v", err)
			}
			if tc.expectedOutput != string(buf.Bytes()) {
				t.Fatalf("expected %v string, found %v", tc.expectedOutput, buf.String())
			}
		})

	}
}
