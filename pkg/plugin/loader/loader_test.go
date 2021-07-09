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

package loader

import (
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/job"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	corev1 "k8s.io/api/core/v1"
)

func TestFindPlugins(t *testing.T) {
	testdir := path.Join("testdata", "plugin.d")
	plugins, err := findPlugins(testdir)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	expected := []string{
		"testdata/plugin.d/daemonset.yaml",
		"testdata/plugin.d/invalid.yml",
		"testdata/plugin.d/job.yml",
	}
	sort.Strings(plugins)

	if !reflect.DeepEqual(expected, plugins) {
		t.Errorf("expected %v, got %v", expected, plugins)
	}
}

func TestLoadNonexistentPlugin(t *testing.T) {
	_, err := LoadDefinitionFromFile("non/existent/path")
	if errors.Cause(err).Error() != "open non/existent/path: no such file or directory" {
		t.Errorf("Expected ErrNotExist, got %v", errors.Cause(err))
	}
}

func TestLoadValidPlugin(t *testing.T) {
	jobDefFileName := "testdata/plugin.d/job.yml"
	jobDef, err := LoadDefinitionFromFile(jobDefFileName)
	if err != nil {
		t.Fatalf("Unexpected error reading job plugin: %v", err)
	}

	if jobDef.SonobuoyConfig.Driver != "Job" {
		t.Errorf("expected driver Job, got %q", jobDef.SonobuoyConfig.Driver)
	}
	if jobDef.SonobuoyConfig.PluginName != "test-job-plugin" {
		t.Errorf("expected name test-job-plugin, got %q", jobDef.SonobuoyConfig.PluginName)
	}

	if jobDef.Spec.Image != "gcr.io/heptio-images/heptio-e2e:master" {
		t.Errorf("expected name gcr.io/heptio-images/heptio-e2e:master, got %q", jobDef.Spec.Image)
	}

	daemonDefFileName := "testdata/plugin.d/daemonset.yaml"
	daemonDef, err := LoadDefinitionFromFile(daemonDefFileName)
	if err != nil {
		t.Fatalf("Unexpected error creating daemonset plugin: %v", err)
	}

	if daemonDef.SonobuoyConfig.Driver != "DaemonSet" {
		t.Errorf("expected driver DaemonSet, got %q", daemonDef.SonobuoyConfig.Driver)
	}
	if daemonDef.SonobuoyConfig.PluginName != "test-daemon-set-plugin" {
		t.Errorf("expected name test-daemon-set-plugin, got %q", daemonDef.SonobuoyConfig.PluginName)
	}
	if daemonDef.Spec.Image != "gcr.io/heptio-images/heptio-e2e:master" {
		t.Errorf("expected name gcr.io/heptio-images/heptio-e2e:master, got %q", jobDef.Spec.Image)
	}
}

func TestLoadValidPluginWithSkipCleanup(t *testing.T) {
	testCases := []struct {
		desc           string
		jobDefFileName string
		expectedValue  bool
	}{
		{
			desc:           "skip-cleanup set to true results in true",
			jobDefFileName: "testdata/skip-cleanup/set-true.yml",
			expectedValue:  true,
		},
		{
			desc:           "skip-cleanup set to true results in true",
			jobDefFileName: "testdata/skip-cleanup/set-false.yml",
			expectedValue:  false,
		},
		{
			desc:           "skip-cleanup not set defaults to false",
			jobDefFileName: "testdata/skip-cleanup/not-set.yml",
			expectedValue:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			jobDef, err := LoadDefinitionFromFile(tc.jobDefFileName)
			if err != nil {
				t.Fatalf("Unexpected error reading job plugin: %v", err)
			}

			if jobDef.SonobuoyConfig.SkipCleanup != tc.expectedValue {
				t.Errorf("expected skip-cleanup to be %v but was %v", tc.expectedValue, jobDef.SonobuoyConfig.SkipCleanup)
			}

		})

	}
}

func TestLoadJobPlugin(t *testing.T) {
	namespace := "loader_test"
	image := "gcr.io/heptio-images/sonobuoy:latest"
	jobDef := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			Driver:     "Job",
			PluginName: "test-job-plugin",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Image: "gcr.io/heptio-images/heptio-e2e:master",
			},
		},
	}

	pluginIface, err := loadPlugin(jobDef, namespace, image, "Always", "image-pull-secrets", nil)
	if err != nil {
		t.Fatalf("unexpected error loading plugin: %v", err)
	}

	jobPlugin, ok := pluginIface.(*job.Plugin)

	if !ok {
		t.Fatalf("loaded plugin not a job.Plugin")
	}

	if jobPlugin.GetName() != "test-job-plugin" {
		t.Errorf("expected plugin name 'test-job-plugin', got '%v'", jobPlugin.GetName())
	}
	if jobPlugin.Definition.Spec.Image != "gcr.io/heptio-images/heptio-e2e:master" {
		t.Errorf("expected plugin name 'gcr.io/heptio-images/heptio-e2e:master', got '%v'", jobPlugin.Definition.Spec.Image)
	}
	if jobPlugin.Namespace != namespace {
		t.Errorf("expected plugin name '%q', got '%v'", namespace, jobPlugin.Namespace)
	}
	if jobPlugin.ImagePullSecrets != "image-pull-secrets" {
		t.Errorf("Expected imagePullSecrets with name %v but got %v", "image-pull-secret", jobPlugin.ImagePullSecrets)
	}
}

func TestLoadDaemonSet(t *testing.T) {
	namespace := "loader_test"
	image := "gcr.io/heptio-images/sonobuoy:latest"
	daemonDef := manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			Driver:     "DaemonSet",
			PluginName: "test-daemon-set-plugin",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Image: "gcr.io/heptio-images/heptio-e2e:master",
			},
		},
	}

	pluginIface, err := loadPlugin(daemonDef, namespace, image, "Always", "image-pull-secrets", nil)
	if err != nil {
		t.Fatalf("unexpected error loading plugin: %v", err)
	}

	daemonPlugin, ok := pluginIface.(*daemonset.Plugin)

	if !ok {
		t.Fatalf("loaded plugin not a daemon.Plugin")
	}

	if daemonPlugin.GetName() != "test-daemon-set-plugin" {
		t.Errorf("expected plugin name 'test-daemon-set-plugin', got '%v'", daemonPlugin.GetName())
	}
	if daemonPlugin.Definition.Spec.Image != "gcr.io/heptio-images/heptio-e2e:master" {
		t.Errorf("expected plugin name 'gcr.io/heptio-images/heptio-e2e:master', got '%v'", daemonPlugin.Definition.Spec.Image)
	}
	if daemonPlugin.Namespace != namespace {
		t.Errorf("expected plugin name '%q', got '%v'", namespace, daemonPlugin.Namespace)
	}
	if daemonPlugin.ImagePullSecrets != "image-pull-secrets" {
		t.Errorf("Expected imagePullSecrets with name %v but got %v", "image-pull-secret", daemonPlugin.ImagePullSecrets)
	}
}

func TestFilterList(t *testing.T) {
	definitions := []manifest.Manifest{
		{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "test1"}},
		{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "test2"}},
	}

	selections := []plugin.Selection{
		{Name: "test1"},
		{Name: "test3"},
	}

	expected := []manifest.Manifest{definitions[0]}
	filtered := filterPluginDef(definitions, selections)
	if !reflect.DeepEqual(filtered, expected) {
		t.Errorf("expected %+#v, got %+#v", expected, filtered)
	}
}

func TestLoadAllPlugins(t *testing.T) {
	testcases := []struct {
		testname            string
		namespace           string
		sonobuoyImage       string
		imagePullPolicy     string
		imagePullSecrets    string
		customAnnotations   map[string]string
		searchPath          []string
		selections          []plugin.Selection
		expectedPluginNames []string
	}{
		{
			testname:   "ensure duplicate paths do not result in duplicate loaded plugins.",
			searchPath: []string{path.Join("testdata", "plugin.d"), path.Join("testdata", "plugin.d")},
			selections: []plugin.Selection{
				{Name: "test-job-plugin"},
				{Name: "test-daemon-set-plugin"},
			},
			expectedPluginNames: []string{"test-job-plugin", "test-daemon-set-plugin"},
		}, {
			testname:            "nil selections defaults to run all",
			searchPath:          []string{path.Join("testdata", "onlyvalid")},
			selections:          nil,
			expectedPluginNames: []string{"test-job-plugin", "test-daemon-set-plugin"},
		}, {
			testname:            "empty (non-nil) selection runs none",
			searchPath:          []string{path.Join("testdata", "plugin.d")},
			selections:          []plugin.Selection{},
			expectedPluginNames: []string{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.testname, func(t *testing.T) {
			plugins, err := LoadAllPlugins(tc.namespace, tc.sonobuoyImage, tc.imagePullPolicy, tc.imagePullSecrets, tc.customAnnotations, tc.searchPath, tc.selections)
			if err != nil {
				t.Fatalf("error loading all plugins: %v", err)
			}
			if len(plugins) != len(tc.expectedPluginNames) {
				t.Fatalf("expected %v plugins but got %v", len(tc.expectedPluginNames), len(plugins))
			}
			for i, plugin := range plugins {
				found := false
				for _, expectedPlugin := range tc.expectedPluginNames {
					if plugin.GetName() == expectedPlugin {
						found = true
					}
				}
				if !found {
					t.Fatalf("Expected %v but got %v", tc.expectedPluginNames[i], plugin.GetName())
				}
			}
		})
	}
}
