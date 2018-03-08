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
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var preflightChecks = []func(kubernetes.Interface) error{
	preflightDNSCheck,
	preflightVersionCheck,
}

// PreflightChecks runs all preflight checks in order, returning the first error encountered.
func (c *SonobuoyClient) PreflightChecks() []error {
	client, err := c.Client()
	if err != nil {
		return []error{err}
	}

	errors := []error{}

	for _, check := range preflightChecks {
		if err := check(client); err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

const (
	kubeSystemNamespace = "kube-system"
	kubeDNSLabelKey     = "k8s-app"
	kubeDNSLabelValue   = "kube-dns"
)

func preflightDNSCheck(client kubernetes.Interface) error {
	selector := metav1.AddLabelToSelector(&metav1.LabelSelector{}, kubeDNSLabelKey, kubeDNSLabelValue)

	obj, err := client.CoreV1().Pods(kubeSystemNamespace).List(
		metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)},
	)
	if err != nil {
		return errors.Wrap(err, "could not retrieve list of pods")
	}

	if len(obj.Items) == 0 {
		return errors.New("no kube-dns tests found")
	}

	return nil
}

var (
	minimumKubeVersion = version.Must(version.NewVersion(buildinfo.MinimumKubeVersion))
	maximumKubeVersion = version.Must(version.NewVersion(buildinfo.MaximumKubeVersion))
)

func preflightVersionCheck(client kubernetes.Interface) error {
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve server version")
	}

	serverVersion, err := version.NewVersion(versionInfo.String())
	if err != nil {
		return errors.Wrap(err, "couldn't parse version string")
	}

	if serverVersion.LessThan(minimumKubeVersion) {
		return fmt.Errorf("Minimum kubernetes version is %s, got %s", minimumKubeVersion.String(), versionInfo.String())
	}

	if serverVersion.GreaterThan(maximumKubeVersion) {
		return fmt.Errorf("Maximum kubernetes version is %s, got %s", maximumKubeVersion.String(), versionInfo.String())
	}

	return nil
}
