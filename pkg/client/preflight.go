/*
Copyright 2018 the Sonobuoy project contributors

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
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apicorev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

const (
	kubeSystemNamespace = "kube-system"
	kubeDNSLabelKey     = "k8s-app"
	kubeDNSLabelValue   = "kube-dns"
	coreDNSLabelValue   = "coredns"
)

var (
	minimumKubeVersion = version.Must(version.NewVersion(buildinfo.MinimumKubeVersion))
	maximumKubeVersion = version.Must(version.NewVersion(buildinfo.MaximumKubeVersion))

	expectedDNSLabels = []string{
		kubeDNSLabelValue,
		coreDNSLabelValue,
	}

	preflightChecks = []func(kubernetes.Interface, *PreflightConfig) error{
		preflightDNSCheck,
		preflightVersionCheck,
		preflightExistingNamespace,
	}
)

type listFunc func(metav1.ListOptions) (*apicorev1.PodList, error)
type nsGetFunc func(string, metav1.GetOptions) (*apicorev1.Namespace, error)

// PreflightChecks runs all preflight checks in order, returning the first error encountered.
func (c *SonobuoyClient) PreflightChecks(cfg *PreflightConfig) []error {
	if cfg == nil {
		return []error{errors.New("nil PreflightConfig provided")}
	}

	if err := cfg.Validate(); err != nil {
		return []error{errors.Wrap(err, "config validation failed")}
	}

	client, err := c.Client()
	if err != nil {
		return []error{err}
	}

	errors := []error{}
	for _, check := range preflightChecks {
		if err := check(client, cfg); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func preflightDNSCheck(client kubernetes.Interface, cfg *PreflightConfig) error {
	return dnsCheck(
		client.CoreV1().Pods(kubeSystemNamespace).List,
		expectedDNSLabels...,
	)
}

func dnsCheck(listPods listFunc, dnsLabels ...string) error {
	if len(dnsLabels) == 0 {
		return nil
	}

	var nPods = 0
	for _, labelValue := range dnsLabels {
		selector := metav1.AddLabelToSelector(&metav1.LabelSelector{}, kubeDNSLabelKey, labelValue)

		obj, err := listPods(
			metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)},
		)
		if err != nil {
			return errors.Wrap(err, "could not retrieve list of pods")
		}

		if len(obj.Items) > 0 {
			nPods += len(obj.Items)
			break
		}
	}

	if nPods == 0 {
		return fmt.Errorf("no dns pods found with the labels [%s] in namespace kube-system", strings.Join(dnsLabels, ", "))
	}

	return nil
}

func preflightVersionCheck(client kubernetes.Interface, cfg *PreflightConfig) error {
	return versionCheck(
		client.Discovery(),
		minimumKubeVersion,
		maximumKubeVersion,
	)
}

func versionCheck(versionClient discovery.ServerVersionInterface, min, max *version.Version) error {
	versionInfo, err := versionClient.ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve server version")
	}
	serverVersion, err := version.NewVersion(versionInfo.String())
	if err != nil {
		return errors.Wrap(err, "couldn't parse version string")
	}

	if serverVersion.LessThan(min) {
		return fmt.Errorf("minimum supported Kubernetes version is %s, but the server version is %s", min.String(), versionInfo.String())
	}

	if serverVersion.GreaterThan(max) {
		logrus.Warningf("The maximum supported Kubernetes version is %s, but the server version is %s. Sonobuoy will continue but unexpected results may occur.", max.String(), versionInfo.String())
	}

	return nil
}

func preflightExistingNamespace(client kubernetes.Interface, cfg *PreflightConfig) error {
	return nsCheck(
		client.CoreV1().Namespaces().Get,
		cfg.Namespace,
	)
}

func nsCheck(getter nsGetFunc, ns string) error {
	_, err := getter(ns, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		return nil
	case err != nil:
		return errors.Wrap(err, "error checking for namespace")
	case err == nil:
		return errors.New("namespace already exists")
	}
	return nil
}
