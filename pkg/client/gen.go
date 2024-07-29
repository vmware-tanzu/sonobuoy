/*
Copyright 2018 Heptio Inc.

Copyright 2022 the Sonobuoy Project contributors

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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/config"
	"github.com/vmware-tanzu/sonobuoy/pkg/discovery"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	kutil "github.com/vmware-tanzu/sonobuoy/pkg/k8s"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	manifesthelper "github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest/helper"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	kyaml "sigs.k8s.io/yaml"
)

const (
	e2ePluginName   = "e2e"
	systemdLogsName = "systemd-logs"

	aggregatorEnvOverrideKey = `sonobuoy`

	envVarKeyExtraArgs         = "E2E_EXTRA_ARGS"
	defaultImagePullSecretName = "auth-repo-cred"

	// sonobuoyKey is just a true/false env to indicate that the container was launched/tagged by Sonobuoy.
	sonobuoyKey                 = "SONOBUOY"
	sonobuoyK8sVersionKey       = "SONOBUOY_K8S_VERSION"
	sonobuoyResultsDirKey       = "SONOBUOY_RESULTS_DIR"
	sonobuoyLegacyResultsDirKey = "RESULTS_DIR"
	sonobuoyConfigDirKey        = "SONOBUOY_CONFIG_DIR"
	sonobuoyProgressPortKey     = "SONOBUOY_PROGRESS_PORT"

	sonobuoyDefaultConfigDir = "/tmp/sonobuoy/config"
)

var (
	// A few messy lines get output and this is the easiest way to remove them. Consider using
	// regexp to avoid whitespace repeats but ultimately that could be slower/more complicated.
	removeLines = [][]byte{
		// Remove creationTimestamp manually; see https://github.com/kubernetes-sigs/controller-tools/issues/402
		[]byte("\n  creationTimestamp: null\n"),
		[]byte("\n        resources: {}\n"),
		[]byte("\n      resources: {}\n"),
		[]byte("\n    resources: {}\n"),
		[]byte("\nstatus: {}\n"),
		[]byte("\nspec: {}\n"),
		[]byte("\nstatus:\n"),
		[]byte("\n  loadBalancer: {}\n"),
	}

	defaultRunAsUser  int64 = 1000
	defaultRunAsGroup int64 = 3000
	defaultFSGroup    int64 = 2000
)

// GenerateManifest fills in a template with a Sonobuoy config
func (c *SonobuoyClient) GenerateManifest(cfg *GenConfig) ([]byte, error) {
	b, _, err := c.GenerateManifestAndPlugins(cfg)
	return b, err
}

// GenerateManifestAndPlugins fills in a template with a Sonobuoy config and also provides the objects
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
	if cfg.Config == nil {
		cfg.Config = config.New()
	}

	// If the user didnt provide plugins at all fallback to our original
	// defaults. Legacy logic was that the user could specify plugins via the
	// config.PluginSelection field but since the CLI handles custom plugins
	// we moved to the list of actual plugin data. The PluginSelection is only
	// a server-side capability to run a subset of available plugins.
	if len(cfg.DynamicPlugins) == 0 && len(cfg.StaticPlugins) == 0 {
		cfg.DynamicPlugins = []string{e2ePluginName, systemdLogsName}
	}

	// Skip plugins will just remove all plugins. Only added to support a zero-plugin
	// use case where the user just wants to use Sonobuoy for data gathering.
	if cfg.Config.SkipPlugins {
		cfg.DynamicPlugins = nil
		cfg.StaticPlugins = nil
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

	// Apply our universal transforms; only override defaultImagePullPolicy if desired.
	for _, p := range plugins {
		if cfg.Config.ForceImagePullPolicy {
			p.Spec.ImagePullPolicy = corev1.PullPolicy(cfg.Config.ImagePullPolicy)
		}
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
		// Sidecars get mounts too so that they can leverage the feature too.
		if p.PodSpec != nil {
			for i := range p.PodSpec.Containers {
				p.PodSpec.Containers[i].VolumeMounts = append(p.PodSpec.Containers[i].VolumeMounts,
					corev1.VolumeMount{
						Name:      fmt.Sprintf("sonobuoy-%v-vol", p.SonobuoyConfig.PluginName),
						MountPath: sonobuoyDefaultConfigDir,
					})
			}
		}
		p.Spec.VolumeMounts = append(p.Spec.VolumeMounts,
			corev1.VolumeMount{
				Name:      fmt.Sprintf("sonobuoy-%v-vol", p.SonobuoyConfig.PluginName),
				MountPath: sonobuoyDefaultConfigDir,
			},
		)
	}

	err := checkPluginNames(plugins)
	if err != nil {
		return nil, nil, errors.Wrap(err, "plugin YAML generation")
	}

	cfg.PluginEnvOverrides, plugins = applyAutoEnvVars(cfg.KubeVersion, cfg.Config.ResultsDir, cfg.Config.ProgressUpdatesPort, cfg.PluginEnvOverrides, plugins)
	discovery.AutoAttachResultsDir(plugins, cfg.Config.ResultsDir)
	if err := applyEnvOverrides(cfg.PluginEnvOverrides, plugins); err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	if err := generateYAMLComponents(&buf, cfg, plugins, configs); err != nil {
		return nil, nil, errors.Wrap(err, "failed to generate YAML from configuration")
	}

	return buf.Bytes(), plugins, nil
}

func generateYAMLComponents(w io.Writer, cfg *GenConfig, plugins []*manifest.Manifest, configs map[string]map[string]string) error {
	if err := generateNS(w, *cfg); err != nil {
		return err
	}
	if err := generateServiceAcct(w, cfg); err != nil {
		return err
	}
	if cfg.Config.E2EDockerConfigFile != "" {
		if err := generateRegistrySecret(w, cfg); err != nil {
			return err
		}
	}
	if err := generateRBAC(w, cfg); err != nil {
		return err
	}
	if err := generateConfigMap(w, cfg); err != nil {
		return err
	}
	if err := generateSecret(w, cfg); err != nil {
		return err
	}

	if err := generatePluginConfigmap(w, cfg, plugins); err != nil {
		return err
	}
	if err := generateAdditionalConfigmaps(w, cfg, configs); err != nil {
		return err
	}
	if err := generateAggregatorAndService(w, cfg); err != nil {
		return err
	}

	return nil
}

func generateAdditionalConfigmaps(w io.Writer, cfg *GenConfig, configs map[string]map[string]string) error {
	// Must iterate through in a predictable manner.
	pluginNames := []string{}
	for pluginName := range configs {
		pluginNames = append(pluginNames, pluginName)
	}
	sort.Strings(pluginNames)

	for _, pluginName := range pluginNames {
		cm := &corev1.ConfigMap{Data: map[string]string{}}
		cm.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
		cm.Name = fmt.Sprintf("plugin-%v-cm", pluginName)
		cm.Namespace = cfg.Config.Namespace

		filenames := []string{}
		for filename := range configs[pluginName] {
			filenames = append(filenames, filename)
		}
		sort.Strings(filenames)
		for _, filename := range filenames {
			cm.Data[filename] = configs[pluginName][filename]

		}

		if err := appendAsYAML(w, cm); err != nil {
			return err
		}
	}
	return nil
}

func generatePluginConfigmap(w io.Writer, cfg *GenConfig, plugins []*manifest.Manifest) error {
	cm := &corev1.ConfigMap{Data: map[string]string{}}
	cm.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
	cm.Name = "sonobuoy-plugins-cm"
	cm.Namespace = cfg.Config.Namespace
	cm.Labels = map[string]string{"component": "sonobuoy"}
	for i, p := range plugins {
		b, err := manifesthelper.ToYAML(p, cfg.ShowDefaultPodSpec)
		if err != nil {
			return errors.Wrapf(err, "failed to serialize plugin %v", p.SonobuoyConfig.PluginName)
		}
		cm.Data[fmt.Sprintf("plugin-%v.yaml", i)] = strings.TrimSpace(string(b))
	}

	return appendAsYAML(w, cm)
}

func objToYAML(w io.Writer, o kuberuntime.Object) error {
	output, err := kyaml.Marshal(o)
	if err != nil {
		return err
	}

	// We currently use bytes.Replace for ease/speed but we have to repeat extra lines due to varying whitespace.
	// May consider doing regexp but it will probably result in it being slower/more
	for _, removeLine := range removeLines {
		output = bytes.ReplaceAll(output, removeLine, []byte{'\n'})
	}

	_, err = fmt.Fprint(w, string(output))
	return err
}

func appendAsYAML(w io.Writer, o kuberuntime.Object) error {
	if err := objToYAML(w, o); err != nil {
		return err
	}
	_, err := w.Write([]byte("---\n"))
	return err
}

func generateRegistrySecret(w io.Writer, cfg *GenConfig) error {
	contents, err := os.ReadFile(cfg.Config.E2EDockerConfigFile)
	if err != nil {
		return fmt.Errorf("error reading docker config file: %v", err)
	}
	s := &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{corev1.DockerConfigJsonKey: contents},
	}
	s.Name = cfg.Config.ImagePullSecrets
	s.Namespace = cfg.Config.Namespace

	s.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})

	return appendAsYAML(w, s)

}

func generateSecret(w io.Writer, cfg *GenConfig) error {
	if len(cfg.SSHKeyPath) == 0 {
		return nil
	}

	sshKeyData, err := os.ReadFile(cfg.SSHKeyPath)
	if err != nil {
		return errors.Wrapf(err, "unable to read SSH key file: %v", cfg.SSHKeyPath)
	}

	s := &corev1.Secret{
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"id_rsa": []byte(base64.StdEncoding.EncodeToString(sshKeyData))},
	}
	s.Name = "ssh-key"
	s.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"})

	return appendAsYAML(w, s)
}

func generateAggregatorAndService(w io.Writer, cfg *GenConfig) error {
	p := &corev1.Pod{}
	p.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
	p.Name = "sonobuoy"
	p.Namespace = cfg.Config.Namespace
	p.Labels = map[string]string{
		"component":          "sonobuoy",
		"sonobuoy-component": "aggregator",
		"tier":               "analysis",
	}
	p.Spec = corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:            "kube-sonobuoy",
				Command:         []string{"/sonobuoy"},
				Args:            []string{"aggregator", "--no-exit", fmt.Sprintf("--level=%v", logrus.GetLevel().String()), fmt.Sprintf("-v=%v", errlog.GetLevelForGlog()), "--alsologtostderr"},
				Image:           cfg.Config.WorkerImage,
				ImagePullPolicy: corev1.PullPolicy(cfg.ImagePullPolicy),
				VolumeMounts: []corev1.VolumeMount{
					{MountPath: "/etc/sonobuoy", Name: "sonobuoy-config-volume"},
					{MountPath: "/plugins.d", Name: "sonobuoy-plugins-volume"},
					{MountPath: config.AggregatorResultsPath, Name: "output-volume"},
				},
				Env: []corev1.EnvVar{
					{
						Name:      "SONOBUOY_ADVERTISE_IP",
						ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}},
					},
				},
			},
		},
		Volumes: []corev1.Volume{
			{Name: "sonobuoy-config-volume", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "sonobuoy-config-cm"}}}},
			{Name: "sonobuoy-plugins-volume", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "sonobuoy-plugins-cm"}}}},
			{Name: "output-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		},
		ServiceAccountName: cfg.Config.ServiceAccountName,
		Tolerations: []corev1.Toleration{
			{Key: "kubernetes.io/e2e-evict-taint-key", Operator: corev1.TolerationOpExists},
		},
		RestartPolicy: corev1.RestartPolicyNever,
	}
	if len(cfg.Config.ImagePullSecrets) > 0 {
		p.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: cfg.Config.ImagePullSecrets}}
	}
	if len(cfg.NodeSelectors) > 0 {
		p.Spec.NodeSelector = cfg.NodeSelectors
	}
	if len(cfg.Config.CustomAnnotations) > 0 {
		p.ObjectMeta.Annotations = cfg.Config.CustomAnnotations
	}
	if len(cfg.Config.AggregatorTolerations) > 0 {
		for _, t := range cfg.Config.AggregatorTolerations {
			var toleration corev1.Toleration
			if val, exists := t["key"]; exists {
				toleration.Key = val
			}
			if val, exists := t["value"]; exists {
				toleration.Value = val
			}
			if val, exists := t["effect"]; exists {
				if val == "NoSchedule" {
					toleration.Effect = corev1.TaintEffectNoSchedule
				} else if val == "NoExecute" {
					toleration.Effect = corev1.TaintEffectNoExecute
				} else if val == "PreferNoSchedule" {
					toleration.Effect = corev1.TaintEffectPreferNoSchedule
				} else {
					return errors.New("Invalid effect: " + val)
				}
			}
			if val, exists := t["operator"]; exists {
				if val == "Equal" {
					toleration.Operator = corev1.TolerationOpEqual
				} else if val == "Exists" {
					toleration.Operator = corev1.TolerationOpExists
				} else {
					return errors.New("Invalid operator: " + val)
				}
			}
			p.Spec.Tolerations = append(p.Spec.Tolerations, toleration)
		}
	}

	switch cfg.Config.SecurityContextMode {
	case "none":
	case "nonroot":
		p.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  &defaultRunAsUser,
			RunAsGroup: &defaultRunAsGroup,
			FSGroup:    &defaultFSGroup,
		}
	default:
		p.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  &defaultRunAsUser,
			RunAsGroup: &defaultRunAsGroup,
			FSGroup:    &defaultFSGroup,
		}
	}
	if len(cfg.PluginEnvOverrides) > 0 && cfg.PluginEnvOverrides[aggregatorEnvOverrideKey] != nil {
		newEnv, removeVals := processEnvVals(cfg.PluginEnvOverrides[aggregatorEnvOverrideKey])
		p.Spec.Containers[0].Env = kutil.MergeEnv(newEnv, p.Spec.Containers[0].Env, removeVals)
	}
	ser := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"sonobuoy-component": "aggregator"},
			Type:     "ClusterIP",
			Ports: []corev1.ServicePort{
				{
					Port:       8080,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{IntVal: 8080},
				},
			},
		},
	}
	ser.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"})
	ser.Name = "sonobuoy-aggregator"
	ser.Namespace = cfg.Config.Namespace
	ser.Labels = map[string]string{
		"component":          "sonobuoy",
		"sonobuoy-component": "aggregator",
	}
	if err := appendAsYAML(w, p); err != nil {
		return err
	}
	return appendAsYAML(w, ser)
}

func generateConfigMap(w io.Writer, cfg *GenConfig) error {
	// No error path here for better organization; unlikely this will happen
	// and the user gets an error message and their run will fail in a pretty clear manner.
	marshalledConfig, err := json.Marshal(cfg.Config)
	if err != nil {
		return errors.Wrap(err, "unable to marshal config into JSON")
	}
	cm := &corev1.ConfigMap{}
	cm.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"})
	cm.Name = "sonobuoy-config-cm"
	cm.Namespace = cfg.Config.Namespace
	cm.Labels = map[string]string{"component": "sonobuoy"}
	cm.Data = map[string]string{"config.json": string(marshalledConfig)}
	return appendAsYAML(w, cm)
}

func clusterAdminRBAC(w io.Writer, cfg *GenConfig) error {
	cr, crb := &v1.ClusterRole{}, &v1.ClusterRoleBinding{}
	cr.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"})
	crb.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"})

	crb.Name = fmt.Sprintf("sonobuoy-serviceaccount-%v", cfg.Config.Namespace)
	crb.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	crb.RoleRef = v1.RoleRef{
		Name:     fmt.Sprintf("sonobuoy-serviceaccount-%v", cfg.Config.Namespace),
		Kind:     "ClusterRole",
		APIGroup: "rbac.authorization.k8s.io",
	}
	crb.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "sonobuoy-serviceaccount",
			Namespace: cfg.Config.Namespace,
		},
	}
	cr.Name = fmt.Sprintf("sonobuoy-serviceaccount-%v", cfg.Config.Namespace)
	cr.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	cr.Rules = []v1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
		{
			NonResourceURLs: []string{"/metrics", "/logs", "/logs/*"},
			Verbs:           []string{"get"},
		},
	}
	if err := appendAsYAML(w, crb); err != nil {
		return err
	}
	return appendAsYAML(w, cr)
}

func clusterReadRBAC(w io.Writer, cfg *GenConfig) error {
	cr, crb := &v1.ClusterRole{}, &v1.ClusterRoleBinding{}
	cr.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"})
	crb.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"})

	crb.Name = fmt.Sprintf("sonobuoy-serviceaccount-%v", cfg.Config.Namespace)
	crb.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	crb.RoleRef = v1.RoleRef{
		Name:     fmt.Sprintf("sonobuoy-serviceaccount-%v", cfg.Config.Namespace),
		Kind:     "ClusterRole",
		APIGroup: "rbac.authorization.k8s.io",
	}
	crb.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "sonobuoy-serviceaccount",
			Namespace: cfg.Config.Namespace,
		},
	}
	cr.Name = fmt.Sprintf("sonobuoy-serviceaccount-%v", cfg.Config.Namespace)
	cr.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	cr.Rules = []v1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			NonResourceURLs: []string{"/metrics", "/logs", "/logs/*"},
			Verbs:           []string{"get"},
		},
	}
	if err := appendAsYAML(w, crb); err != nil {
		return err
	}
	if err := appendAsYAML(w, cr); err != nil {
		return err
	}
	r, rb := &v1.Role{}, &v1.RoleBinding{}
	r.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"})
	rb.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"})
	rb.Name = "sonobuoy-serviceaccount-sonobuoy"
	rb.Namespace = cfg.Config.Namespace
	rb.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	rb.RoleRef = v1.RoleRef{
		Name:     "sonobuoy-serviceaccount-sonobuoy",
		Kind:     "Role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	rb.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "sonobuoy-serviceaccount",
			Namespace: cfg.Config.Namespace,
		},
	}
	r.Name = "sonobuoy-serviceaccount-sonobuoy"
	r.Namespace = cfg.Config.Namespace
	r.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	r.Rules = []v1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
	}
	if err := appendAsYAML(w, rb); err != nil {
		return err
	}
	return appendAsYAML(w, r)
}
func namespaceAdminRBAC(w io.Writer, cfg *GenConfig) error {
	r, rb := &v1.Role{}, &v1.RoleBinding{}
	r.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"})
	rb.SetGroupVersionKind(schema.GroupVersionKind{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"})
	rb.Name = "sonobuoy-serviceaccount-sonobuoy"
	rb.Namespace = cfg.Config.Namespace
	rb.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	rb.RoleRef = v1.RoleRef{
		Name:     "sonobuoy-serviceaccount-sonobuoy",
		Kind:     "Role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	rb.Subjects = []v1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "sonobuoy-serviceaccount",
			Namespace: cfg.Config.Namespace,
		},
	}
	r.Name = "sonobuoy-serviceaccount-sonobuoy"
	r.Namespace = cfg.Config.Namespace
	r.Labels = map[string]string{clusterRoleFieldName: clusterRoleFieldValue, clusterRoleFieldNamespace: cfg.Config.Namespace}
	r.Rules = []v1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
	}
	if err := appendAsYAML(w, rb); err != nil {
		return err
	}
	return appendAsYAML(w, r)
}

func generateRBAC(w io.Writer, cfg *GenConfig) error {
	if !cfg.EnableRBAC {
		return nil
	}

	switch cfg.Config.AggregatorPermissions {
	case config.AggregatorPermissionsClusterAdmin:
		return clusterAdminRBAC(w, cfg)
	case config.AggregatorPermissionsNamespaceAdmin:
		return namespaceAdminRBAC(w, cfg)
	case config.AggregatorPermissionsClusterRead:
		return clusterReadRBAC(w, cfg)
	default:
		return fmt.Errorf("unknown aggregator permission: %v", cfg.Config.AggregatorPermissions)
	}
}

func generateServiceAcct(w io.Writer, cfg *GenConfig) error {
	if cfg.Config.ExistingServiceAccount {
		return nil
	}

	sa := &corev1.ServiceAccount{}
	sa.Name = cfg.Config.ServiceAccountName
	sa.Namespace = cfg.Config.Namespace
	sa.Labels = map[string]string{"component": "sonobuoy"}
	sa.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"})
	return appendAsYAML(w, sa)
}

func generateNS(w io.Writer, cfg GenConfig) error {
	if cfg.Config.AggregatorPermissions != config.AggregatorPermissionsClusterAdmin {
		return nil
	}

	labels := make(map[string]string)
	labels["pod-security.kubernetes.io/enforce"] = cfg.Config.NamespacePSAEnforceLevel
	ns := &corev1.Namespace{}
	ns.Name = cfg.Config.Namespace
	ns.Labels = labels
	ns.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})
	return appendAsYAML(w, ns)
}

func applyEnvOverrides(pluginEnvOverrides map[string]map[string]string, plugins []*manifest.Manifest) error {
	for pluginName, envVars := range pluginEnvOverrides {
		found := false
		for _, p := range plugins {
			if p.SonobuoyConfig.PluginName == pluginName {
				found = true
				newEnv, removeVals := processEnvVals(envVars)
				p.Spec.Env = kutil.MergeEnv(newEnv, p.Spec.Env, removeVals)
				if p.PodSpec != nil {
					for i := range p.PodSpec.Containers {
						p.PodSpec.Containers[i].Env = kutil.MergeEnv(newEnv, p.PodSpec.Containers[i].Env, removeVals)
					}
				}
			}
		}

		// Require overrides to target existing plugins and provide a helpful message if there is a mismatch.
		// Dont error if the plugin in question is "e2e" since we default to setting those values regardless of
		// if they choose that plugin or not.
		if !found && pluginName != e2ePluginName && pluginName != aggregatorEnvOverrideKey {
			pluginNames := []string{}
			for _, p := range plugins {
				pluginNames = append(pluginNames, p.SonobuoyConfig.PluginName)
			}

			return fmt.Errorf("failed to override env vars for plugin %v, no plugin with that name found; have plugins: %v", pluginName, pluginNames)
		}
	}
	return nil
}

// processEnvVals takes a map of key/value pairs representing env vars (from the --plugin-env flag)
// and processes them to represent which envs should be added vs removed. These can then be
// used with the mergeEnv function as needed to assign them into a pod.
func processEnvVals(envVars map[string]string) ([]corev1.EnvVar, map[string]struct{}) {
	newEnv := []corev1.EnvVar{}
	removeVals := map[string]struct{}{}
	for k, v := range envVars {
		if v != "" {
			newEnv = append(newEnv, corev1.EnvVar{Name: k, Value: v})
		} else {
			removeVals[k] = struct{}{}
		}
	}
	return newEnv, removeVals
}

func applyAutoEnvVars(imageVersion, resultsDir, progressPort string, env map[string]map[string]string, plugins []*manifest.Manifest) (map[string]map[string]string, []*manifest.Manifest) {
	// Set env on all plugins and swap out dynamic images.
	if env == nil {
		env = map[string]map[string]string{}
	}
	for i, p := range plugins {
		if env[p.SonobuoyConfig.PluginName] == nil {
			env[p.SonobuoyConfig.PluginName] = map[string]string{}
		}
		env[p.SonobuoyConfig.PluginName][sonobuoyK8sVersionKey] = imageVersion
		env[p.SonobuoyConfig.PluginName][sonobuoyResultsDirKey] = resultsDir
		env[p.SonobuoyConfig.PluginName][sonobuoyLegacyResultsDirKey] = resultsDir
		env[p.SonobuoyConfig.PluginName][sonobuoyProgressPortKey] = progressPort
		env[p.SonobuoyConfig.PluginName][sonobuoyConfigDirKey] = sonobuoyDefaultConfigDir
		env[p.SonobuoyConfig.PluginName][sonobuoyKey] = "true"
		plugins[i].Spec.Image = strings.ReplaceAll(plugins[i].Spec.Image, "$"+sonobuoyK8sVersionKey, imageVersion)
	}
	return env, plugins
}

func checkPluginNames(plugins []*manifest.Manifest) error {
	names := map[string]struct{}{}
	for _, v := range plugins {
		if v.SonobuoyConfig.PluginName == aggregatorEnvOverrideKey {
			return fmt.Errorf("plugin name %q is a reserved value", aggregatorEnvOverrideKey)
		}
		if _, exists := names[v.SonobuoyConfig.PluginName]; exists {
			return fmt.Errorf("plugin names must be unique, got duplicated plugin name '%v'", v.SonobuoyConfig.PluginName)
		}
		names[v.SonobuoyConfig.PluginName] = struct{}{}
	}
	return nil
}

func SystemdLogsManifest(cfg *GenConfig) *manifest.Manifest {
	trueVal := true
	m := &manifest.Manifest{
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
					{Name: "RESULTS_DIR", Value: cfg.Config.ResultsDir},
					{Name: "SONOBUOY_RESULTS_DIR", Value: cfg.Config.ResultsDir},
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
						Name:      "root",
						MountPath: "/node",
					},
				},
			},
		},
	}

	m.PodSpec = &manifest.PodSpec{
		PodSpec: driver.DefaultPodSpec(m.SonobuoyConfig.Driver),
	}
	// systemd-logs only makes sense on linux.
	// TODO(jschnake): Instead of systemd-logs, make an os-agnostic log gathering plugin.
	m.PodSpec.PodSpec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}

	return m
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
			},
		},
	}
	m.PodSpec = &manifest.PodSpec{
		PodSpec: driver.DefaultPodSpec(m.SonobuoyConfig.Driver),
	}
	m.PodSpec.PodSpec.NodeSelector = map[string]string{"kubernetes.io/os": "linux"}

	m.Spec.Env = updateExtraArgs(m.Spec.Env, cfg.Config.ProgressUpdatesPort, cfg.Config.E2EDockerConfigFile)

	return m
}

// updateExtraArgs adds the flag expected by the e2e plugin for the progress report URL.
// If no port is given, the default "8099" is used.
func updateExtraArgs(envs []corev1.EnvVar, port, e2eDockerConfigFile string) []corev1.EnvVar {
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
	if e2eDockerConfigFile != "" {
		credFile := filepath.Base(e2eDockerConfigFile)
		registryCredLocation := fmt.Sprintf("%s/%s", sonobuoyDefaultConfigDir, credFile)
		val += fmt.Sprintf(" --e2e-docker-config-file=%s", registryCredLocation)
	}
	envs = append(envs, corev1.EnvVar{Name: envVarKeyExtraArgs, Value: val})
	return envs
}
