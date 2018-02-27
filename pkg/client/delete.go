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
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	clusterRoleFieldName  = "component"
	clusterRoleFieldValue = "sonobuoy"

	e2eNamespacePrefix = "e2e-"
)

func (c *SonobuoyClient) Delete(cfg *DeleteConfig, client kubernetes.Interface) error {
	// Delete the namespace
	if err := client.CoreV1().Namespaces().Delete(cfg.Namespace, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "failed to delete namespace")
	}
	logrus.WithField("namespace", cfg.Namespace).Info("deleted namespace")

	if cfg.EnableRBAC {
		// ClusterRole and ClusterRoleBindings aren't namespaced, so delete them seperately
		selector := metav1.AddLabelToSelector(
			&metav1.LabelSelector{},
			clusterRoleFieldName,
			clusterRoleFieldValue,
		)

		deleteOpts := &metav1.DeleteOptions{}
		listOpts := metav1.ListOptions{
			LabelSelector: metav1.FormatLabelSelector(selector),
		}

		if err := client.RbacV1().ClusterRoleBindings().DeleteCollection(deleteOpts, listOpts); err != nil {
			return errors.Wrap(err, "failed to delete cluster role binding")
		}

		// ClusterRole and ClusterRole bindings aren't namespaced, so delete them manually
		if err := client.RbacV1().ClusterRoles().DeleteCollection(deleteOpts, listOpts); err != nil {
			return errors.Wrap(err, "failed to delete cluster role")
		}
		logrus.Info("deleted clusterrolebindings and clusterroles")
	}

	// Delete any dangling E2E namespaces
	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list namespaces")
	}

	for _, namespace := range namespaces.Items {
		if strings.HasPrefix(namespace.Name, e2eNamespacePrefix) {
			if err := client.CoreV1().Namespaces().Delete(namespace.Name, &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete e2e namespace")
			}
			logrus.WithField("namespace", namespace.Name).Info("deleted E2E namespace")
		}
	}
	return nil
}
