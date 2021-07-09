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

			doneCh, timeoutCh := make(chan (struct{}), 1), make(chan (struct{}), 1)
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
				a.RunAndMonitorPlugin(ctx, testTimeout, tc.plugin, fclient, nil, "testname", testCert, &corev1.Pod{}, "")
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

type MockCleanupPlugin struct {
	skipCleanup bool
	cleanedUp   bool
}

func (cp *MockCleanupPlugin) Run(_ kubernetes.Interface, _ string, _ *tls.Certificate, _ *corev1.Pod, _ string) error {
	return nil
}

func (cp *MockCleanupPlugin) Cleanup(_ kubernetes.Interface) {
	cp.cleanedUp = true
}

func (cp *MockCleanupPlugin) Monitor(_ context.Context, _ kubernetes.Interface, _ []corev1.Node, _ chan<- *plugin.Result) {
	return
}

func (cp *MockCleanupPlugin) ExpectedResults(_ []corev1.Node) []plugin.ExpectedResult {
	return []plugin.ExpectedResult{}
}

func (cp *MockCleanupPlugin) FillTemplate(_ string, _ *tls.Certificate) ([]byte, error) {
	return []byte{}, nil
}

func (cp *MockCleanupPlugin) GetName() string {
	return "mock-cleanup-plugin"
}

func (cp *MockCleanupPlugin) SkipCleanup() bool {
	return cp.skipCleanup
}

func (cp *MockCleanupPlugin) GetResultFormat() string {
	return ""
}

func (cp *MockCleanupPlugin) GetResultFiles() []string {
	return []string{}
}

func (cp *MockCleanupPlugin) GetDescription() string {
	return "A mock plugin used for testing purposes"
}

func (cp *MockCleanupPlugin) GetSourceURL() string {
	return ""
}

func TestCleanup(t *testing.T) {
	createPlugin := func(skipCleanup bool) *MockCleanupPlugin {
		return &MockCleanupPlugin{
			skipCleanup: skipCleanup,
			cleanedUp:   false,
		}
	}

	testCases := []struct {
		desc                    string
		plugins                 []*MockCleanupPlugin
		expectedCleanedUpValues []bool
	}{
		{
			desc:                    "plugins without skip cleanup are all cleaned up",
			plugins:                 []*MockCleanupPlugin{createPlugin(false), createPlugin(false)},
			expectedCleanedUpValues: []bool{true, true},
		},
		{
			desc:                    "plugins with skip cleanup are not cleaned up",
			plugins:                 []*MockCleanupPlugin{createPlugin(true), createPlugin(false), createPlugin(true)},
			expectedCleanedUpValues: []bool{false, true, false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var plugins []plugin.Interface = make([]plugin.Interface, len(tc.plugins))
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
