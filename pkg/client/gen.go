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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	"github.com/heptio/sonobuoy/pkg/templates"

	corev1 "k8s.io/api/core/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	e2ePluginName    = "e2e"
	systemdLogsName  = "systemd-logs"
	pluginResultsDir = "/tmp/results"
)

// templateValues are used for direct template substitution for manifest generation.
type templateValues struct {
	Plugins []string

	SonobuoyConfig  string
	SonobuoyImage   string
	Namespace       string
	EnableRBAC      bool
	ImagePullPolicy string
	SSHKey          string
	SSHUser         string

	// CustomRegistries should be a multiline yaml string which represents
	// the file contents of KUBE_TEST_REPO_LIST, the overrides for k8s e2e
	// registries.
	CustomRegistries string
}

// GenerateManifest fills in a template with a Sonobuoy config
func (*SonobuoyClient) GenerateManifest(cfg *GenConfig) ([]byte, error) {
	marshalledConfig, err := json.Marshal(cfg.Config)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't marshall selector")
	}

	sshKeyData := []byte{}
	if len(cfg.SSHKeyPath) > 0 {
		var err error
		sshKeyData, err = ioutil.ReadFile(cfg.SSHKeyPath)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read SSH key file: %v", cfg.SSHKeyPath)
		}
	}

	// Support legacy logic for the time being.
	if len(cfg.DynamicPlugins) == 0 && len(cfg.StaticPlugins) == 0 {
		if cfg.Config.PluginSelections != nil {
			// Empty (but non-nil) means run nothing. Setting any value means run
			// those explicitly.
			for _, v := range cfg.Config.PluginSelections {
				cfg.DynamicPlugins = append(cfg.DynamicPlugins, v.Name)
			}
		} else {
			// Nil plugin selection now means to run all plugins that are loaded.
			// If the user didnt provide plugins at all fallback to our original
			// defaults.
			cfg.DynamicPlugins = []string{e2ePluginName, systemdLogsName}
		}
	}

	plugins := []*manifest.Manifest{}
	for _, v := range cfg.DynamicPlugins {
		switch v {
		case e2ePluginName:
			plugins = append(plugins, e2eManifest(cfg))
		case systemdLogsName:
			plugins = append(plugins, systemdLogsManifest(cfg))
		}
	}
	plugins = append(plugins, cfg.StaticPlugins...)

	sort.Slice(plugins, func(i, j int) bool {
		return strings.ToLower(plugins[i].SonobuoyConfig.PluginName) < strings.ToLower(plugins[j].SonobuoyConfig.PluginName)
	})

	err = checkPluginsUnique(plugins)
	if err != nil {
		return nil, errors.Wrap(err, "plugin YAML generation")
	}

	pluginYAML := []string{}
	for _, v := range plugins {
		yaml, err := kuberuntime.Encode(manifest.Encoder, v)
		if err != nil {
			return nil, errors.Wrapf(err, "serializing plugin %v as YAML", v.SonobuoyConfig.PluginName)
		}
		pluginYAML = append(pluginYAML, strings.TrimSpace(string(yaml)))
	}

	tmplVals := &templateValues{
		SonobuoyConfig:  string(marshalledConfig),
		SonobuoyImage:   cfg.Image,
		Namespace:       cfg.Namespace,
		EnableRBAC:      cfg.EnableRBAC,
		ImagePullPolicy: cfg.ImagePullPolicy,
		SSHKey:          base64.StdEncoding.EncodeToString(sshKeyData),
		SSHUser:         cfg.SSHUser,

		Plugins: pluginYAML,

		// Often created from reading a file, this value could have trailing newline.
		CustomRegistries: strings.TrimSpace(cfg.E2EConfig.CustomRegistries),
	}

	var buf bytes.Buffer

	if err := templates.Manifest.Execute(&buf, tmplVals); err != nil {
		return nil, errors.Wrap(err, "couldn't execute manifest template")
	}

	return buf.Bytes(), nil
}

func checkPluginsUnique(plugins []*manifest.Manifest) error {
	names := map[string]struct{}{}
	for _, v := range plugins {
		if _, exists := names[v.SonobuoyConfig.PluginName]; exists {
			return fmt.Errorf("plugin names must be unique, got duplicated plugin name '%v'", v.SonobuoyConfig.PluginName)
		}
		names[v.SonobuoyConfig.PluginName] = struct{}{}
	}
	return nil
}

func includes(set []plugin.Selection, s string) bool {
	for _, v := range set {
		if s == v.Name {
			return true
		}
	}
	return false
}

func systemdLogsManifest(cfg *GenConfig) *manifest.Manifest {
	trueVal := true
	return &manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName: "systemd-logs",
			Driver:     "DaemonSet",
			ResultType: "systemd-logs",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name:            "systemd-logs",
				Image:           "gcr.io/heptio-images/sonobuoy-plugin-systemd-logs:latest",
				Command:         []string{"/bin/sh", "-c", "/get_systemd_logs.sh && sleep 3600"},
				ImagePullPolicy: corev1.PullPolicy(cfg.ImagePullPolicy),
				Env: []corev1.EnvVar{
					corev1.EnvVar{Name: "CHROOT_DIR", Value: "/node"},
					corev1.EnvVar{Name: "RESULTS_DIR", Value: pluginResultsDir},
					corev1.EnvVar{Name: "NODE_NAME",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
						},
					},
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &trueVal,
				},
				VolumeMounts: []corev1.VolumeMount{
					corev1.VolumeMount{
						ReadOnly:  false,
						Name:      "results",
						MountPath: pluginResultsDir,
					}, corev1.VolumeMount{
						ReadOnly:  false,
						Name:      "root",
						MountPath: "/node",
					},
				},
			},
		},
	}
}

func e2eManifest(cfg *GenConfig) *manifest.Manifest {
	m := &manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName: "e2e",
			Driver:     "Job",
			ResultType: "e2e",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name:            "e2e",
				Image:           cfg.KubeConformanceImage,
				Command:         []string{"/run_e2e.sh"},
				ImagePullPolicy: corev1.PullPolicy(cfg.ImagePullPolicy),
				Env: []corev1.EnvVar{
					corev1.EnvVar{Name: "E2E_FOCUS", Value: cfg.E2EConfig.Focus},
					corev1.EnvVar{Name: "E2E_SKIP", Value: cfg.E2EConfig.Skip},
					corev1.EnvVar{Name: "E2E_PARALLEL", Value: cfg.E2EConfig.Parallel},
				},
				VolumeMounts: []corev1.VolumeMount{
					corev1.VolumeMount{
						ReadOnly:  false,
						Name:      "results",
						MountPath: pluginResultsDir,
					},
				},
			},
		},
	}

	// Add volume mount, volume, and env var for custom registries.
	if len(cfg.E2EConfig.CustomRegistries) > 0 {
		m.ExtraVolumes = append(m.ExtraVolumes, manifest.Volume{
			Volume: corev1.Volume{
				Name: "repolist-vol",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "repolist-cm"},
					},
				},
			},
		})
		m.Spec.Env = append(m.Spec.Env, corev1.EnvVar{
			Name: "KUBE_TEST_REPO_LIST", Value: "/tmp/sonobuoy/repo-list.yaml",
		})
		m.Spec.VolumeMounts = append(m.Spec.VolumeMounts,
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      "repolist-vol",
				MountPath: "/tmp/sonobuoy",
			},
		)
	}

	// Add volume mount, volume, and 3 env vars (for different possible platforms) for SSH capabilities.
	if len(cfg.SSHKeyPath) > 0 {
		defMode := int32(256)
		m.ExtraVolumes = append(m.ExtraVolumes, manifest.Volume{
			Volume: corev1.Volume{
				Name: "sshkey-vol",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  "ssh-key",
						DefaultMode: &defMode,
					},
				},
			},
		})
		m.Spec.Env = append(m.Spec.Env,
			corev1.EnvVar{Name: "LOCAL_SSH_KEY", Value: "id_rsa"},
			corev1.EnvVar{Name: "AWS_SSH_KEY", Value: "/root/.ssh/id_rsa"},
			corev1.EnvVar{Name: "KUBE_SSH_KEY", Value: "id_rsa"},
		)
		m.Spec.VolumeMounts = append(m.Spec.VolumeMounts,
			corev1.VolumeMount{
				ReadOnly:  false,
				Name:      "sshkey-vol",
				MountPath: "/root/.ssh",
			},
		)
	}

	if len(cfg.SSHUser) > 0 {
		m.Spec.Env = append(m.Spec.Env,
			corev1.EnvVar{Name: "KUBE_SSH_USER", Value: cfg.SSHUser},
		)
	}

	return m
}
