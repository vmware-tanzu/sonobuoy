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
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/driver/utils"
	"github.com/pkg/errors"
	gouuid "github.com/satori/go.uuid"
	v1 "k8s.io/api/core/v1"
	v1beta1ext "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
)

// Plugin is a plugin driver that dispatches containers to each node,
// expecting each pod to report to the master.
type Plugin struct {
	Name       string
	PodSpec    *v1.PodSpec          `json:"spec"`
	Config     *plugin.WorkerConfig `json:"config"`
	Namespace  string
	UUID       gouuid.UUID
	ResultType string

	cleanedUp bool
}

// Ensure DaemonSetPlugin implements plugin.Interface
var _ plugin.Interface = &Plugin{}

// NewPlugin creates a new DaemonSet plugin from the given Plugin Definition
// and sonobuoy master address
func NewPlugin(namespace string, dfn plugin.Definition, cfg *plugin.WorkerConfig) *Plugin {
	return &Plugin{
		Name:       dfn.Name,
		UUID:       gouuid.NewV4(),
		ResultType: dfn.ResultType,
		PodSpec:    &dfn.PodSpec,
		Namespace:  namespace,
		Config:     cfg,
	}
}

func (p *Plugin) configMapName() string {
	return "sonobuoy-" + strings.Replace(p.Name, "_", "-", -1) + "-config-" + p.GetSessionID()
}

func (p *Plugin) daemonSetName() string {
	return "sonobuoy-" + strings.Replace(p.Name, "_", "-", -1) + "-daemonset-" + p.GetSessionID()
}

// ExpectedResults returns the list of results expected for this daemonset
func (p *Plugin) ExpectedResults(nodes []v1.Node) []plugin.ExpectedResult {
	ret := make([]plugin.ExpectedResult, 0, len(nodes))

	for _, node := range nodes {
		ret = append(ret, plugin.ExpectedResult{
			NodeName:   node.Name,
			ResultType: p.ResultType,
		})
	}

	return ret
}

// GetResultType returns the ResultType for this plugin (to adhere to plugin.Interface)
func (p *Plugin) GetResultType() string {
	return p.ResultType
}

// Run dispatches worker pods according to the DaemonSet's configuration.
func (p *Plugin) Run(kubeclient kubernetes.Interface) error {
	var err error
	configMap, err := p.buildConfigMap()
	if err != nil {
		return err
	}
	daemonSet, err := p.buildDaemonSet()
	if err != nil {
		return err
	}

	// Submit them to the API server, capturing the results
	if _, err = kubeclient.CoreV1().ConfigMaps(p.Namespace).Create(configMap); err != nil {
		return errors.Wrapf(err, "could not create ConfigMap for daemonset plugin %v", p.GetName())
	}
	if _, err = kubeclient.ExtensionsV1beta1().DaemonSets(p.Namespace).Create(daemonSet); err != nil {
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
	err := kubeclient.ExtensionsV1beta1().DaemonSets(p.Namespace).DeleteCollection(
		&deleteOptions,
		listOptions,
	)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not delete DaemonSet %v for daemonset plugin %v", p.daemonSetName(), p.GetName()))
	}

	// Delete the ConfigMap created by this plugin
	err = kubeclient.CoreV1().ConfigMaps(p.Namespace).DeleteCollection(
		&deleteOptions,
		listOptions,
	)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not delete ConfigMap %v for daemonset plugin %v", p.configMapName(), p.GetName()))
	}
}

func (p *Plugin) listOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
}

// findDaemonSet gets the daemonset that we created, using a kubernetes label search
func (p *Plugin) findDaemonSet(kubeclient kubernetes.Interface) (*v1beta1ext.DaemonSet, error) {
	dsets, err := kubeclient.ExtensionsV1beta1().DaemonSets(p.Namespace).List(p.listOptions())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(dsets.Items) != 1 {
		return nil, errors.Errorf("expected plugin %v to create 1 daemonset, found %v", p.Name, len(dsets.Items))
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
						p.Name,
					),
				}, node.Name)
			}
		}
	}
}

// GetSessionID returns a unique identifier for this dispatcher, used for tagging
// objects and cleaning them up later
func (p *Plugin) GetSessionID() string {
	ret := make([]byte, hex.EncodedLen(8))
	hex.Encode(ret, p.UUID.Bytes()[0:8])
	return string(ret)
}

// GetName returns the name of this DaemonSet plugin
func (p *Plugin) GetName() string {
	return p.Name
}

// GetPodSpec returns the pod spec for this DaemonSet
func (p *Plugin) GetPodSpec() *v1.PodSpec {
	return p.PodSpec
}

func (p *Plugin) buildConfigMap() (*v1.ConfigMap, error) {
	// We get to build the worker config directly from our own data structures,
	// this is where doing this natively in golang helps a lot (as opposed to
	// shelling out to kubectl)
	cfgjson, err := json.Marshal(p.Config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	cmap := &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.configMapName(),
			Labels:    utils.ApplyDefaultLabels(p, map[string]string{}),
			Namespace: p.Namespace,
		},
		Data: map[string]string{
			"worker.json": string(cfgjson),
		},
	}

	return cmap, nil
}

func (p *Plugin) buildDaemonSet() (*v1beta1ext.DaemonSet, error) {
	// Fix up the pod spec to use this session's config map
	for _, vol := range p.PodSpec.Volumes {
		if vol.ConfigMap != nil && vol.ConfigMap.Name == "__SONOBUOY_CONFIGMAP__" {
			vol.ConfigMap.Name = p.configMapName()
		}
	}

	ds := &v1beta1ext.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.daemonSetName(),
			Labels:    utils.ApplyDefaultLabels(p, map[string]string{}),
			Namespace: p.Namespace,
		},
		Spec: v1beta1ext.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"sonobuoy-run": p.GetSessionID(),
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: utils.ApplyDefaultLabels(p, map[string]string{}),
				},
				Spec: *p.PodSpec,
			},
		},
	}

	return ds, nil
}
