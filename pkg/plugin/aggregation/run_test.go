package aggregation

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"math/big"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/job"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	sonotime "github.com/vmware-tanzu/sonobuoy/pkg/time/timetest"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRunAndMonitorPlugin(t *testing.T) {
	// Dead simple plugin works for this test. No need to test daemonset/job specific logic so
	// a job plugin is much simpler to test against.
	testPlugin := &job.Plugin{
		Base: driver.Base{
			Definition: manifest.Manifest{
				SonobuoyConfig: manifest.SonobuoyConfig{
					PluginName: "myPlugin",
				},
			},
			Namespace: "testNS",
		},
	}
	testPluginExpectedResults := []plugin.ExpectedResult{
		{ResultType: "myPlugin", NodeName: "global"},
	}
	healthyPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
	}
	failingPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Reason: "Unschedulable"},
			},
		},
	}
	testCert, err := getTestCert()
	if err != nil {
		t.Fatalf("Could not generate test cert: %v", err)
	}
	testTimeout := 1 * time.Minute
	sonotime.UseShortAfter()
	defer sonotime.ResetAfter()

	testCases := []struct {
		desc string

		expectNumResults   int
		expectStillRunning bool
		forceResults       bool
		cancelContext      bool

		plugin           plugin.Interface
		expectedResults  []plugin.ExpectedResult
		podList          *corev1.PodList
		podCreationError error
	}{
		{
			desc:            "Continue monitoring if no results/errors",
			plugin:          testPlugin,
			expectedResults: testPluginExpectedResults,
			podList: &corev1.PodList{
				Items: []corev1.Pod{healthyPod},
			},
			expectStillRunning: true,
		}, {
			desc:            "Error launching plugin causes exit and plugin result",
			plugin:          testPlugin,
			expectedResults: testPluginExpectedResults,
			podList: &corev1.PodList{
				Items: []corev1.Pod{healthyPod},
			},
			podCreationError: errors.New("createPod error"),
			expectNumResults: 1,
		}, {
			desc:            "Failing plugin causes exit and plugin result",
			plugin:          testPlugin,
			expectedResults: testPluginExpectedResults,
			podList: &corev1.PodList{
				Items: []corev1.Pod{failingPod},
			},
			expectNumResults: 1,
		}, {
			desc:            "Plugin obtaining results in exits",
			plugin:          testPlugin,
			expectedResults: testPluginExpectedResults,
			podList: &corev1.PodList{
				Items: []corev1.Pod{healthyPod},
			},
			forceResults:     true,
			expectNumResults: 1,
		}, {
			desc:            "Context cancellation results in exit",
			plugin:          testPlugin,
			expectedResults: testPluginExpectedResults,
			podList: &corev1.PodList{
				Items: []corev1.Pod{healthyPod},
			},
			cancelContext:    true,
			expectNumResults: 0,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "sonobuoy-test")
			if err != nil {
				t.Fatalf("Failed to make temp directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)
			a := NewAggregator(tmpDir, tc.expectedResults)
			ctx, cancel := context.WithCancel(context.Background())

			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, tc.podList, nil
			})
			fclient.PrependReactor("create", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, tc.podCreationError
			})

			doneCh, timeoutCh := make(chan struct{}, 1), make(chan struct{}, 1)
			if tc.cancelContext {
				cancel()
			} else {
				// Max timeout for test to unblock.
				go func() {
					time.Sleep(2 * time.Second)
					timeoutCh <- struct{}{}
					cancel()
				}()
			}

			go func() {
				a.RunAndMonitorPlugin(ctx, testTimeout, tc.plugin, fclient, nil, "testname", testCert, &corev1.Pod{}, "", "/tmp/sonobuoy/results")
				doneCh <- struct{}{}
			}()

			if tc.forceResults {
				a.resultsMutex.Lock()
				a.Results["myPlugin/global"] = &plugin.Result{}
				a.resultsMutex.Unlock()
			}

			// Wait for completion/timeout and see which happens first.
			wasStillRunning := false
			select {
			case <-doneCh:
				t.Log("runAndMonitor is done")
			case <-timeoutCh:
				t.Log("runAndMonitor timed out")
				wasStillRunning = true
			}

			if len(a.Results) != tc.expectNumResults {
				t.Errorf("Expected %v results but found %v: %+v", tc.expectNumResults, len(a.Results), a.Results)
			}
			if wasStillRunning != tc.expectStillRunning {
				t.Errorf("Expected wasStillMonitoring %v but found %v", tc.expectStillRunning, wasStillRunning)
			}
		})
	}
}

func (cp *MockRunPlugin) Run(_ kubernetes.Interface, _ string, _ *tls.Certificate, _ *corev1.Pod, _, _ string) error {
	return nil
}

func (cp *MockRunPlugin) Cleanup(_ kubernetes.Interface) {
	cp.cleanedUp = true
}

func (cp *MockRunPlugin) Monitor(_ context.Context, _ kubernetes.Interface, _ []corev1.Node, _ chan<- *plugin.Result) {
	return
}

func (cp *MockRunPlugin) ExpectedResults(_ []corev1.Node) []plugin.ExpectedResult {
	return []plugin.ExpectedResult{}
}

func (cp *MockRunPlugin) FillTemplate(_ string, _ *tls.Certificate) ([]byte, error) {
	return []byte{}, nil
}

func (cp *MockRunPlugin) GetName() string {
	return cp.name
}

func (cp *MockRunPlugin) SkipCleanup() bool {
	return cp.skipCleanup
}

func (cp *MockRunPlugin) GetResultFormat() string {
	return ""
}

func (cp *MockRunPlugin) GetResultFiles() []string {
	return []string{}
}

func (cp *MockRunPlugin) GetDescription() string {
	return "A mock plugin used for testing purposes"
}

func (cp *MockRunPlugin) GetSourceURL() string {
	return ""
}

func (cp *MockRunPlugin) GetOrder() int {
	return cp.order
}

type MockRunPlugin struct {
	skipCleanup bool
	cleanedUp   bool
	order       int
	name        string
}

func TestGetOrderedPlugins(t *testing.T) {
	testCases := []struct {
		desc     string
		input    []plugin.Interface
		expected [][]plugin.Interface
	}{
		{
			desc: "Ensure lists of plugin.Interface are ordered by plugin order",
			input: []plugin.Interface{
				&MockRunPlugin{
					order: 1,
				},
				&MockRunPlugin{
					order: 0,
				},
				&MockRunPlugin{
					order: 1,
				},
				&MockRunPlugin{
					order: 1,
				},
				&MockRunPlugin{
					order: 2,
				},
			},
			expected: [][]plugin.Interface{
				{
					&MockRunPlugin{
						order: 0,
					},
				},
				{
					&MockRunPlugin{
						order: 1,
					},
					&MockRunPlugin{
						order: 1,
					},
					&MockRunPlugin{
						order: 1,
					},
				},
				{
					&MockRunPlugin{
						order: 2,
					},
				},
			},
		},
		{
			desc: "Ensure lists of plugin.Interface are sorted by plugin name",
			input: []plugin.Interface{
				&MockRunPlugin{
					order: 0,
					name:  "Test plugin",
				},
				&MockRunPlugin{
					order: 0,
					name:  "Another ordering test plugin",
				},
				&MockRunPlugin{
					order: 0,
					name:  "Ordering test plugin",
				},
			},
			expected: [][]plugin.Interface{
				{
					&MockRunPlugin{
						order: 0,
						name:  "Another ordering test plugin",
					},
					&MockRunPlugin{
						order: 0,
						name:  "Ordering test plugin",
					},
					&MockRunPlugin{
						order: 0,
						name:  "Test plugin",
					},
				},
			},
		},
		{
			desc:     "Ensure empty slice of plugin.Interface returns correct result",
			input:    []plugin.Interface{},
			expected: [][]plugin.Interface{},
		},
		{
			desc:     "Ensure null input returns correct result",
			input:    nil,
			expected: [][]plugin.Interface{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if !reflect.DeepEqual(getOrderedPlugins(tc.input), tc.expected) {
				t.Errorf("Expected output: %v", tc.expected)
			}
		})
	}
}

func getTestCert() (*tls.Certificate, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(0),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create certificate")
	}

	return &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privKey,
	}, nil
}

func TestCleanup(t *testing.T) {
	createPlugin := func(skipCleanup bool) *MockRunPlugin {
		return &MockRunPlugin{
			skipCleanup: skipCleanup,
			cleanedUp:   false,
		}
	}

	testCases := []struct {
		desc                    string
		plugins                 []*MockRunPlugin
		expectedCleanedUpValues []bool
	}{
		{
			desc:                    "plugins without skip cleanup are all cleaned up",
			plugins:                 []*MockRunPlugin{createPlugin(false), createPlugin(false)},
			expectedCleanedUpValues: []bool{true, true},
		},
		{
			desc:                    "plugins with skip cleanup are not cleaned up",
			plugins:                 []*MockRunPlugin{createPlugin(true), createPlugin(false), createPlugin(true)},
			expectedCleanedUpValues: []bool{false, true, false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var plugins = make([]plugin.Interface, len(tc.plugins))
			for i, p := range tc.plugins {
				plugins[i] = p
			}

			Cleanup(nil, plugins)

			for i, p := range tc.plugins {
				if p.cleanedUp != tc.expectedCleanedUpValues[i] {
					if p.cleanedUp {
						t.Error("Expected plugin not to be cleaned up")
					} else {
						t.Error("Expected plugin to be cleaned up")
					}
				}
			}
		})
	}
}
