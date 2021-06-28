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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
)

// GetStatus returns the aggregation status that is set as an annotation on the aggregator pod.
// Returns an error if unable to find the namespace, pod, or annotation. Use GetStatusPod to
// also return the pod itself so that you can better understand the state of the system.
func (c *SonobuoyClient) GetStatus(cfg *StatusConfig) (*aggregation.Status, error) {
	s, _, err := c.GetStatusPod(cfg)
	return s, err
}

// GetStatus returns the aggregation status that is set as an annotation on the aggregator pod.
// Returns an error if unable to find the namespace, pod, or annotation. Also returns the aggregator
// pod itself so that you can check its exact status or other annotations.
func (c *SonobuoyClient) GetStatusPod(cfg *StatusConfig) (*aggregation.Status, *corev1.Pod, error) {
	if cfg == nil {
		return nil, nil, errors.New("nil StatusConfig provided")
	}

	if err := cfg.Validate(); err != nil {
		return nil, nil, errors.Wrap(err, "config validation failed")
	}

	client, err := c.Client()
	if err != nil {
		return nil, nil, err
	}

	return aggregation.GetStatus(client, cfg.Namespace)
}
