package loader

import (
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/pkg/errors"

	"github.com/heptio/sonobuoy/pkg/plugin/driver/daemonset"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/job"
)

func TestFindPlugins(t *testing.T) {
	testdir := path.Join("testdata", "plugin.d")
	plugins, err := findPlugins(testdir)
	if err != nil {
		t.Fatalf("unexpected err %v", err)
	}

	expected := []string{
		"testdata/plugin.d/daemonset.yml",
		"testdata/plugin.d/invalid.yml",
		"testdata/plugin.d/job.yml",
	}
	sort.Strings(plugins)

	if !reflect.DeepEqual(expected, plugins) {
		t.Errorf("expected %v, got %v", expected, plugins)
	}
}

func TestLoadNonexistentPlugin(t *testing.T) {
	_, err := loadPlugin("non/existent/path", "")
	if errors.Cause(err).Error() != "open non/existent/path: no such file or directory" {
		t.Errorf("Expected ErrNotExist, got %v", errors.Cause(err))
	}
}

func TestLoadValidPlugin(t *testing.T) {
	namespace := "test"
	jobSpec := "testdata/plugin.d/job.yml"
	plugin, err := loadPlugin(jobSpec, namespace)
	if err != nil {
		t.Fatalf("Unexpected error creating job plugin: %v", err)
	}

	jobPlugin, ok := plugin.(*job.Plugin)
	if !ok {
		t.Fatalf("loaded plugin not a job.Plugin")
	}

	if jobPlugin.Definition.Name != "test-job-plugin" {
		t.Errorf("expected plugin name 'test-job-plugin', got '%v'", jobPlugin.Definition.Name)
	}
	if jobPlugin.Namespace != namespace {
		t.Errorf("expected plugin name '%q', got '%v'", namespace, jobPlugin.Namespace)
	}

	daemonSetSpec := "testdata/plugin.d/daemonset.yml"
	plugin, err = loadPlugin(daemonSetSpec, namespace)
	if err != nil {
		t.Fatalf("Unexpected error creating job plugin: %v", err)
	}

	daemonSetPlugin, ok := plugin.(*daemonset.Plugin)
	if !ok {
		t.Fatalf("loaded plugin not daemonset.Plugin")
	}

	if daemonSetPlugin.Definition.Name != "test-daemon-set-plugin" {
		t.Errorf("expected plugin name 'test-daemon-set-plugin', got '%v'", daemonSetPlugin.Definition.Name)
	}
	if daemonSetPlugin.Namespace != namespace {
		t.Errorf("expected plugin name '%q', got '%v'", namespace, daemonSetPlugin.Namespace)
	}
}
