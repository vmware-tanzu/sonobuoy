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
