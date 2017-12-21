package results_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"k8s.io/api/core/v1"

	"github.com/heptio/sonobuoy/results"
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
			r, err := results.OpenArchive(tc.version.path())
			if err != nil {
				t.Fatalf("error opening archive: %v", err)
			}
			groups := r.ServerGroups()
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
			r, err := results.OpenArchive(tc.version.path())
			if err != nil {
				t.Fatalf("error opening archive: %v", err)
			}
			info := r.ServerVersion()
			if info.Major+"."+info.Minor != tc.expectedVersion {
				t.Fatalf("Versions don't match. Expected: %v, actual: %v", tc.expectedVersion, info.Major+"."+info.Minor)
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
			r, err := results.OpenArchive(tc.version.path())
			if err != nil {
				t.Fatalf("error opening archive: %v", err)
			}
			if tc.expectedVersion != r.Version {
				t.Fatalf("Expected %v but found %v", tc.expectedVersion, r.Version)
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
			r, err := results.OpenArchive(tc.version.path())
			defer r.Close()
			if err != nil {
				t.Fatalf("Failed to open archive: %v", err)
			}
			nodes := make([]*v1.Node, 0)
			err = r.NonNamespacedResources(func(path string, info os.FileInfo, err error) error {
				//TODO(chuckha): not super happy the consumer has to have a path here to filter.
				// The other option is parsing enough of the data to figure out if they care about it.
				if strings.HasSuffix(path, results.NodesFile) {
					reader, ok := info.Sys().(io.Reader)
					if !ok {
						return errors.New("info.Sys() is not an io.Reader")
					}
					fileData, err := ioutil.ReadAll(reader)
					if err != nil {
						return errors.New(fmt.Sprintf("failed to read data: %v", err))
					}
					err = json.Unmarshal(fileData, &nodes)
					if err != nil {
						return errors.New(fmt.Sprintf("failed to unmarshal nodes: %v", err))
					}
				}
				return err
			})
			if err != nil {
				t.Fatalf("Got an error reading NamespacedResources: %v", err)
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
			archive, err := results.OpenArchive(tc.version.path())
			defer archive.Close()
			if err != nil {
				t.Fatalf("Failed to open Archive: %v", err)
			}
			svcs := make([]*v1.Service, 0)
			err = archive.NamespacedResources(func(path string, info os.FileInfo, err error) error {
				//TODO(chuckha): not super happy the consumer has to have a path here to filter.
				// The other option is parsing enough of the data to figure out if they care about it.
				if strings.HasSuffix(path, tc.namespace+"/Services.json") {
					reader, ok := info.Sys().(io.Reader)
					if !ok {
						return errors.New("info.Sys() is not an io.Reader")
					}
					fileData, err := ioutil.ReadAll(reader)
					if err != nil {
						return errors.New(fmt.Sprintf("failed to read data: %v", err))
					}
					services := []*v1.Service{}
					err = json.Unmarshal(fileData, &services)
					if err != nil {
						return errors.New(fmt.Sprintf("failed to unmarshal services: %v", err))
					}
					for _, service := range services {
						svcs = append(svcs, service)
					}
				}
				return err
			})
			if err != nil {
				t.Fatalf("Got an error reading NamespacedResources: %v", err)
			}
			if len(svcs) != len(tc.expectedServices) {
				t.Fatalf("expected %v services found %v", len(tc.expectedServices), len(svcs))
			}
		})
	}
}

// func TestMetadata(t *testing.T) {
// 	for _, resultsFilename := range allResultVersionFiles {

// 	}
// 	r, err := results.NewResults(testResultsFile)
// 	if err != nil {
// 		t.Fatalf("error getting NewResults: %v\n", err)
// 	}
// 	metadata := r.Metadata()
// 	if metadata.Config.UUID != "0659bf2a-80db-48ca-935b-8d30dbefa14e" {
// 		t.Error("Expected the correct UUID")
// 	}
// 	if len(metadata.QueryData) == 0 {
// 		t.Error("Expected some query data but didn't see any")
// 	}
// }

// func TestHosts(t *testing.T) {
// 	r, err := results.NewResults(testResultsFile)
// 	if err != nil {
// 		t.Fatalf("error getting NewResults: %v\n", err)
// 	}
// 	hostNames := []string{}
// 	r.Hosts(func(path string, info os.FileInfo, err error) error {
// 		if !info.IsDir() {
// 			return nil
// 		}
// 		hostNames = append(hostNames, info.Name())
// 		return err
// 	})
// 	found := false
// 	for _, name := range hostNames {
// 		found = name == "ip-10-0-26-239.us-west-2.compute.internal"
// 		if found {
// 			break
// 		}
// 	}
// 	if !found {
// 		t.Fatal("Expected to find a specific host")
// 	}
// }

// func TestServerVersion(t *testing.T) {
// 	r, err := results.NewResults(testResultsFile)
// 	if err != nil {
// 		t.Fatalf("error getting NewResults: %v\n", err)
// 	}
// 	version := r.ServerVersion()
// 	if version.Major != "1" && version.Minor != "8" {
// 		t.Fatalf("Expected 1.8 but got something else")
// 	}
// }

// func TestServerGroups(t *testing.T) {
// 	r, err := results.NewResults(testResultsFile)
// 	if err != nil {
// 		t.Fatalf("error getting NewResults: %v\n", err)
// 	}
// 	groups := r.ServerGroups()
// 	if groups.APIVersion != "v1" {
// 		t.Fatalf("Expected v1 got something else")
// 	}
// }
