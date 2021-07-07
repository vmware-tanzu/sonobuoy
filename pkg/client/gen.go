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

	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	manifesthelper "github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest/helper"

	corev1 "k8s.io/api/core/v1"
)

const (
	e2ePluginName   = "e2e"
	systemdLogsName = "systemd-logs"

	envVarKeyExtraArgs = "E2E_EXTRA_ARGS"

	sonobuoyK8sVersionKey = "SONOBUOY_K8S_VERSION"
)

// templateValues are used for direct template substitution for manifest generation.
type templateValues struct {
	Plugins []string

	SonobuoyConfig    string
	SonobuoyImage     string
	Namespace         string
	EnableRBAC        bool
	ImagePullPolicy   string
	ImagePullSecrets  string
	CustomAnnotations map[string]string
	SSHKey            string

	NodeSelectors map[string]string

	// configmap name, filename, string
	ConfigMaps map[string]map[string]string

	// CustomRegistries should be a multiline yaml string which represents
	// the file contents of KUBE_TEST_REPO_LIST, the overrides for k8s e2e
	// registries.
	CustomRegistries string
}

// GenerateManifest fills in a template with a Sonobuoy config
func (c *SonobuoyClient) GenerateManifest(cfg *GenConfig) ([]byte, error) {
	b, _, err := c.GenerateManifestAndPlugins(cfg)
	return b, err
}

// GenerateManifest fills in a template with a Sonobuoy config and also provides the objects
// representing the plugins. This is useful if you want to do any structured handling of the resulting
// plugins that would have been run.
func (*SonobuoyClient) GenerateManifestAndPlugins(cfg *GenConfig) ([]byte, []*manifest.Manifest, error) {
	if cfg == nil {
		return nil, nil, errors.New("nil GenConfig provided")
	}

	if err := cfg.Validate(); err != nil {
		return nil, nil, errors.Wrap(err, "config validation failed")
	}

	// Allow nil cfg.Config but avoid dereference errors.
	conf := &config.Config{}
	if cfg.Config != nil {
		conf = cfg.Config
	}

	marshalledConfig, err := json.Marshal(conf)
	if err != nil {
		return nil, nil, errors.Wrap(err, "couldn't marshall selector")
	}

	sshKeyData := []byte{}
	if len(cfg.SSHKeyPath) > 0 {
		var err error
		sshKeyData, err = ioutil.ReadFile(cfg.SSHKeyPath)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to read SSH key file: %v", cfg.SSHKeyPath)
		}
	}

	// If the user didnt provide plugins at all fallback to our original
	// defaults. Legacy logic was that the user could specify plugins via the
	// config.PluginSelection field but since the CLI handles custom plugins
	// we moved to the list of actual plugin data. The PluginSelection is only
	// a server-side capability to run a subset of available plugins.
	if len(cfg.DynamicPlugins) == 0 && len(cfg.StaticPlugins) == 0 {
		cfg.DynamicPlugins = []string{e2ePluginName, systemdLogsName}
	}

	plugins := []*manifest.Manifest{}
	for _, v := range cfg.DynamicPlugins {
		switch v {
		case e2ePluginName:
			plugins = append(plugins, E2EManifest(cfg))
		case systemdLogsName:
			plugins = append(plugins, SystemdLogsManifest(cfg))
		}
	}
	plugins = append(plugins, cfg.StaticPlugins...)

	sort.Slice(plugins, func(i, j int) bool {
		return strings.ToLower(plugins[i].SonobuoyConfig.PluginName) < strings.ToLower(plugins[j].SonobuoyConfig.PluginName)
	})

	// Apply our universal transforms; only applies to ImagePullPolicy. Overrides all
	// plugin values.
	for _, p := range plugins {
		p.Spec.ImagePullPolicy = corev1.PullPolicy(conf.ImagePullPolicy)
	}

	// Apply transforms. Ensure this is before handling configmaps and applying the k8s_version.
	for pluginName, transforms := range cfg.PluginTransforms {
		for _, p := range plugins {
			if p.SonobuoyConfig.PluginName == pluginName {
				for _, transform := range transforms {
					if err := transform(p); err != nil {
						return nil, nil, err
					}
				}
			}
		}
	}

	// If they have a configmap, associate the plugin with that configmap for mounting.
	configs := map[string]map[string]string{}
	for _, p := range plugins {
		if len(p.ConfigMap) == 0 {
			continue
		}
		configs[p.SonobuoyConfig.PluginName] = p.ConfigMap
		p.ExtraVolumes = append(p.ExtraVolumes,
			manifest.Volume{
				Volume: corev1.Volume{
					Name: fmt.Sprintf("sonobuoy-%v-vol", p.SonobuoyConfig.PluginName),
					VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: fmt.Sprintf("plugin-%v-cm", p.SonobuoyConfig.PluginName)},
					}},
				},
			})
		p.Spec.VolumeMounts = append(p.Spec.VolumeMounts,
			corev1.VolumeMount{
				Name:      fmt.Sprintf("sonobuoy-%v-vol", p.SonobuoyConfig.PluginName),
				MountPath: "/tmp/sonobuoy/config",
			},
		)
	}

	err = checkPluginsUnique(plugins)
	if err != nil {
		return nil, nil, errors.Wrap(err, "plugin YAML generation")
	}

	cfg.PluginEnvOverrides, plugins = applyK8sVersion(cfg.KubeVersion, cfg.PluginEnvOverrides, plugins)

	for pluginName, envVars := range cfg.PluginEnvOverrides {
		found := false
		for _, p := range plugins {
			if p.SonobuoyConfig.PluginName == pluginName {
				found = true
				newEnv := []corev1.EnvVar{}
				removeVals := map[string]struct{}{}
				for k, v := range envVars {
					if v != "" {
						newEnv = append(newEnv, corev1.EnvVar{Name: k, Value: v})
					} else {
						removeVals[k] = struct{}{}
					}
				}
				p.Spec.Env = mergeEnv(newEnv, p.Spec.Env, removeVals)
			}
		}

		// Require overrides to target existing plugins and provide a helpful message if there is a mismatch.
		// Dont error if the plugin in question is "e2e" since we default to setting those values regardless of
		// if they choose that plugin or not.
		if !found && pluginName != e2ePluginName {
			pluginNames := []string{}
			for _, p := range plugins {
				pluginNames = append(pluginNames, p.SonobuoyConfig.PluginName)
			}

			return nil, nil, fmt.Errorf("failed to override env vars for plugin %v, no plugin with that name found; have plugins: %v", pluginName, pluginNames)
		}
	}

	pluginYAML := []string{}
	for _, v := range plugins {
		yaml, err := manifesthelper.ToYAML(v, cfg.ShowDefaultPodSpec)
		if err != nil {
			return nil, nil, err
		}
		pluginYAML = append(pluginYAML, strings.TrimSpace(string(yaml)))
	}

	tmplVals := &templateValues{
		SonobuoyConfig:    string(marshalledConfig),
		SonobuoyImage:     conf.WorkerImage,
		Namespace:         conf.Namespace,
		EnableRBAC:        cfg.EnableRBAC,
		ImagePullPolicy:   conf.ImagePullPolicy,
		ImagePullSecrets:  conf.ImagePullSecrets,
		CustomAnnotations: conf.CustomAnnotations,
		SSHKey:            base64.StdEncoding.EncodeToString(sshKeyData),

		Plugins: pluginYAML,

		NodeSelectors: cfg.NodeSelectors,

		ConfigMaps: configs,
	}

	var buf bytes.Buffer

	if err := genManifest.Execute(&buf, tmplVals); err != nil {
		return nil, nil, errors.Wrap(err, "couldn't execute manifest template")
	}

	return buf.Bytes(), plugins, nil
}

func applyK8sVersion(imageVersion string, env map[string]map[string]string, plugins []*manifest.Manifest) (map[string]map[string]string, []*manifest.Manifest) {
	// Set env on all plugins and swap out dynamic images.
	if env == nil {
		env = map[string]map[string]string{}
	}
	for i, p := range plugins {
		if env[p.SonobuoyConfig.PluginName] == nil {
			env[p.SonobuoyConfig.PluginName] = map[string]string{}
		}
		env[p.SonobuoyConfig.PluginName][sonobuoyK8sVersionKey] = imageVersion
		plugins[i].Spec.Image = strings.ReplaceAll(plugins[i].Spec.Image, "$"+sonobuoyK8sVersionKey, imageVersion)
	}
	return env, plugins
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

// mergeEnv will combine the values from two env var sets with priority being
// given to values in the first set in case of collision. Afterwards, any env
// var with a name in the removal set will be removed.
func mergeEnv(e1, e2 []corev1.EnvVar, removeKeys map[string]struct{}) []corev1.EnvVar {
	envSet := map[string]corev1.EnvVar{}
	returnEnv := []corev1.EnvVar{}
	for _, e := range e1 {
		envSet[e.Name] = e
	}
	for _, e := range e2 {
		if _, seen := envSet[e.Name]; !seen {
			envSet[e.Name] = e
		}
	}
	for name, v := range envSet {
		if _, remove := removeKeys[name]; !remove {
			returnEnv = append(returnEnv, v)
		}
	}

	sort.Slice(returnEnv, func(i, j int) bool {
		return strings.ToLower(returnEnv[i].Name) < strings.ToLower(returnEnv[j].Name)
	})

	return returnEnv
}

func SystemdLogsManifest(cfg *GenConfig) *manifest.Manifest {
	trueVal := true
	return &manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName:   "systemd-logs",
			Driver:       "DaemonSet",
			ResultFormat: "raw",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name:            "systemd-logs",
				Image:           config.DefaultSystemdLogsImage,
				Command:         []string{"/bin/sh", "-c", `/get_systemd_logs.sh; while true; do echo "Plugin is complete. Sleeping indefinitely to avoid container exit and automatic restarts from Kubernetes"; sleep 3600; done`},
				ImagePullPolicy: corev1.PullPolicy(cfg.ImagePullPolicy),
				Env: []corev1.EnvVar{
					{Name: "CHROOT_DIR", Value: "/node"},
					{Name: "RESULTS_DIR", Value: plugin.ResultsDir},
					{Name: "NODE_NAME",
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
						},
					},
				},
				SecurityContext: &corev1.SecurityContext{
					Privileged: &trueVal,
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						ReadOnly:  false,
						Name:      "results",
						MountPath: plugin.ResultsDir,
					}, {
						ReadOnly:  false,
						Name:      "root",
						MountPath: "/node",
					},
				},
			},
		},
	}
}

func E2EManifest(cfg *GenConfig) *manifest.Manifest {
	if cfg.Config == nil {
		cfg.Config = config.New()
	}

	m := &manifest.Manifest{
		SonobuoyConfig: manifest.SonobuoyConfig{
			PluginName:   "e2e",
			Driver:       "Job",
			ResultFormat: "junit",
		},
		Spec: manifest.Container{
			Container: corev1.Container{
				Name:            "e2e",
				Image:           fmt.Sprintf("%v:%v", config.UpstreamKubeConformanceImageURL, "$SONOBUOY_K8S_VERSION"),
				Command:         []string{"/run_e2e.sh"},
				ImagePullPolicy: corev1.PullPolicy(cfg.ImagePullPolicy),
				Env: []corev1.EnvVar{
					{Name: "E2E_FOCUS", Value: cfg.PluginEnvOverrides["e2e"]["E2E_FOCUS"]},
					{Name: "E2E_SKIP", Value: cfg.PluginEnvOverrides["e2e"]["E2E_SKIP"]},
					{Name: "E2E_PARALLEL", Value: cfg.PluginEnvOverrides["e2e"]["E2E_PARALLEL"]},
					{Name: "E2E_USE_GO_RUNNER", Value: "true"},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						ReadOnly:  false,
						Name:      "results",
						MountPath: plugin.ResultsDir,
					},
				},
			},
		},
	}
	m.PodSpec = &manifest.PodSpec{
		PodSpec: driver.DefaultPodSpec(m.SonobuoyConfig.Driver),
	}
	m.PodSpec.PodSpec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}

	m.Spec.Env = updateExtraArgs(m.Spec.Env, cfg.Config.ProgressUpdatesPort)

	return m
}

// updateExtraArgs adds the flag expected by the e2e plugin for the progress report URL.
// If no port is given, the default "8099" is used.
func updateExtraArgs(envs []corev1.EnvVar, port string) []corev1.EnvVar {
	for _, env := range envs {
		// If set by user, just leave as-is.
		if env.Name == envVarKeyExtraArgs {
			return envs
		}
	}
	if port == "" {
		port = config.DefaultProgressUpdatesPort
	}
	val := fmt.Sprintf("--progress-report-url=http://localhost:%v/progress", port)
	envs = append(envs, corev1.EnvVar{Name: envVarKeyExtraArgs, Value: val})
	return envs
}
