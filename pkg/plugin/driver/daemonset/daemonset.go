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
	"context"
	"crypto/tls"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver/utils"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	sonotime "github.com/vmware-tanzu/sonobuoy/pkg/time"
)

const (
	// pollingInterval is the time between polls when monitoring the job status.
	pollingInterval = 10 * time.Second

	// defaultSleepSeconds is the time after the plugin finishes for which Sonobuoy will sleep.
	// The sleep functions as a way to prevent the daemonset from restarting the container once the
	// process completes. There is currently no way to have a "run-once daemonset". Defaults
	// to sleeping forever.
	defaultSleepSeconds = "-1"
)

// Plugin is a plugin driver that dispatches containers to each node,
// expecting each pod to report to the aggregator.
type Plugin struct {
	driver.Base
}

// Ensure DaemonSetPlugin implements plugin.Interface
var _ plugin.Interface = &Plugin{}

// NewPlugin creates a new DaemonSet plugin from the given Plugin Definition
// and sonobuoy aggregator address.
func NewPlugin(dfn manifest.Manifest, namespace, sonobuoyImage, imagePullPolicy, imagePullSecrets string, customAnnotations map[string]string) *Plugin {
	return &Plugin{
		driver.Base{
			Definition:        dfn,
			SessionID:         utils.GetSessionID(),
			Namespace:         namespace,
			SonobuoyImage:     sonobuoyImage,
			ImagePullPolicy:   imagePullPolicy,
			ImagePullSecrets:  imagePullSecrets,
			CustomAnnotations: customAnnotations,
			CleanedUp:         false,
		},
	}
}

// ExpectedResults returns the list of results expected for this daemonset.
func (p *Plugin) ExpectedResults(nodes []v1.Node) []plugin.ExpectedResult {
	nodes = p.filterByNodeSelector(nodes)
	ret := make([]plugin.ExpectedResult, 0, len(nodes))

	for _, node := range nodes {
		ret = append(ret, plugin.ExpectedResult{
			NodeName:   node.Name,
			ResultType: p.GetName(),
		})
	}

	return ret
}

// filterByNodeSelector will filter the list of nodes to just the ones matching the affinity of the plugin.
func (p *Plugin) filterByNodeSelector(nodes []v1.Node) []v1.Node {
	ps := p.Base.Definition.PodSpec
	if ps == nil ||
		ps.Affinity == nil ||
		ps.Affinity.NodeAffinity == nil ||
		ps.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return nodes
	}

	retNodes := []v1.Node{}
	nodeSelector := ps.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	for _, node := range nodes {
		if nodeMatchesNodeSelector(&node, nodeSelector) {
			retNodes = append(retNodes, node)
		}
	}
	return retNodes
}

func (p *Plugin) createDaemonSetDefinition(hostname string, cert *tls.Certificate, ownerPod *v1.Pod, progressPort string) appsv1.DaemonSet {
	ds := appsv1.DaemonSet{}
	annotations := map[string]string{
		"sonobuoy-driver": p.GetDriver(),
		"sonobuoy-plugin": p.GetName(),
	}
	for k, v := range p.CustomAnnotations {
		annotations[k] = v
	}
	labels := map[string]string{
		"component":          "sonobuoy",
		"sonobuoy-component": "plugin",
		"sonobuoy-plugin":    p.GetName(),
		"sonobuoy-run":       p.SessionID,
		"tier":               "analysis",
	}

	ds.ObjectMeta = metav1.ObjectMeta{
		Name:        fmt.Sprintf("sonobuoy-%s-daemon-set-%s", p.GetName(), p.SessionID),
		Namespace:   p.Namespace,
		Labels:      labels,
		Annotations: annotations,
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       ownerPod.GetName(),
				UID:        ownerPod.GetUID(),
			},
		},
	}

	ds.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"sonobuoy-run": p.SessionID,
		},
	}

	ds.Spec.Template.ObjectMeta.Labels = labels
	ds.Spec.Template.ObjectMeta.Annotations = p.CustomAnnotations

	var podSpec v1.PodSpec
	if p.Definition.PodSpec != nil {
		podSpec = p.Definition.PodSpec.PodSpec
	} else {
		podSpec = driver.DefaultPodSpec(p.GetDriver())
	}

	podSpec.Containers = append(podSpec.Containers,
		p.Definition.Spec.Container,
		p.CreateWorkerContainerDefintion(hostname, cert, []string{"/sonobuoy", "worker", "single-node", "-v=5", "--logtostderr", "--sleep=" + defaultSleepSeconds}, []string{}, progressPort),
	)

	if len(p.ImagePullSecrets) > 0 {
		podSpec.ImagePullSecrets = append(podSpec.ImagePullSecrets, v1.LocalObjectReference{
			Name: p.ImagePullSecrets,
		})
	}

	podSpec.Volumes = append(podSpec.Volumes, v1.Volume{
		Name: "results",
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})

	for _, v := range p.Definition.ExtraVolumes {
		podSpec.Volumes = append(podSpec.Volumes, v.Volume)
	}

	ds.Spec.Template.Spec = podSpec
	return ds
}

// Run dispatches worker pods according to the DaemonSet's configuration.
func (p *Plugin) Run(kubeclient kubernetes.Interface, hostname string, cert *tls.Certificate, ownerPod *v1.Pod, progressPort string) error {
	daemonSet := p.createDaemonSetDefinition(fmt.Sprintf("https://%s", hostname), cert, ownerPod, progressPort)

	secret, err := p.MakeTLSSecret(cert, ownerPod)
	if err != nil {
		return errors.Wrapf(err, "couldn't make secret for daemonset plugin %v", p.GetName())
	}

	if _, err := kubeclient.CoreV1().Secrets(p.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "couldn't create TLS secret for daemonset plugin %v", p.GetName())
	}

	if _, err := kubeclient.AppsV1().DaemonSets(p.Namespace).Create(context.TODO(), &daemonSet, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "could not create DaemonSet for daemonset plugin %v", p.GetName())
	}

	return nil
}

// Cleanup cleans up the k8s DaemonSet and ConfigMap created by this plugin instance.
func (p *Plugin) Cleanup(kubeclient kubernetes.Interface) {
	p.CleanedUp = true
	gracePeriod := int64(1)
	deletionPolicy := metav1.DeletePropagationBackground

	listOptions := p.listOptions()
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &deletionPolicy,
	}

	// Delete the DaemonSet created by this plugin
	// TODO(EKF): Move to v1 in 1.11
	err := kubeclient.AppsV1().DaemonSets(p.Namespace).DeleteCollection(
		context.TODO(),
		deleteOptions,
		listOptions,
	)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not delete DaemonSet-%v for daemonset plugin %v", p.GetSessionID(), p.GetName()))
	}
}

func (p *Plugin) listOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
}

// findDaemonSet gets the daemonset that we created, using a kubernetes label search.
func (p *Plugin) findDaemonSet(kubeclient kubernetes.Interface) (*appsv1.DaemonSet, error) {
	dsets, err := kubeclient.AppsV1().DaemonSets(p.Namespace).List(context.TODO(), p.listOptions())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(dsets.Items) != 1 {
		return nil, errors.Errorf("expected plugin %v to create 1 daemonset, found %v", p.GetName(), len(dsets.Items))
	}

	return &dsets.Items[0], nil
}

// Monitor adheres to plugin.Interface by ensuring the DaemonSet is correctly
// configured and that each pod is running normally.
func (p *Plugin) Monitor(ctx context.Context, kubeclient kubernetes.Interface, availableNodes []v1.Node, resultsCh chan<- *plugin.Result) {
	availableNodes = p.filterByNodeSelector(availableNodes)
	podsReported := make(map[string]bool)
	podsFound := make(map[string]bool, len(availableNodes))
	for _, node := range availableNodes {
		podsFound[node.Name] = false
		podsReported[node.Name] = false
	}

	for {
		// Sleep between each poll, which should give the DaemonSet
		// enough time to create pods
		select {
		case <-ctx.Done():
			switch {
			// This routine is monitoring the entire daemonset and has no knowledge of which
			// nodes have returned results to the aggregator. We can report the error for every node though
			// since the aggregator will throw out duplicate results.
			case ctx.Err() == context.DeadlineExceeded:
				logrus.Errorf("Timeout waiting for plugin %v. Try checking the pod logs and other data in the results tarball for more information.", p.GetName())
				errs := makeErrorResultsForNodes(
					p.GetName(),
					map[string]interface{}{"error": plugin.TimeoutErrMsg},
					availableNodes,
				)
				for _, e := range errs {
					resultsCh <- e
				}
			case ctx.Err() == context.Canceled:
				// Do nothing, just stop.
			case ctx.Err() != nil:
				logrus.Errorf("Error seen while monitoring plugin %v: %v", p.GetName(), ctx.Err().Error())
				errs := makeErrorResultsForNodes(
					p.GetName(),
					map[string]interface{}{"error": ctx.Err().Error()},
					availableNodes,
				)
				for _, e := range errs {
					resultsCh <- e
				}
			}
			return
		case <-sonotime.After(pollingInterval):
		}

		done, errResults := p.monitorOnce(kubeclient, availableNodes, podsFound, podsReported)
		for _, v := range errResults {
			resultsCh <- v
		}
		if done {
			return
		}
	}
}

// monitorOnce handles the actual logic executed in the Monitor routine which also adds polling.
// It will return a boolean, indicating monitoring should stop, along with a result if one should
// be generated. The arguments, podsFound and podsReported, are used to persist some knowledge about
// the pods between calls.
func (p *Plugin) monitorOnce(kubeclient kubernetes.Interface, availableNodes []v1.Node, podsFound, podsReported map[string]bool) (done bool, retErrs []*plugin.Result) {
	// If we've cleaned up after ourselves, stop monitoring
	if p.CleanedUp {
		return true, nil
	}

	// If we don't have a daemonset created, retry next time.  We
	// only send errors if we successfully see that an expected pod
	// is having issues.
	ds, err := p.findDaemonSet(kubeclient)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not find DaemonSet created by plugin %v, will retry", p.GetName()))
		return false, nil
	}

	// Find all the pods configured by this daemonset
	pods, err := kubeclient.CoreV1().Pods(p.Namespace).List(context.TODO(), p.listOptions())
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not find pods created by plugin %v, will retry", p.GetName()))
		// Likewise, if we can't query for pods, just retry next time.
		return false, nil
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

			retErrs = append(retErrs, utils.MakeErrorResult(p.GetName(), map[string]interface{}{
				"error": reason,
				"pod":   pod,
			}, nodeName))
		}
	}

	// The main caller, Monitor, filters the list typically before passing it to us. However,
	// the cost of the call is pretty small and this aids in testing to ensure that we do
	// not error when the plugin isn't targeting that node.
	availableNodes = p.filterByNodeSelector(availableNodes)

	// DaemonSets are a bit strange, if node taints are preventing
	// scheduling, pods won't even be created (unlike say Jobs,
	// which will create the pod and leave it in an unscheduled
	// state.)  So take any nodes we didn't see pods on, and report
	// issues scheduling them.
	for _, node := range availableNodes {
		if !podsFound[node.Name] && !podsReported[node.Name] {
			podsReported[node.Name] = true
			retErrs = append(retErrs, utils.MakeErrorResult(p.GetName(), map[string]interface{}{
				"error": fmt.Sprintf(
					"No pod was scheduled on node %v within %v. Check tolerations for plugin %v",
					node.Name,
					time.Since(ds.CreationTimestamp.Time),
					p.GetName(),
				),
			}, node.Name))
		}
	}

	return false, retErrs
}

func makeErrorResultsForNodes(resultType string, errdata map[string]interface{}, nodes []v1.Node) []*plugin.Result {
	results := []*plugin.Result{}
	for _, n := range nodes {
		results = append(results, utils.MakeErrorResult(resultType, errdata, n.Name))
	}
	return results
}

// nodeMatchesNodeSelector checks if a node's labels satisfy a node selector. It is a simplification
// of upstream logic in `k8s.io/kubernetes` which isn't intended for consumption as a module.
// If the nodeSelector has multiple expressions/terms; this method returns the union of the nodes
// satisfying the individual terms.
func nodeMatchesNodeSelector(node *v1.Node, sel *v1.NodeSelector) bool {
	for _, term := range sel.NodeSelectorTerms {
		// We only support MatchExpressions at this time.
		for _, exp := range term.MatchExpressions {
			switch exp.Operator {
			case v1.NodeSelectorOpExists:
				if _, ok := node.Labels[exp.Key]; ok {
					return true
				}
			case v1.NodeSelectorOpDoesNotExist:
				if _, ok := node.Labels[exp.Key]; !ok {
					return true
				}
			case v1.NodeSelectorOpIn:
				if val, ok := node.Labels[exp.Key]; ok && stringInList(exp.Values, val) {
					return true
				}
			case v1.NodeSelectorOpNotIn:
				if val, ok := node.Labels[exp.Key]; !ok || !stringInList(exp.Values, val) {
					return true
				}
			default:
				continue
			}
		}
	}
	return false
}

func stringInList(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
