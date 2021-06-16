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

package job

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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
)

// Plugin is a plugin driver that dispatches a single pod to the given
// kubernetes cluster.
type Plugin struct {
	driver.Base
}

// Ensure Plugin implements plugin.Interface
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
			CleanedUp:         false, // be explicit
		},
	}
}

// ExpectedResults returns the list of results expected for this plugin. Since
// a Job only launches one pod, only one result type is expected.
func (p *Plugin) ExpectedResults(nodes []v1.Node) []plugin.ExpectedResult {
	return []plugin.ExpectedResult{
		{ResultType: p.GetName(), NodeName: plugin.GlobalResult},
	}
}

func (p *Plugin) createPodDefinition(hostname string, cert *tls.Certificate, ownerPod *v1.Pod, progressPort string) v1.Pod {
	pod := v1.Pod{}
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

	pod.ObjectMeta = metav1.ObjectMeta{
		Name:        fmt.Sprintf("sonobuoy-%s-job-%s", p.GetName(), p.SessionID),
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

	var podSpec v1.PodSpec
	if p.Definition.PodSpec != nil {
		podSpec = p.Definition.PodSpec.PodSpec
	} else {
		podSpec = driver.DefaultPodSpec(p.GetDriver())
	}

	podSpec.Containers = append(podSpec.Containers,
		p.Definition.Spec.Container,
		p.CreateWorkerContainerDefintion(hostname, cert, []string{"/sonobuoy"}, []string{"worker", "global", "-v", "5", "--logtostderr"}, progressPort),
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

	pod.Spec = podSpec
	return pod
}

// Run dispatches worker pods according to the Job's configuration.
func (p *Plugin) Run(kubeclient kubernetes.Interface, hostname string, cert *tls.Certificate, ownerPod *v1.Pod, progressPort string) error {
	job := p.createPodDefinition(fmt.Sprintf("https://%s", hostname), cert, ownerPod, progressPort)

	secret, err := p.MakeTLSSecret(cert, ownerPod)
	if err != nil {
		return errors.Wrapf(err, "couldn't make secret for Job plugin %v", p.GetName())
	}

	if _, err := kubeclient.CoreV1().Secrets(p.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "couldn't create TLS secret for job plugin %v", p.GetName())
	}

	if _, err := kubeclient.CoreV1().Pods(p.Namespace).Create(context.TODO(), &job, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "could not create Job resource for Job plugin %v", p.GetName())
	}

	return nil
}

// Monitor adheres to plugin.Interface by ensuring the pod created by the job
// doesn't have any unrecoverable failures. It closes the results channel when
// it is done.
func (p *Plugin) Monitor(ctx context.Context, kubeclient kubernetes.Interface, _ []v1.Node, resultsCh chan<- *plugin.Result) {
	defer close(resultsCh)
	for {
		// Sleep between each poll, which should give the Job
		// enough time to create a Pod.
		// TODO: maybe use a watcher instead of polling.
		select {
		case <-ctx.Done():
			switch {
			case ctx.Err() == context.DeadlineExceeded:
				logrus.Errorf("Timeout waiting for plugin %v. Try checking the pod logs and other data in the results tarball for more information.", p.GetName())
				resultsCh <- utils.MakeErrorResult(
					p.GetName(),
					map[string]interface{}{"error": plugin.TimeoutErrMsg},
					plugin.GlobalResult,
				)
			case ctx.Err() == context.Canceled:
				// Do nothing, just stop.
			case ctx.Err() != nil:
				logrus.Errorf("Error seen while monitoring plugin %v: %v", p.GetName(), ctx.Err().Error())
				resultsCh <- utils.MakeErrorResult(
					p.GetName(),
					map[string]interface{}{"error": ctx.Err().Error()},
					plugin.GlobalResult,
				)
			}
			return
		case <-sonotime.After(pollingInterval):
		}

		done, errResult := p.monitorOnce(kubeclient, nil)
		if errResult != nil {
			resultsCh <- errResult
		}
		if done {
			return
		}
	}
}

func (p *Plugin) monitorOnce(kubeclient kubernetes.Interface, _ []v1.Node) (done bool, errResult *plugin.Result) {
	// If we've cleaned up after ourselves, stop monitoring
	if p.CleanedUp {
		return true, nil
	}

	// Make sure there's a pod; dont fail the pod if there are issues querying the API server.
	pod, err := p.findPod(kubeclient)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "could not find pod created by plugin %v, will retry", p.GetName()))
		return false, nil
	}

	// Make sure the pod isn't failing
	if isFailing, reason := utils.IsPodFailing(pod); isFailing {
		return true, utils.MakeErrorResult(p.GetName(), map[string]interface{}{
			"error": reason,
			"pod":   pod,
		}, plugin.GlobalResult)
	}

	return false, nil
}

// Cleanup cleans up the k8s Job and ConfigMap created by this plugin instance
func (p *Plugin) Cleanup(kubeclient kubernetes.Interface) {
	p.CleanedUp = true
	gracePeriod := int64(plugin.GracefulShutdownPeriod)
	deletionPolicy := metav1.DeletePropagationBackground

	listOptions := metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
		PropagationPolicy:  &deletionPolicy,
	}

	// Delete the Pod created by the job manually (just deleting the Job
	// doesn't kill the pod, it still lets it finish.)
	// TODO: for now we're not actually creating a Job at all, just a
	// single Pod, to get the restart semantics we want. But later if we
	// want to make this a real Job, we still need to delete pods manually
	// after deleting the job.
	err := kubeclient.CoreV1().Pods(p.Namespace).DeleteCollection(
		context.TODO(),
		deleteOptions,
		listOptions,
	)
	if err != nil {
		errlog.LogError(errors.Wrapf(err, "error deleting pods for Job-%v", p.GetSessionID()))
	}
}

func (p *Plugin) listOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: "sonobuoy-run=" + p.GetSessionID(),
	}
}

// findPod finds the pod created by this plugin, using a kubernetes label
// search.  If no pod is found, or if multiple pods are found, returns an
// error.
func (p *Plugin) findPod(kubeclient kubernetes.Interface) (*v1.Pod, error) {
	pods, err := kubeclient.CoreV1().Pods(p.Namespace).List(context.TODO(), p.listOptions())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(pods.Items) != 1 {
		return nil, errors.Errorf("no pods were created by plugin %v", p.GetName())
	}

	return &pods.Items[0], nil
}
