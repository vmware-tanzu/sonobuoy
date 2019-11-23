/*
Copyright 2018 Heptio Inc.

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
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestLateErrors(t *testing.T) {
	quotes := []string{
		"WHAT ONE DOES WHEN FACED WITH THE TRUTH IS MORE DIFFICULT THAN YOU’D THINK.",
		"YOU ARE STRONGER THAN YOU BELIEVE. YOU HAVE GREATER POWERS THAN YOU KNOW.",
		"YOU LET THIS LITTLE THING TELL YOU WHAT TO DO?",
		"I’M WILLING TO FIGHT FOR THOSE WHO CANNOT FIGHT FOR THEMSELVES.",
	}
	bytestream := make(chan []byte)
	go func() {
		for _, quote := range quotes {
			bytestream <- []byte(quote)
		}
		close(bytestream)
	}()
	errc := make(chan error)

	reader := NewReader(bytestream, errc)

	// read the entire first message.
	mybuf := make([]byte, len(quotes[0]))

	n, err := reader.Read(mybuf)
	if err != nil {
		t.Fatalf("expected nil but got: %v", err)
	}
	if n != len(mybuf) {
		t.Fatalf("unexpected number of bytes read: %v", n)
	}

	go func() { errc <- errors.New("introduce an error") }()

	// We should see all the messages (3) and the error message (1) then get the error returned.
	errcount := 0
	for i := 0; i <= 4; i++ {
		_, err := reader.Read(mybuf)
		if err != nil && err != io.EOF {
			errcount++
		}
	}
	if errcount == 0 {
		t.Fatalf("Never saw an expected error.")
	}
}

func TestLogEarlyErrors(t *testing.T) {
	input := "sonobuoy will help you on your way to greatness"
	bytestream := make(chan []byte)
	go func() {
		defer close(bytestream)
		bytestream <- []byte(input)
	}()
	errc := make(chan error)
	go func() { errc <- errors.New("A seriously bad error") }()

	reader := NewReader(bytestream, errc)

	mybuf := make([]byte, 1024)
	errcount := 0
	for i := 0; i <= 5; i++ {
		_, err := reader.Read(mybuf)
		if err != nil && err != io.EOF {
			errcount++
		}
	}
	if errcount == 0 {
		t.Fatal("did not receive any errors but there should be one.")
	}
}

func TestLogReaderNoError(t *testing.T) {
	testcases := []struct {
		name          string
		input         []string
		bufsize       int
		expectedReads []string
	}{
		{
			name:          "tiny buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       1,
			expectedReads: []string{"H", "e", "l", "l", "o", " ", "w", "o", "r", "l", "d", " ", "0", ""},
		},
		{
			name:          "small buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       2,
			expectedReads: []string{"He", "ll", "o ", "wo", "rl", "d ", "0"},
		},
		{
			name:          "big buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       1000,
			expectedReads: []string{"Hello world 0"},
		},
		{
			name:          "exact buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       len("Hello world 0"),
			expectedReads: []string{"Hello world 0"},
		},
		{
			name: "big buffer, small messages",
			input: []string{
				"Once you start down the dark path, forever will it dominate your destiny.",
				"Luminous beings are we, not this crude matter.",
				"Fear is the path to the dark side. Fear leads to anger. Anger leads to hate. Hate leads to suffering.",
			},
			bufsize: 1024,
			expectedReads: []string{
				"Once you start down the dark path, forever will it dominate your destiny.",
				"Luminous beings are we, not this crude matter.",
				"Fear is the path to the dark side. Fear leads to anger. Anger leads to hate. Hate leads to suffering.",
			},
		},
		{
			name: "small buffer, big input",
			input: []string{
				"this is some log line",
				"this is another log line",
				"this is a third log line!!",
			},
			bufsize: 10,
			expectedReads: []string{
				"this is so",
				"me log lin",
				"e",
				"this is an",
				"other log ",
				"line",
				"this is a ",
				"third log ",
				"line!!",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bytestream := make(chan []byte)
			errc := make(chan error)

			go func(data chan []byte, e chan error, inputs []string) {
				for _, input := range inputs {
					data <- []byte(input)
				}
				close(data)
				errc <- io.EOF
			}(bytestream, errc, tc.input)
			reader := NewReader(bytestream, errc)
			mybuf := make([]byte, tc.bufsize)
			i := 0
			for ; ; i++ {
				n, err := reader.Read(mybuf)

				if err != nil && err != io.EOF {
					t.Fatalf("Expected no errors got %v", err)
				}
				if err == io.EOF {
					break
				}
				if n > len(mybuf) {
					t.Fatalf("n is too big: %v mybuf is only %v", n, len(mybuf))
				}
				if i >= len(tc.expectedReads) {
					t.Fatalf("Too many actual reads, not enough expected reads. BUF: %q", string(mybuf[:n]))
				}
				if len(mybuf[:n]) != len(tc.expectedReads[i]) {
					t.Errorf("Expected to read %v bytes, got %v buf: '%v' expected: '%v'", len(tc.expectedReads[i]), n, string(mybuf[:n]), tc.expectedReads[i])
				}
				if string(mybuf[:n]) != tc.expectedReads[i] {
					t.Errorf("Expected '%v' got '%v'", tc.expectedReads[i], string(mybuf[:n]))
				}

			}
			i++ // add one to i for the final read.
			if i < len(tc.expectedReads) {
				t.Fatalf("Expected Read to be called %v times but was only called %v times", len(tc.expectedReads), i)
			}
		})
	}
}

func TestLogReaderInvalidConfig(t *testing.T) {
	testcases := []struct {
		desc             string
		config           *LogConfig
		expectedError    bool
		expectedErrorMsg string
	}{
		{
			desc:             "providing a nil config results in an error",
			config:           nil,
			expectedError:    true,
			expectedErrorMsg: "nil LogConfig provided",
		},
		{
			desc:             "providing an invalid config results in an error",
			config:           &LogConfig{},
			expectedError:    true,
			expectedErrorMsg: "config validation failed",
		},
	}

	c, err := NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		_, err = c.LogReader(tc.config)
		if !tc.expectedError && err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if tc.expectedError {
			if err == nil {
				t.Errorf("Expected provided config to be invalid but got no error")
			} else if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
				t.Errorf("Expected error to contain %q, got %q", tc.expectedErrorMsg, err.Error())
			}
		}
	}
}

func TestPodsForLogs(t *testing.T) {
	pluginName := "my-plugin"
	pluginPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"sonobuoy-plugin": pluginName,
			},
		},
	}
	allPods := &corev1.PodList{Items: []corev1.Pod{pluginPod, {}}}
	testCases := []struct {
		desc                  string
		pluginName            string
		expectedCallCount     int
		expectedLabelSelector string
		podListError          error
		expectedPodCount      int
		expectedError         error
	}{
		{
			desc:                  "No plugin specified results in all pods being fetched once",
			pluginName:            "",
			expectedLabelSelector: "",
			podListError:          nil,
			expectedPodCount:      2,
			expectedError:         nil,
		},
		{
			desc:                  "Plugin specified results in plugin pods being fetched once",
			pluginName:            pluginName,
			expectedLabelSelector: "sonobuoy-plugin=my-plugin",
			podListError:          nil,
			expectedPodCount:      1,
			expectedError:         nil,
		},
		{
			desc:                  "Error when fetching plugin pods results in error being returned",
			pluginName:            pluginName,
			expectedLabelSelector: "sonobuoy-plugin=my-plugin",
			podListError:          errors.New("error"),
			expectedPodCount:      1,
			expectedError:         errors.New("failed to list pods: error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := &LogConfig{Plugin: tc.pluginName}

			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
				listAction := action.(k8stesting.ListAction)
				labelSelector := listAction.GetListRestrictions().Labels.String()
				if labelSelector != tc.expectedLabelSelector {
					t.Errorf("expected label selector to be %q, got %q", tc.expectedLabelSelector, labelSelector)
				}
				err := tc.podListError
				return true, allPods, err
			})

			podCh := make(chan *v1.Pod)
			err := getPodsToStreamLogs(fclient, cfg, podCh)
			if tc.expectedError != nil {
				if err == nil || err.Error() != tc.expectedError.Error() {
					t.Errorf("Unexpected error result, expected %q, got %q", tc.expectedError, err)
				}
			} else {
				podCount := 0
				for range podCh {
					podCount++
				}
				if podCount != tc.expectedPodCount {
					t.Errorf("Unexpected number of pods returned, expected %d, got %d", tc.expectedPodCount, podCount)
				}
			}
		})
	}
}

func getTestContainerStatus() *[]v1.ContainerStatus {
	return &[]v1.ContainerStatus{
		{
			Name: "foo",
			State: v1.ContainerState{
				Waiting: &v1.ContainerStateWaiting{
					Reason:  "foo container is waiting",
					Message: "foo container is waiting",
				},
			},
		},
		{
			Name: "bar",
			State: v1.ContainerState{
				Terminated: &v1.ContainerStateTerminated{
					ExitCode: 143,
					Reason:   "sigterm",
					Message:  "container bar was terminated",
				},
			},
		},
		{
			Name: "baz",
			State: v1.ContainerState{
				Running: &v1.ContainerStateRunning{
					StartedAt: metav1.Now(),
				},
			},
		},
	}
}

func TestIsContainerRunning(t *testing.T) {
	testCases := []struct {
		name                       string
		inputContainerStatuses     *[]v1.ContainerStatus
		inputContainerName         string
		expectedIsContainerRunning bool
	}{
		{
			name:                       "should report container as running",
			inputContainerStatuses:     getTestContainerStatus(),
			inputContainerName:         "baz",
			expectedIsContainerRunning: true,
		},
		{
			name:                       "should report waiting container as not running",
			inputContainerStatuses:     getTestContainerStatus(),
			inputContainerName:         "foo",
			expectedIsContainerRunning: false,
		},
		{
			name:                       "should report terminated container as not running",
			inputContainerStatuses:     getTestContainerStatus(),
			inputContainerName:         "foo",
			expectedIsContainerRunning: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualIsContianerRunning := isContainerRunning(tc.inputContainerStatuses, tc.inputContainerName)
			if tc.expectedIsContainerRunning != actualIsContianerRunning {
				t.Fatalf("isContainerRunning failed, Want %t; Got %t", tc.expectedIsContainerRunning, actualIsContianerRunning)
			}
		})
	}
}
