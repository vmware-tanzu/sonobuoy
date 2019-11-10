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

package daemonset

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/backplane/ca"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	"github.com/kylelemons/godebug/pretty"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	expectedImageName = "gcr.io/org/sonobuoy:master"
	expectedNamespace = "test-namespace"
)

var aggregatorPod corev1.Pod

func createClientCertificate(name string) (*tls.Certificate, error) {
	auth, err := ca.NewAuthority()
	if err != nil {
		return nil, fmt.Errorf("couldn't make CA Authority %v", err)
	}

	clientCert, err := auth.ClientKeyPair("test-job")
	if err != nil {
		return nil, fmt.Errorf("couldn't make client certificate %v", err)
	}
	return clientCert, nil
}

func TestCreateDaemonSetDefintion(t *testing.T) {
	pluginName := "test-plugin"
	testDaemonSet := NewPlugin(
		manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{
				Driver:     "DaemonSet",
				PluginName: pluginName,
			},
			Spec: manifest.Container{
				Container: corev1.Container{
					Name: "producer-container",
				},
			},
			ExtraVolumes: []manifest.Volume{
				{
					Volume: corev1.Volume{
						Name: "test1",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/test",
							},
						},
					},
				},
				{
					Volume: corev1.Volume{
						Name: "test2",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/test2",
							},
						},
					},
				},
			},
		}, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{"key1": "val1", "key2": "val2"})

	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("couldn't make CA Authority %v", err)
	}
	clientCert, err := auth.ClientKeyPair(pluginName)
	if err != nil {
		t.Fatalf("couldn't make client certificate %v", err)
	}

	daemonSet := testDaemonSet.createDaemonSetDefinition("", clientCert, &corev1.Pod{}, "")

	expectedName := fmt.Sprintf("sonobuoy-%v-daemon-set-%v", pluginName, testDaemonSet.SessionID)
	if daemonSet.Name != expectedName {
		t.Errorf("Expected daemonSet name %v, got %v", expectedName, daemonSet.Name)
	}

	if daemonSet.Namespace != expectedNamespace {
		t.Errorf("Expected daemonSet namespace %v, got %v", expectedNamespace, daemonSet.Namespace)
	}

	pluginLabel := "sonobuoy-plugin"
	if daemonSet.Labels[pluginLabel] != pluginName {
		t.Errorf("Expected daemonSet to have label %q with value %q, but had value %q", pluginLabel, pluginName, daemonSet.Labels[pluginLabel])
	}
	containers := daemonSet.Spec.Template.Spec.Containers

	expectedContainers := 2
	if len(containers) != expectedContainers {
		t.Errorf("Expected to have %v containers, got %v", expectedContainers, len(containers))
	} else {
		// Don't segfault if the count is incorrect
		expectedProducerName := "producer-container"
		if containers[0].Name != expectedProducerName {
			t.Errorf(
				"Expected producer pod to have name %v, got %v",
				expectedProducerName,
				containers[0].Name,
			)
		}
		if containers[1].Image != expectedImageName {
			t.Errorf(
				"Expected consumer pod to have image %v, got %v",
				expectedImageName,
				containers[1].Image,
			)
		}
	}

	env := make(map[string]string)
	for _, envVar := range daemonSet.Spec.Template.Spec.Containers[1].Env {
		env[envVar.Name] = envVar.Value
	}

	caCertPEM, ok := env["CA_CERT"]
	if !ok {
		t.Fatal("no env var CA_CERT")
	}
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		t.Fatal("No PEM block found.")
	}

	caCertFingerprint := sha1.Sum(caCertBlock.Bytes)

	if caCertFingerprint != sha1.Sum(auth.CACert().Raw) {
		t.Errorf("CA_CERT fingerprint didn't match")
	}

	if len(daemonSet.Spec.Template.Spec.Volumes) != 4 {
		t.Errorf("Expected 4 volumes defined, got %d", len(daemonSet.Spec.Template.Spec.Volumes))
	}

	pullSecrets := daemonSet.Spec.Template.Spec.ImagePullSecrets
	if len(pullSecrets) != 1 {
		t.Errorf("Expected 1 imagePullSecrets but got %v", len(pullSecrets))
	} else {
		if pullSecrets[0].Name != "image-pull-secret" {
			t.Errorf("Expected imagePullSecrets with name %v but got %v", "image-pull-secret", pullSecrets)
		}
	}

	if daemonSet.Annotations["key1"] != "val1" ||
		daemonSet.Annotations["key2"] != "val2" {
		t.Errorf("Expected annotations key1:val1 and key2:val2 to be set, but got %v", daemonSet.Annotations)
	}
	if daemonSet.Spec.Template.Annotations["key1"] != "val1" ||
		daemonSet.Spec.Template.Annotations["key2"] != "val2" {
		t.Errorf("Expected annotations key1:val1 and key2:val2 to be set, but got %v", daemonSet.Spec.Template.Annotations)
	}
}

func TestCreateDaemonSetDefintionUsesDefaultPodSpec(t *testing.T) {
	testDaemonSet := NewPlugin(
		manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{
				Driver:     "DaemonSet",
				PluginName: "test-plugin",
			},
			Spec: manifest.Container{
				Container: corev1.Container{
					Name: "producer-container",
				},
			},
		}, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{"key1": "val1", "key2": "val2"})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	daemonSet := testDaemonSet.createDaemonSetDefinition("", clientCert, &corev1.Pod{}, "")
	podSpec := daemonSet.Spec.Template.Spec

	expectedServiceAccount := "sonobuoy-serviceaccount"

	if podSpec.ServiceAccountName != expectedServiceAccount {
		t.Errorf("expected pod spec to have default service account name %q, got %q", expectedServiceAccount, podSpec.ServiceAccountName)

	}

	// Check something specific to the daemonset default pod spec
	expectedNumTolerations := 1
	actualNumTolerations := len(podSpec.Tolerations)
	if actualNumTolerations != expectedNumTolerations {
		t.Errorf("expected pod spec to %v tolerations, got %v", expectedNumTolerations, actualNumTolerations)
	}
}

func TestCreateDaemonSetDefintionUsesProvidedPodSpec(t *testing.T) {
	expectedServiceAccountName := "test-serviceaccount"
	testDaemonSet := NewPlugin(
		manifest.Manifest{
			SonobuoyConfig: manifest.SonobuoyConfig{
				Driver:     "DaemonSet",
				PluginName: "test-plugin",
			},
			Spec: manifest.Container{
				Container: corev1.Container{
					Name: "producer-container",
				},
			},
			PodSpec: &manifest.PodSpec{
				PodSpec: corev1.PodSpec{ServiceAccountName: expectedServiceAccountName},
			},
		}, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	daemonSet := testDaemonSet.createDaemonSetDefinition("", clientCert, &corev1.Pod{}, "")
	podSpec := daemonSet.Spec.Template.Spec

	if podSpec.ServiceAccountName != expectedServiceAccountName {
		t.Errorf("expected pod spec to have provided service account name %q, got %q", expectedServiceAccountName, podSpec.ServiceAccountName)
	}
}

func TestCreateDaemonSetDefinitionAddsToExistingResourcesInPodSpec(t *testing.T) {
	m := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			Driver:     "DaemonSet",
			PluginName: "test-job",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name: "producer-container",
			},
		},
		ExtraVolumes: []manifest.Volume{{Volume: corev1.Volume{Name: "test1"}}},
		PodSpec: &manifest.PodSpec{
			PodSpec: corev1.PodSpec{
				Containers:       []corev1.Container{{}},
				ImagePullSecrets: []corev1.LocalObjectReference{{}},
				Volumes:          []corev1.Volume{{}},
			},
		},
	}
	testPlugin := NewPlugin(m, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	daemonSet := testPlugin.createDaemonSetDefinition("", clientCert, &corev1.Pod{}, "")
	podSpec := daemonSet.Spec.Template.Spec

	// Existing container in pod spec, plus 2 added by Sonobuoy
	expectedNumContainers := 3
	actualNumContainers := len(podSpec.Containers)
	if expectedNumContainers != actualNumContainers {
		t.Errorf("expected pod spec to have %v containers, got %v", expectedNumContainers, actualNumContainers)
	}

	// Existing image pull secret in pod spec, plus 1 added by Sonobuoy
	expectedNumImagePullSecrets := 2
	actualNumImagePullSecrets := len(podSpec.ImagePullSecrets)
	if expectedNumImagePullSecrets != actualNumImagePullSecrets {
		t.Errorf("expected pod spec to have %v image pull secrets, got %v", expectedNumImagePullSecrets, actualNumImagePullSecrets)
	}

	// Existing volume in pod spec, plus 1 extra volume, plus 1 added by Sonobuoy
	expectedNumVolumes := 3
	actualNumVolumes := len(podSpec.Volumes)
	if expectedNumVolumes != actualNumVolumes {
		t.Errorf("expected pod spec to have %v volumes, got %v", expectedNumVolumes, actualNumVolumes)
	}
}

func TestCreateDaemonSetDefinitionSetsOwnerReference(t *testing.T) {
	m := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			Driver:     "DaemonSet",
			PluginName: "test-job",
		},
		Spec: manifest.Container{Container: corev1.Container{}},
	}
	testPlugin := NewPlugin(m, expectedNamespace, expectedImageName, "Always", "image-pull-secret", map[string]string{})

	clientCert, err := createClientCertificate("test-job")
	if err != nil {
		t.Fatalf("couldn't create client certificate: %v", err)
	}

	aggregatorPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "sonobuoy-aggregator",
			UID:  "123456-abcdef",
		},
	}

	daemonSet := testPlugin.createDaemonSetDefinition("", clientCert, &aggregatorPod, "")
	ownerReferences := daemonSet.ObjectMeta.OwnerReferences

	if len(ownerReferences) != 1 {
		t.Fatalf("Expected 1 owner reference, got %v", len(ownerReferences))
	}

	testCases := []struct {
		field string
		want  string
		got   string
	}{
		{
			field: "APIVersion",
			want:  "v1",
			got:   ownerReferences[0].APIVersion,
		},
		{
			field: "Kind",
			want:  "Pod",
			got:   ownerReferences[0].Kind,
		},
		{
			field: "Name",
			want:  aggregatorPod.ObjectMeta.Name,
			got:   ownerReferences[0].Name,
		},
		{
			field: "UID",
			want:  string(aggregatorPod.ObjectMeta.UID),
			got:   string(ownerReferences[0].UID),
		},
	}

	for _, tc := range testCases {
		if tc.got != tc.want {
			t.Errorf("Expected ownerReference %v to be %q, got %q", tc.field, tc.want, tc.got)
		}
	}
}

func TestMonitorOnce(t *testing.T) {
	// Note: the pods/ds must be marked with the label "sonobuoy-run" or else our labelSelector
	// logic will filter them out even though the fake server returns them.

	// We will need to be able to grab these items repeatedly and tweak the minor details
	// so these helpers make the test cases much more readable.
	testPlugin := &Plugin{Base: driver.Base{Definition: manifest.Manifest{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myPlugin"}}}}
	testPluginWithAffinity := &Plugin{Base: driver.Base{Definition: manifest.Manifest{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myPlugin"}}}}
	testPluginWithAffinity.Base.Definition.PodSpec = &manifest.PodSpec{
		PodSpec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{{Key: "foo", Operator: corev1.NodeSelectorOpExists}},
							},
						},
					},
				},
			},
		},
	}

	validDS := appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
	}
	validPod := func(node string) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
			Spec:       corev1.PodSpec{NodeName: node},
		}
	}
	failingPod := func(node string) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"sonobuoy-run": ""}},
			Spec:       corev1.PodSpec{NodeName: node},
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{
					{Reason: "Unschedulable", Message: "conditionMsg"},
				},
			},
		}
	}
	default3Nodes := []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "node3", Labels: map[string]string{"foo": "bar"}}},
	}

	testCases := []struct {
		desc       string
		expectDone bool

		// Ensure we are getting the err result we expect; lots of ways to get errors
		// that may not be clear.
		expectErrResultMsgs []string

		dsPlugin                *Plugin
		dsOnServer              *appsv1.DaemonSetList
		podsOnServer            *corev1.PodList
		nodes                   []corev1.Node
		errFromServerForDSList  error
		errFromServerForPodList error
	}{
		{
			desc:       "Cleaned up indicates exit without error",
			expectDone: true,
			dsPlugin:   &Plugin{driver.Base{CleanedUp: true}},
		}, {
			desc:       "Missing daemonset results in no errors",
			dsPlugin:   testPlugin,
			dsOnServer: &appsv1.DaemonSetList{},
		}, {
			desc:                    "Failed pod lookup results in no error",
			dsPlugin:                testPlugin,
			dsOnServer:              &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer:            &corev1.PodList{},
			errFromServerForPodList: errors.New("pod lookup err"),
		}, {
			desc:         "Missing pods results in errors for each",
			nodes:        default3Nodes,
			dsPlugin:     testPlugin,
			dsOnServer:   &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer: &corev1.PodList{},
			expectErrResultMsgs: []string{
				"No pod was scheduled on node node1 within 2562047h47m16.854775807s. Check tolerations for plugin myPlugin",
				"No pod was scheduled on node node2 within 2562047h47m16.854775807s. Check tolerations for plugin myPlugin",
				"No pod was scheduled on node node3 within 2562047h47m16.854775807s. Check tolerations for plugin myPlugin",
			},
		}, {
			desc:         "Missing pods results in errors for each only if targeting those nodes",
			nodes:        default3Nodes,
			dsPlugin:     testPluginWithAffinity,
			dsOnServer:   &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer: &corev1.PodList{},
			expectErrResultMsgs: []string{
				"No pod was scheduled on node node3 within 2562047h47m16.854775807s. Check tolerations for plugin myPlugin",
			},
		}, {
			desc:       "Failing pod results in error",
			nodes:      default3Nodes,
			dsPlugin:   testPlugin,
			dsOnServer: &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer: &corev1.PodList{
				Items: []corev1.Pod{failingPod("node1"), validPod("node2"), validPod("node3")},
			},
			expectErrResultMsgs: []string{"Can't schedule pod: conditionMsg"},
		}, {
			desc:       "Two failing pod results in 2 errors",
			nodes:      default3Nodes,
			dsPlugin:   testPlugin,
			dsOnServer: &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer: &corev1.PodList{
				Items: []corev1.Pod{validPod("node2"), failingPod("node1"), failingPod("node3")},
			},
			expectErrResultMsgs: []string{"Can't schedule pod: conditionMsg", "Can't schedule pod: conditionMsg"},
		}, {
			desc:       "Healthy pods results in no error and continued monitoring",
			nodes:      default3Nodes,
			dsPlugin:   testPlugin,
			dsOnServer: &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer: &corev1.PodList{
				Items: []corev1.Pod{validPod("node1"), validPod("node2"), validPod("node3")},
			},
		}, {
			desc:       "Failing and missing pod errors both get reported",
			nodes:      default3Nodes,
			dsPlugin:   testPlugin,
			dsOnServer: &appsv1.DaemonSetList{Items: []appsv1.DaemonSet{validDS}},
			podsOnServer: &corev1.PodList{
				Items: []corev1.Pod{failingPod("node2")},
			},
			expectErrResultMsgs: []string{
				"Can't schedule pod: conditionMsg",
				"No pod was scheduled on node node1 within 2562047h47m16.854775807s. Check tolerations for plugin myPlugin",
				"No pod was scheduled on node node3 within 2562047h47m16.854775807s. Check tolerations for plugin myPlugin",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fclient := fake.NewSimpleClientset()
			fclient.PrependReactor("list", "daemonsets", func(action k8stesting.Action) (handled bool, ret kuberuntime.Object, err error) {
				return true, tc.dsOnServer, tc.errFromServerForDSList
			})
			fclient.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret kuberuntime.Object, err error) {
				return true, tc.podsOnServer, tc.errFromServerForPodList
			})
			foundmap, reportedmap := map[string]bool{}, map[string]bool{}

			done, errResults := tc.dsPlugin.monitorOnce(fclient, tc.nodes, foundmap, reportedmap)
			if done != tc.expectDone {
				t.Errorf("Expected %v but got %v", tc.expectDone, done)
			}

			if len(errResults) != len(tc.expectErrResultMsgs) {
				t.Errorf("Expected %v errors but got %v:", len(tc.expectErrResultMsgs), len(errResults))
				for _, v := range errResults {
					t.Errorf("  - %q\n", v.Error)
				}
				t.FailNow()
			}
			for i := range errResults {
				if errResults[i].Error != tc.expectErrResultMsgs[i] {
					t.Errorf("Expected error[%v] to be %q but got %q", i, tc.expectErrResultMsgs[i], errResults[i].Error)
				}
			}
		})
	}
}

func TestExpectedResults(t *testing.T) {
	testNodes := []corev1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "node2", Labels: map[string]string{"foo": "bar"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "node3", Labels: map[string]string{"foo": "baz"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "node4", Labels: map[string]string{"foo": "bar2"}}},
	}

	pluginWithAffinity := func(reqs []corev1.NodeSelectorRequirement) *Plugin {
		p := &Plugin{
			Base: driver.Base{
				Definition: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myPlugin"},
				},
			},
		}
		if len(reqs) > 0 {
			p.Base.Definition.PodSpec = &manifest.PodSpec{
				PodSpec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: reqs,
									},
								},
							},
						},
					},
				},
			}
		}
		return p
	}

	testCases := []struct {
		desc   string
		p      *Plugin
		expect []plugin.ExpectedResult
	}{
		{
			desc: "Defaults to all nodes",
			expect: []plugin.ExpectedResult{
				{NodeName: "node1", ResultType: "myPlugin"},
				{NodeName: "node2", ResultType: "myPlugin"},
				{NodeName: "node3", ResultType: "myPlugin"},
				{NodeName: "node4", ResultType: "myPlugin"},
			},
			p: pluginWithAffinity(nil),
		}, {
			desc: "Filters for label exists",
			expect: []plugin.ExpectedResult{
				{NodeName: "node2", ResultType: "myPlugin"},
				{NodeName: "node3", ResultType: "myPlugin"},
				{NodeName: "node4", ResultType: "myPlugin"},
			},
			p: pluginWithAffinity([]corev1.NodeSelectorRequirement{
				{Key: "foo", Operator: corev1.NodeSelectorOpExists},
			}),
		}, {
			desc: "Filters for label does not exist",
			expect: []plugin.ExpectedResult{
				{NodeName: "node1", ResultType: "myPlugin"},
			},
			p: pluginWithAffinity([]corev1.NodeSelectorRequirement{
				{Key: "foo", Operator: corev1.NodeSelectorOpDoesNotExist},
			}),
		}, {
			desc: "Filters for label value in",
			expect: []plugin.ExpectedResult{
				{NodeName: "node2", ResultType: "myPlugin"},
				{NodeName: "node3", ResultType: "myPlugin"},
			},
			p: pluginWithAffinity([]corev1.NodeSelectorRequirement{
				{Key: "foo", Operator: corev1.NodeSelectorOpIn, Values: []string{"bar", "baz"}},
			}),
		}, {
			desc: "Filters for label value not in",
			expect: []plugin.ExpectedResult{
				{NodeName: "node1", ResultType: "myPlugin"},
				{NodeName: "node4", ResultType: "myPlugin"},
			},
			p: pluginWithAffinity([]corev1.NodeSelectorRequirement{
				{Key: "foo", Operator: corev1.NodeSelectorOpNotIn, Values: []string{"bar", "baz"}},
			}),
		}, {
			desc: "Can combine filters as union",
			expect: []plugin.ExpectedResult{
				{NodeName: "node1", ResultType: "myPlugin"},
				{NodeName: "node2", ResultType: "myPlugin"},
				{NodeName: "node4", ResultType: "myPlugin"},
			},
			p: pluginWithAffinity([]corev1.NodeSelectorRequirement{
				{Key: "foo", Operator: corev1.NodeSelectorOpNotIn, Values: []string{"bar", "baz"}},
				{Key: "foo", Operator: corev1.NodeSelectorOpIn, Values: []string{"bar"}},
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out := tc.p.ExpectedResults(testNodes)
			if diff := pretty.Compare(tc.expect, out); diff != "" {
				t.Fatalf("\n\n%s\n", diff)
			}
		})
	}
}
