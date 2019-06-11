package aggregation

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"testing"
	"time"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/job"
	sonotime "github.com/heptio/sonobuoy/pkg/time/timetest"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRunAndMonitorPlugin(t *testing.T) {
	// Dead simple plugin works for this test. No need to test daemonset/job specific logic so
	// a job plugin is much simpler to test against.
	testPlugin := &job.Plugin{
		Base: driver.Base{
			Definition: plugin.Definition{
				Name:       "myPlugin",
				ResultType: "myPlugin",
			},
			Namespace: "testNS",
		},
	}
	testPluginExpectedResults := []plugin.ExpectedResult{
		{ResultType: "myPlugin"},
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
			a := NewAggregator(".", tc.expectedResults)
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
				a.RunAndMonitorPlugin(ctx, tc.plugin, fclient, nil, "testname", testCert)
				doneCh <- struct{}{}
			}()

			if tc.forceResults {
				a.resultsMutex.Lock()
				a.Results["myPlugin"] = &plugin.Result{}
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
