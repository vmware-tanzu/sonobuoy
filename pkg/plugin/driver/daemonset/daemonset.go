/*
Copyright 2017 Heptio Inc.

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
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/utils"
	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	"github.com/pkg/errors"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

const clientKeyName = "clientkey"

// Plugin is a plugin driver that dispatches containers to each node,
// expecting each pod to report to the master.
type Plugin struct {
	Definition plugin.Definition
	SessionID  string
	Namespace  string
	cleanedUp  bool
}

// Ensure DaemonSetPlugin implements plugin.Interface
var _ plugin.Interface = &Plugin{}

type templateData struct {
	PluginName        string
	ResultType        string
	SessionID         string
	Namespace         string
	ProducerContainer string
	MasterAddress     string
	CACert            string
	ClientCert        string
	SecretName        string
}

// NewPlugin creates a new DaemonSet plugin from the given Plugin Definition
// and sonobuoy master address
func NewPlugin(dfn plugin.Definition, namespace string) *Plugin {
	return &Plugin{
		Definition: dfn,
		SessionID:  utils.GetSessionID(),
		Namespace:  namespace,
		cleanedUp:  false,
	}
}

// ExpectedResults returns the list of results expected for this daemonset
func (p *Plugin) ExpectedResults(nodes []v1.Node) []plugin.ExpectedResult {
	ret := make([]plugin.ExpectedResult, 0, len(nodes))

	for _, node := range nodes {
		ret = append(ret, plugin.ExpectedResult{
			NodeName:   node.Name,
			ResultType: p.GetResultType(),
		})
	}

	return ret
}

// GetResultType returns the ResultType for this plugin (to adhere to plugin.Interface)
func (p *Plugin) GetResultType() string {
	return p.Definition.ResultType
}

//FillTemplate populates the internal Job YAML template with the values for this particular job.
func (p *Plugin) FillTemplate(hostname string, cert *tls.Certificate) ([]byte, error) {
	var b bytes.Buffer
	container, err := kuberuntime.Encode(manifest.Encoder, &p.Definition.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't reserialize container for daemonset %q", p.Definition.Name)
	}

	cacert := ""
	if len(cert.Certificate) >= 2 {
		certDER := cert.Certificate[len(cert.Certificate)-1]
		cacert = string(pem.EncodeToMemory(&pem.Block{
			Type:  "Certificate",
			Bytes: certDER,
		}))
	}

	clientCert := string(pem.EncodeToMemory(&pem.Block{
		Type:  "Certificate",
		Bytes: cert.Leaf.Raw,
	}))

	vars := templateData{
		PluginName:        p.Definition.Name,
		ResultType:        p.Definition.ResultType,
		SessionID:         p.SessionID,
		Namespace:         p.Namespace,
		ProducerContainer: string(container),
		MasterAddress:     getMasterAddress(hostname),
		CACert:            cacert,
		ClientCert:        clientCert,
		SecretName:        p.getSecretName(),
	}

	if err := daemonSetTemplate.Execute(&b, vars); err != nil {
		return nil, errors.Wrapf(err, "couldn't fill template %q", p.Definition.Name)
	}
	return b.Bytes(), nil
}

// Run dispatches worker pods according to the DaemonSet's configuration.
func (p *Plugin) Run(kubeclient kubernetes.Interface, hostname string, cert *tls.Certificate) error {
	var daemonSet appsv1beta2.DaemonSet

	b, err := p.FillTemplate(hostname, cert)
	if err != nil {
		return errors.Wrap(err, "couldn't fill template")
	}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), b, &daemonSet); err != nil {
		return errors.Wrapf(err, "could not decode the executed template into a daemonset. Plugin name: ", p.GetName())
	}

	if err := p.createClientSecret(kubeclient, cert.PrivateKey); err != nil {
		return errors.Wrapf(err, "couldn't create secret for Job plugin %v", p.GetName())
	}

	// TODO(EKF): Move to v1 in 1.11
	if _, err := kubeclient.AppsV1beta2().DaemonSets(p.Namespace).Create(&daemonSet); err != nil {
		return errors.Wrapf(err, "could not create DaemonSet for daemonset plugin %v", p.GetName())
	}

	return nil
}

// Cleanup cleans up the k8s DaemonSet and ConfigMap created by this plugin instance
func (p *Plugin) Cleanup(kubeclient kubernetes.Interface) {
	p.cleanedUp = true
	gracePeriod := int64(1)
	deletionPolicy := metav1.DeletePropagationBackground

	listOptions := p.listOptions()
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &deletionPolicy,
	}

	// Delete the DaemonSet created by this plugin
	// TODO(EKF): Move to v1 in 1.11
	err := kubeclient.AppsV1beta2().DaemonSets(p.Namespace).DeleteCollection(
		&deleteOptions,
		listOptions,
	)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not delete DaemonSet-%v for daemonset plugin %v", p.SessionID, p.GetName()))
	}
}

func (p *Plugin) listOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
}

func (p *Plugin) getSecretName() string {
	return fmt.Sprintf("daemonset-%s-%s", p.GetName(), p.SessionID)
}

func (p *Plugin) createClientSecret(client kubernetes.Interface, key crypto.PrivateKey) error {
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return errors.New("private key not RSA")
	}

	bytes := x509.MarshalPKCS1PrivateKey(rsaKey)
	secretName := p.getSecretName()

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: p.Namespace,
		},
		Data: map[string][]byte{clientKeyName: bytes},
	}
	_, err := client.CoreV1().Secrets(p.Namespace).Create(secret)
	return errors.Wrap(err, "couldn't create TLS client secret")
}

// findDaemonSet gets the daemonset that we created, using a kubernetes label search
func (p *Plugin) findDaemonSet(kubeclient kubernetes.Interface) (*appsv1beta2.DaemonSet, error) {
	// TODO(EKF): Move to v1 in 1.11
	dsets, err := kubeclient.AppsV1beta2().DaemonSets(p.Namespace).List(p.listOptions())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(dsets.Items) != 1 {
		return nil, errors.Errorf("expected plugin %v to create 1 daemonset, found %v", p.Definition.Name, len(dsets.Items))
	}

	return &dsets.Items[0], nil
}

// Monitor adheres to plugin.Interface by ensuring the DaemonSet is correctly
// configured and that each pod is running normally.
func (p *Plugin) Monitor(kubeclient kubernetes.Interface, availableNodes []v1.Node, resultsCh chan<- *plugin.Result) {
	podsReported := make(map[string]bool)
	podsFound := make(map[string]bool, len(availableNodes))
	for _, node := range availableNodes {
		podsFound[node.Name] = false
		podsReported[node.Name] = false
	}

	for {
		// Sleep between each poll, which should give the DaemonSet
		// enough time to create pods
		time.Sleep(10 * time.Second)
		// If we've cleaned up after ourselves, stop monitoring
		if p.cleanedUp {
			break
		}

		// If we don't have a daemonset created, retry next time.  We
		// only send errors if we successfully see that an expected pod
		// is having issues.
		ds, err := p.findDaemonSet(kubeclient)
		if err != nil {
			errlog.LogError(errors.Wrapf(err, "could not find DaemonSet created by plugin %v, will retry", p.GetName()))
			continue
		}

		// Find all the pods configured by this daemonset
		pods, err := kubeclient.CoreV1().Pods(p.Namespace).List(p.listOptions())
		if err != nil {
			errlog.LogError(errors.Wrapf(err, "could not find pods created by plugin %v, will retry", p.GetName()))
			// Likewise, if we can't query for pods, just retry next time.
			continue
		}

		// Cycle through each pod in this daemonset, reporting any failures.
		for _, pod := range pods.Items {
			nodeName := pod.Spec.NodeName
			// We don't care about nodes we already saw
			if podsReported[nodeName] {
				continue
			}

			podsFound[nodeName] = true
			// Check if it's failing and submit the error result
			if isFailing, reason := utils.IsPodFailing(&pod); isFailing {
				podsReported[nodeName] = true

				resultsCh <- utils.MakeErrorResult(p.GetResultType(), map[string]interface{}{
					"error": reason,
					"pod":   pod,
				}, nodeName)
			}
		}

		// DaemonSets are a bit strange, if node taints are preventing
		// scheduling, pods won't even be created (unlike say Jobs,
		// which will create the pod and leave it in an unscheduled
		// state.)  So take any nodes we didn't see pods on, and report
		// issues scheduling them.
		for _, node := range availableNodes {
			if !podsFound[node.Name] && !podsReported[node.Name] {
				podsReported[node.Name] = true
				resultsCh <- utils.MakeErrorResult(p.GetResultType(), map[string]interface{}{
					"error": fmt.Sprintf(
						"No pod was scheduled on node %v within %v. Check tolerations for plugin %v",
						node.Name,
						time.Now().Sub(ds.CreationTimestamp.Time),
						p.Definition.Name,
					),
				}, node.Name)
			}
		}
	}
}

func (p *Plugin) GetSessionID() string {
	return p.SessionID
}

// GetName returns the name of this DaemonSet plugin
func (p *Plugin) GetName() string {
	return p.Definition.Name
}

func getMasterAddress(hostname string) string {
	return fmt.Sprintf("http://%s/api/v1/results/by-node", hostname)
}
