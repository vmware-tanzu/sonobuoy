package client_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/buildinfo"
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var update = flag.Bool("update", false, "update .golden files")

func TestGenerateManifest(t *testing.T) {
	tcs := []struct {
		name     string
		inputcm  *client.GenConfig
		expected *config.Config
	}{
		{
			name: "nil config",
			inputcm: &client.GenConfig{
				Config: nil,
			},
			expected: &config.Config{},
		},
		{
			name: "Defaults in yield a default manifest.",
			inputcm: &client.GenConfig{
				Config: &config.Config{},
			},
			expected: &config.Config{},
		},
		{
			name: "Overriding the bind address",
			inputcm: &client.GenConfig{
				Config: &config.Config{
					Aggregation: plugin.AggregationConfig{
						BindAddress: "10.0.0.1",
					},
				},
			},
			expected: &config.Config{
				Aggregation: plugin.AggregationConfig{
					BindAddress: "10.0.0.1",
				},
			},
		},
		{
			name: "Overriding the plugin selection",
			inputcm: &client.GenConfig{
				Config: &config.Config{
					PluginSelections: []plugin.Selection{
						{
							Name: "systemd-logs",
						},
					},
				},
			},
			expected: &config.Config{
				PluginSelections: []plugin.Selection{
					{
						Name: "systemd-logs",
					},
				},
				Aggregation: plugin.AggregationConfig{},
			},
		},
		{
			name: "The plugin search path is not modified",
			inputcm: &client.GenConfig{
				Config: &config.Config{
					PluginSearchPath: []string{"a", "b", "c", "a"},
				},
			},
			expected: &config.Config{
				Aggregation:      plugin.AggregationConfig{},
				PluginSearchPath: []string{"a", "b", "c", "a"},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sbc, err := client.NewSonobuoyClient(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := sbc.GenerateManifest(tc.inputcm)
			if err != nil {
				t.Fatal(err)
			}

			// TODO(chuckha) this is not my favorite thing.
			items := bytes.Split(manifest, []byte("---"))

			decoder := scheme.Codecs.UniversalDeserializer()
			for _, item := range items {
				o, gvk, err := decoder.Decode(item, nil, nil)
				if err != nil || gvk.Kind != "ConfigMap" {
					continue
				}

				cm, ok := o.(*v1.ConfigMap)
				if !ok {
					t.Fatal("was not a config map...")
				}

				// TODO(chuckha) test other pieces of the generated yaml
				if cm.ObjectMeta.Name != "sonobuoy-config-cm" {
					continue
				}

				configuration := &config.Config{}
				err = json.Unmarshal([]byte(cm.Data["config.json"]), configuration)
				if err != nil {
					t.Errorf("got error %v", err)
				}
				if !reflect.DeepEqual(configuration, tc.expected) {
					t.Fatalf("Expected %v to equal %v", tc.expected, configuration)
				}
			}
		})
	}
}

func TestGenerateManifestGolden(t *testing.T) {
	staticConfig := func() *config.Config {
		c := config.New()

		// Make version static so it doesn't have to be updated when we bump versions.
		// Use `replace` so we are still effectively testing that the version would have
		// matched the build version.
		c.Version = strings.Replace(c.Version, buildinfo.Version, "static-version-for-testing", -1)
		c.WorkerImage = strings.Replace(c.WorkerImage, buildinfo.Version, "static-version-for-testing", -1)
		return c
	}

	fromConfig := func(f func(*config.Config) *config.Config) *config.Config {
		c := staticConfig()
		return f(c)
	}

	tcs := []struct {
		name       string
		inputcm    *client.GenConfig
		goldenFile string
		expectErr  string
	}{
		{
			name: "Default",
			inputcm: &client.GenConfig{
				Config:      staticConfig(),
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "default.golden"),
		}, {
			name: "Only e2e (legacy plugin choice)",
			inputcm: &client.GenConfig{
				Config: fromConfig(func(c *config.Config) *config.Config {
					c.PluginSelections = []plugin.Selection{{Name: "e2e"}}
					return c
				}),
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "e2e-default.golden"),
		}, {
			name: "Only systemd_logs (legacy plugin choice)",
			inputcm: &client.GenConfig{
				Config: fromConfig(func(c *config.Config) *config.Config {
					c.PluginSelections = []plugin.Selection{{Name: "systemd-logs"}}
					return c
				}),
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "systemd-logs-default.golden"),
		}, {
			name: "Empty array leads to default plugins, not 0",
			inputcm: &client.GenConfig{
				Config: fromConfig(func(c *config.Config) *config.Config {
					c.PluginSelections = []plugin.Selection{}
					return c
				}),
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "default-plugins-via-selection.golden"),
		}, {
			// For backwards compatibility.
			name: "Nil plugin selection and no manual choice leads to e2e/systemd",
			inputcm: &client.GenConfig{
				Config: fromConfig(func(c *config.Config) *config.Config {
					c.PluginSelections = nil
					return c
				}),
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "default-plugins-via-nil-selection.golden"),
		}, {
			name: "Manually specify e2e",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				KubeVersion:    "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "manual-e2e.golden"),
		}, {
			name: "Manually specify custom plugin",
			inputcm: &client.GenConfig{
				StaticPlugins: []*manifest.Manifest{
					{
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "foo"},
					},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "manual-custom-plugin.golden"),
		}, {
			name: "Manually custom plugin and e2e plugins",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				StaticPlugins: []*manifest.Manifest{
					{
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "foo"},
					},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "manual-custom-plugin-plus-e2e.golden"),
		}, {
			name: "Manually custom plugin and systemd-logs plugins",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"systemd-logs"},
				StaticPlugins: []*manifest.Manifest{
					{
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "foo"},
					},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "manual-custom-plugin-plus-systemd.golden"),
		}, {
			name: "Duplicates plugin names fail",
			inputcm: &client.GenConfig{
				StaticPlugins: []*manifest.Manifest{
					{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "a"}},
					{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "a"}},
				},
				KubeVersion: "v99+static.testing",
			},
			expectErr: "plugin YAML generation: plugin names must be unique, got duplicated plugin name 'a'",
		}, {
			// In this case the server will just load both and filter like it does currently.
			name: "Plugin selection and custom plugins both specified allowed",
			inputcm: &client.GenConfig{
				Config: fromConfig(func(c *config.Config) *config.Config {
					c.PluginSelections = []plugin.Selection{
						{
							Name: "a",
						},
					}
					return c
				}),
				StaticPlugins: []*manifest.Manifest{
					{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "a"}},
					{SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "b"}},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "plugins-and-pluginSelection.golden"),
		}, {
			name: "ImagePullSecrets is set on plugins and aggregator",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				Config: &config.Config{
					ImagePullSecrets: "foo",
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "imagePullSecrets.golden"),
		}, {
			name: "Env overrides",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				PluginEnvOverrides: map[string]map[string]string{
					"e2e": {"E2E_SKIP": "override", "E2E_DRYRUN": "true"},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "envoverrides.golden"),
		}, {
			name: "Env overrides must match plugin names",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				PluginEnvOverrides: map[string]map[string]string{
					"e2e2": {"E2E_SKIP": "override", "E2E_DRYRUN": "true"},
				},
				KubeVersion: "v99+static.testing",
			},
			expectErr: "failed to override env vars for plugin e2e2, no plugin with that name found; have plugins: [e2e]",
		}, {
			name: "Default pod spec is included if requested and no other pod spec provided",
			inputcm: &client.GenConfig{
				ShowDefaultPodSpec: true,
				KubeVersion:        "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "default-pod-spec.golden"),
		}, {
			name: "E2E_USE_GO_RUNNER can be overridden/removed",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				PluginEnvOverrides: map[string]map[string]string{
					"e2e": {"E2E_USE_GO_RUNNER": ""},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "goRunnerRemoved.golden"),
		}, {
			name: "Existing pod spec is not modified if default pod spec is requested",
			inputcm: &client.GenConfig{
				ShowDefaultPodSpec: true,
				StaticPlugins: []*manifest.Manifest{
					{
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "a"},
						PodSpec:        &manifest.PodSpec{},
					},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "use-existing-pod-spec.golden"),
		}, {
			name: "Conformance images >= v1.17 support progress",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				KubeVersion:    "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "e2e-progress.golden"),
		}, {
			name: "ProgressUpdatesPort is customizable for e2e",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				Config: &config.Config{
					ProgressUpdatesPort: "1234",
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "e2e-progress-custom-port.golden"),
		}, {
			name: "Conformance images >= v1.17 will not override E2E_EXTRA_ARGS if specified by user",
			inputcm: &client.GenConfig{
				DynamicPlugins: []string{"e2e"},
				PluginEnvOverrides: map[string]map[string]string{
					"e2e": {"E2E_EXTRA_ARGS": "user-defined"},
				},
				KubeVersion: "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "e2e-progress-vs-user-defined.golden"),
		}, {
			name: "Node selector can be added",
			inputcm: &client.GenConfig{
				Config:        staticConfig(),
				NodeSelectors: map[string]string{"foo": "bar"},
				KubeVersion:   "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "single-node-selector.golden"),
		}, {
			name: "Multiple node selectors can be added",
			inputcm: &client.GenConfig{
				Config:        staticConfig(),
				NodeSelectors: map[string]string{"foo": "bar", "fizz": "buzz"},
				KubeVersion:   "v99+static.testing",
			},
			goldenFile: filepath.Join("testdata", "multiple-node-selector.golden"),
		}, {
			name: "Plugins can specify configmaps",
			inputcm: &client.GenConfig{
				Config:      staticConfig(),
				KubeVersion: "v99+static.testing",
				StaticPlugins: []*manifest.Manifest{
					{
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myplugin1"},
						ConfigMap: map[string]string{
							"file1": "contents1",
							"file2": "contents2",
						},
					}, {
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myplugin2"},
						ConfigMap: map[string]string{
							"file3": "contents3",
							"file4": "contents4",
						},
					},
				},
			},
			goldenFile: filepath.Join("testdata", "plugin-configmaps.golden"),
		}, {
			name: "ImagePullPolicy applied to all plugins",
			inputcm: &client.GenConfig{
				Config:      staticConfig(),
				KubeVersion: "v99+static.testing",
				StaticPlugins: []*manifest.Manifest{
					{
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myplugin1"},
						Spec:           manifest.Container{Container: v1.Container{ImagePullPolicy: "Never"}},
					}, {
						SonobuoyConfig: manifest.SonobuoyConfig{PluginName: "myplugin2"},
						Spec:           manifest.Container{Container: v1.Container{ImagePullPolicy: "Always"}},
					},
				},
			},
			goldenFile: filepath.Join("testdata", "imagePullPolicy-all-plugins.golden"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sbc, err := client.NewSonobuoyClient(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := sbc.GenerateManifest(tc.inputcm)
			switch {
			case err != nil && len(tc.expectErr) == 0:
				t.Fatalf("Expected nil error but got %q", err)
			case err != nil && len(tc.expectErr) > 0:
				if fmt.Sprint(err) != tc.expectErr {
					t.Errorf("Expected error \n\t%q\nbut got\n\t%q", tc.expectErr, err)
				}
				return
			case err == nil && len(tc.expectErr) > 0:
				t.Fatalf("Expected error %q but got nil", tc.expectErr)
			default:
				// No error
			}

			if *update {
				ioutil.WriteFile(tc.goldenFile, manifest, 0666)
			} else {
				fileData, err := ioutil.ReadFile(tc.goldenFile)
				if err != nil {
					t.Fatalf("Failed to read golden file %v: %v", tc.goldenFile, err)
				}
				if !bytes.Equal(fileData, manifest) {
					t.Errorf("Expected manifest to equal goldenfile: %v but instead got: %v", tc.goldenFile, string(manifest))
				}
			}
		})
	}
}

func TestGenerateManifestInvalidConfig(t *testing.T) {
	testcases := []struct {
		desc             string
		config           *client.GenConfig
		expectedErrorMsg string
	}{
		{
			desc:             "Passing a nil config results in an error",
			config:           nil,
			expectedErrorMsg: "nil GenConfig provided",
		},
	}

	c, err := client.NewSonobuoyClient(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			expectedError := len(tc.expectedErrorMsg) > 0
			_, err = c.GenerateManifest(tc.config)
			if !expectedError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if expectedError {
				if err == nil {
					t.Errorf("Expected provided config to be invalid but got no error")
				} else if !strings.Contains(err.Error(), tc.expectedErrorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tc.expectedErrorMsg, err.Error())
				}
			}
		})
	}
}
