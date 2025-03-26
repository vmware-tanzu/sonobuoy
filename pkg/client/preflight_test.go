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

package client

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	testhook "github.com/sirupsen/logrus/hooks/test"
	apicorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
)

func TestVersionCheck(t *testing.T) {
	testHook := &testhook.Hook{}
	logrus.AddHook(testHook)
	logrus.SetOutput(io.Discard)

	serverAtVersion := func(major, minor, git string) *fakeServerVersionInterface {
		return &fakeServerVersionInterface{
			version: k8sversion.Info{
				Major:      major,
				Minor:      minor,
				GitVersion: git,
			},
		}
	}

	brokenServerVersion := &fakeServerVersionInterface{
		err: errors.New("test err"),
	}

	testCases := []struct {
		desc          string
		client        discovery.ServerVersionInterface
		min           *version.Version
		max           *version.Version
		expectErr     string
		expectWarning string
	}{
		{
			desc:   "Simple case",
			client: serverAtVersion("1", "0", "1.0.1"),
			min:    version.Must(version.NewVersion("1.0.0")),
			max:    version.Must(version.NewVersion("2.0.0")),
		}, {
			desc:      "Error getting version",
			client:    brokenServerVersion,
			min:       version.Must(version.NewVersion("1.0.0")),
			max:       version.Must(version.NewVersion("2.0.0")),
			expectErr: "failed to retrieve server version: test err",
		}, {
			desc:      "Below min version",
			client:    serverAtVersion("1", "2", "1.2.3"),
			min:       version.Must(version.NewVersion("2.0.0")),
			max:       version.Must(version.NewVersion("3.0.0")),
			expectErr: "minimum supported Kubernetes version is 2.0.0, but the server version is 1.2.3",
		}, {
			desc:          "Above max version",
			client:        serverAtVersion("1", "2", "1.2.3"),
			min:           version.Must(version.NewVersion("1.1.0")),
			max:           version.Must(version.NewVersion("1.2.0")),
			expectWarning: "The maximum supported Kubernetes version is 1.2.0, but the server version is 1.2.3. Sonobuoy will continue but unexpected results may occur.",
		}, {
			desc:   "Equal to min version",
			client: serverAtVersion("1", "2", "1.2.3"),
			min:    version.Must(version.NewVersion("1.2.3")),
			max:    version.Must(version.NewVersion("2.0.0")),
		}, {
			desc:   "Equal to max version",
			client: serverAtVersion("1", "2", "1.2.3"),
			min:    version.Must(version.NewVersion("1.0.0")),
			max:    version.Must(version.NewVersion("1.2.3")),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := versionCheck(tc.client, tc.min, tc.max)
			if err != nil && len(tc.expectErr) == 0 {
				t.Fatalf("Expected nil error but got %q", err)
			}
			if err == nil && len(tc.expectErr) > 0 {
				t.Fatalf("Expected error %q but got nil", tc.expectErr)
			}
			if err != nil && fmt.Sprint(err) != tc.expectErr {
				t.Fatalf("Expected error to be %q but got %q", tc.expectErr, err)
			}
			if len(tc.expectWarning) > 0 {
				last := testHook.LastEntry()
				if last == nil {
					t.Errorf("expected warning entry, got nothing")
				} else if last.Level != logrus.WarnLevel {
					t.Errorf("expected level %v, got %v", logrus.WarnLevel, last.Level)
				} else if last.Message != tc.expectWarning {
					t.Errorf("expected warning message %q, got %q", tc.expectWarning, last.Message)
				}
			}
		})
	}
}

// fakeServerVersionInterface is used as a test implementation as
// discovery.ServerVersionInterface.
type fakeServerVersionInterface struct {
	err     error
	version k8sversion.Info
}

func (f *fakeServerVersionInterface) ServerVersion() (*k8sversion.Info, error) {
	return &f.version, f.err
}

func TestDNSCheck(t *testing.T) {
	testCases := []struct {
		desc         string
		lister       listFunc
		dnsNamespace string
		dnsLabels    []string
		expectErr    string
	}{
		{
			desc: "Needs only a single pod",
			lister: func(context.Context, metav1.ListOptions) (*apicorev1.PodList, error) {
				return &apicorev1.PodList{
					Items: []apicorev1.Pod{
						{},
					},
				}, nil
			},
			dnsNamespace: "dns-namespace",
			dnsLabels:    []string{"foo"},
		}, {
			desc: "Multiple pods OK",
			lister: func(context.Context, metav1.ListOptions) (*apicorev1.PodList, error) {
				return &apicorev1.PodList{
					Items: []apicorev1.Pod{
						{},
						{},
					},
				}, nil
			},
			dnsNamespace: "dns-namespace",
			dnsLabels:    []string{"foo"},
		}, {
			desc: "Requires at least one pod",
			lister: func(context.Context, metav1.ListOptions) (*apicorev1.PodList, error) {
				return &apicorev1.PodList{}, nil
			},
			dnsNamespace: "dns-namespace",
			dnsLabels:    []string{"foo"},
			expectErr:    "no dns pods found with the labels [foo] in namespace dns-namespace",
		}, {
			desc: "Skipped if no labels required",
			lister: func(context.Context, metav1.ListOptions) (*apicorev1.PodList, error) {
				return &apicorev1.PodList{}, nil
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := dnsCheck(tc.lister, tc.dnsNamespace, tc.dnsLabels...)
			if err != nil && len(tc.expectErr) == 0 {
				t.Fatalf("Expected nil error but got %q", err)
			}
			if err == nil && len(tc.expectErr) > 0 {
				t.Fatalf("Expected error %q but got nil", tc.expectErr)
			}
			if err != nil && fmt.Sprint(err) != tc.expectErr {
				t.Fatalf("Expected error to be %q but got %q", tc.expectErr, err)
			}
		})
	}
}

func TestNamespaceCheck(t *testing.T) {
	testCases := []struct {
		desc      string
		getter    nsGetFunc
		ns        string
		expectErr string
	}{
		{
			desc: "Namespace and no error indicates it exists",
			getter: func(context.Context, string, metav1.GetOptions) (*apicorev1.Namespace, error) {
				return &apicorev1.Namespace{}, nil
			},
			expectErr: "namespace already exists",
		}, {
			desc: "Random error bubbled up",
			getter: func(context.Context, string, metav1.GetOptions) (*apicorev1.Namespace, error) {
				return nil, errors.New("test")
			},
			expectErr: "error checking for namespace: test",
		}, {
			desc: "Does not exist errors pass the check",
			getter: func(context.Context, string, metav1.GetOptions) (*apicorev1.Namespace, error) {
				return nil, &statusErr{err: "test", status: metav1.Status{Reason: metav1.StatusReasonNotFound}}
			},
		}, {
			desc: "Other API status errors still bubble up",
			getter: func(context.Context, string, metav1.GetOptions) (*apicorev1.Namespace, error) {
				return nil, &statusErr{err: "test", status: metav1.Status{Reason: metav1.StatusReasonBadRequest}}
			},
			expectErr: "error checking for namespace: test",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := nsCheck(tc.getter, tc.ns)
			if err != nil && len(tc.expectErr) == 0 {
				t.Fatalf("Expected nil error but got %q", err)
			}
			if err == nil && len(tc.expectErr) > 0 {
				t.Fatalf("Expected error %q but got nil", tc.expectErr)
			}
			if err != nil && fmt.Sprint(err) != tc.expectErr {
				t.Fatalf("Expected error to be %q but got %q", tc.expectErr, err)
			}
		})
	}
}

func TestPreflightChecksInvalidConfig(t *testing.T) {
	testcases := []struct {
		desc               string
		config             *PreflightConfig
		expectedErrorCount int
		expectedErrorMsgs  []string
	}{
		{
			desc:               "providing a nil config results in an error",
			config:             nil,
			expectedErrorCount: 1,
			expectedErrorMsgs:  []string{"nil PreflightConfig provided"},
		},
		{
			desc:               "providing an invalid config results in an error",
			config:             &PreflightConfig{},
			expectedErrorCount: 1,
			expectedErrorMsgs:  []string{"config validation failed"},
		},
	}

	c, err := NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		errors := c.PreflightChecks(tc.config)

		if len(errors) != tc.expectedErrorCount {
			t.Errorf("Unexpected number of errors, expected %d, got %d", tc.expectedErrorCount, len(errors))
		} else {
			for i, err := range errors {
				expectedErrorMsg := tc.expectedErrorMsgs[i]
				if !strings.Contains(err.Error(), expectedErrorMsg) {
					t.Errorf("Expected error to contain %q, got %q", expectedErrorMsg, err.Error())
				}
			}
		}
	}
}

type statusErr struct {
	err    string
	status metav1.Status
}

func (e *statusErr) Error() string {
	return e.err
}

func (e *statusErr) Status() metav1.Status {
	return e.status
}
