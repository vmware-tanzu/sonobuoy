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

package driver

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Base is the struct that stores state for plugin drivers and contains helper methods.
type Base struct {
	Definition        manifest.Manifest
	SessionID         string
	Namespace         string
	SonobuoyImage     string
	CleanedUp         bool
	ImagePullPolicy   string
	ImagePullSecrets  string
	CustomAnnotations map[string]string
}

// GetSessionID returns the session id associated with the plugin.
func (b *Base) GetSessionID() string {
	return b.SessionID
}

// GetName returns the name of this Job plugin.
func (b *Base) GetName() string {
	return b.Definition.SonobuoyConfig.PluginName
}

// GetSecretName gets a name for a secret based on the plugin name and session ID.
func (b *Base) GetSecretName() string {
	return fmt.Sprintf("sonobuoy-plugin-%s-%s", b.GetName(), b.GetSessionID())
}

// GetDriver returns the Driver for this plugin.
func (b *Base) GetDriver() string {
	return b.Definition.SonobuoyConfig.Driver
}

// SkipCleanup returns whether cleanup for this plugin should be skipped or not.
func (b *Base) SkipCleanup() bool {
	return b.Definition.SonobuoyConfig.SkipCleanup
}

// GetResultFormat returns the ResultFormat of this plugin.
func (b *Base) GetResultFormat() string {
	return b.Definition.SonobuoyConfig.ResultFormat
}

// GetResultFiles returns the files to be post-processed for this plugin.
func (b *Base) GetResultFiles() []string {
	return b.Definition.SonobuoyConfig.ResultFiles
}

// GetSourceURL returns the sourceURL of the plugin.
func (b *Base) GetSourceURL() string {
	return b.Definition.SonobuoyConfig.SourceURL
}

// GetDescription returns the human-readable plugin description.
func (b *Base) GetDescription() string {
	return b.Definition.SonobuoyConfig.Description
}

// MakeTLSSecret makes a Kubernetes secret object for the given TLS certificate.
func (b *Base) MakeTLSSecret(cert *tls.Certificate, ownerPod *v1.Pod) (*v1.Secret, error) {
	rsaKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key not ECDSA")
	}

	if len(cert.Certificate) <= 0 {
		return nil, errors.New("no certs in tls.certificate")
	}

	certDER := cert.Certificate[0]
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM, err := getKeyPEM(rsaKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't PEM encode TLS key")
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetSecretName(),
			Namespace: b.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       ownerPod.GetName(),
					UID:        ownerPod.GetUID(),
				},
			},
		},
		Data: map[string][]byte{
			v1.TLSPrivateKeyKey: keyPEM,
			v1.TLSCertKey:       certPEM,
		},
		Type: v1.SecretTypeTLS,
	}, nil

}

// getCACertPEM extracts the CA cert from a tls.Certificate.
// If the provided Certificate has only one certificate in the chain, the CA
// will be the leaf cert.
func getCACertPEM(cert *tls.Certificate) string {
	cacert := ""
	if len(cert.Certificate) > 0 {
		caCertDER := cert.Certificate[len(cert.Certificate)-1]
		cacert = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caCertDER,
		}))
	}
	return cacert
}

// getKeyPEM turns an RSA Private Key into a PEM-encoded string
func getKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	derKEY, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derKEY,
	}), nil
}

func (b *Base) workerEnvironment(hostname string, cert *tls.Certificate, progressPort string) []v1.EnvVar {
	envVars := []v1.EnvVar{
		{
			Name: "NODE_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name:  "RESULTS_DIR",
			Value: plugin.ResultsDir,
		},
		{
			Name:  "RESULT_TYPE",
			Value: b.GetName(),
		},
		{
			Name:  "AGGREGATOR_URL",
			Value: hostname,
		},
		{
			Name:  "CA_CERT",
			Value: getCACertPEM(cert),
		},
		{
			Name: "CLIENT_CERT",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: b.GetSecretName(),
					},
					Key: "tls.crt",
				},
			},
		},
		{
			Name: "CLIENT_KEY",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: b.GetSecretName(),
					},
					Key: "tls.key",
				},
			},
		},
		{
			Name:  "SONOBUOY_PROGRESS_PORT",
			Value: progressPort,
		},
	}

	return envVars
}

// CreateWorkerContainerDefintion creates the container definition to run the Sonobuoy worker for a plugin.
func (b *Base) CreateWorkerContainerDefintion(hostname string, cert *tls.Certificate, command, args []string, progressPort string) v1.Container {
	container := v1.Container{
		Name:            "sonobuoy-worker",
		Image:           b.SonobuoyImage,
		Command:         command,
		Args:            args,
		Env:             b.workerEnvironment(hostname, cert, progressPort),
		ImagePullPolicy: v1.PullPolicy(b.ImagePullPolicy),
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "results",
				ReadOnly:  false,
				MountPath: plugin.ResultsDir,
			},
		},
	}
	return container
}

// defaultDaemonSetPodSpec returns the default PodSpec used by DaemonSet plugins
func defaultDaemonSetPodSpec() v1.PodSpec {
	podSpec := v1.PodSpec{
		Containers:         []v1.Container{},
		DNSPolicy:          v1.DNSClusterFirstWithHostNet,
		HostIPC:            true,
		HostPID:            true,
		HostNetwork:        true,
		ServiceAccountName: "sonobuoy-serviceaccount",
		Tolerations: []v1.Toleration{
			{
				Operator: v1.TolerationOpExists,
			},
		},
		Volumes: []v1.Volume{
			{
				Name: "root",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/",
					},
				},
			},
		},
	}
	return podSpec
}

// defaultJobPodSpec returns the default PodSpec used by Job plugins
func defaultJobPodSpec() v1.PodSpec {
	return v1.PodSpec{
		Containers:         []v1.Container{},
		RestartPolicy:      v1.RestartPolicyNever,
		ServiceAccountName: "sonobuoy-serviceaccount",
		Tolerations: []v1.Toleration{
			{
				Key:      "node-role.kubernetes.io/master",
				Operator: v1.TolerationOpExists,
				Effect:   v1.TaintEffectNoSchedule,
			},
			{
				Key:      "CriticalAddonsOnly",
				Operator: v1.TolerationOpExists,
			},
			{
				Key:      "kubernetes.io/e2e-evict-taint-key",
				Operator: v1.TolerationOpExists,
			},
		},
		Volumes: []v1.Volume{},

		// Default for jobs to run on linux. If a plugin can run on Windows (the more rare case)
		// they should specify it in their podSpec. This should avoid more problems than it creates.
		NodeSelector: map[string]string{
			"kubernetes.io/os": "linux",
		},
	}
}

// DefaultPodSpec returns the default pod spec used for the given plugin driver type.
func DefaultPodSpec(d string) v1.PodSpec {
	switch strings.ToLower(d) {
	case "daemonset":
		return defaultDaemonSetPodSpec()
	default:
		return defaultJobPodSpec()
	}
}
