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
	"time"

	"github.com/briandowns/spinner"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	clusterRoleFieldName      = "component"
	clusterRoleFieldNamespace = "namespace"
	clusterRoleFieldValue     = "sonobuoy"
	spinnerMode               = "Spinner"
	e2eNamespacePrefix        = "e2e-"
	pollFreq                  = 5 * time.Second
)

// Delete removes all the resources that Sonobuoy had created including
// its own namespace, cluster roles/bindings, and optionally e2e scoped
// namespaces.
func (c *SonobuoyClient) Delete(cfg *DeleteConfig) error {
	if cfg == nil {
		return errors.New("nil DeleteConfig provided")
	}

	if err := cfg.Validate(); err != nil {
		return errors.Wrap(err, "config validation failed")
	}

	client, err := c.Client()
	if err != nil {
		return err
	}

	conditions := []wait.ConditionFunc{}
	nsCondition, err := cleanupNamespace(cfg.Namespace, client)
	if err != nil {
		return err
	}
	conditions = append(conditions, nsCondition)

	if cfg.EnableRBAC {
		rbacCondition, err := deleteRBAC(client, cfg)
		if err != nil {
			return err
		}
		conditions = append(conditions, rbacCondition)
	}

	if cfg.DeleteAll {
		e2eCondition, err := cleanupE2E(client)
		if err != nil {
			return err
		}
		conditions = append(conditions, e2eCondition)
	}

	if cfg.Wait > time.Duration(0) {

		allConditions := func() (bool, error) {
			for _, condition := range conditions {
				done, err := condition()
				if !done || err != nil {
					return done, err
				}
			}
			return true, nil
		}

		if strings.Compare(cfg.WaitOutput, spinnerMode) == 0 {
			var s *spinner.Spinner
			s = getSpinnerInstance()
			s.Start()
			defer s.Stop()
		}
		if err := wait.Poll(pollFreq, cfg.Wait, allConditions); err != nil {
			return errors.Wrap(err, "waiting for delete conditions to be met")
		}
	}

	return nil
}

func cleanupNamespace(namespace string, client kubernetes.Interface) (wait.ConditionFunc, error) {
	// Delete the namespace
	log := logrus.WithFields(logrus.Fields{
		"kind":      "namespace",
		"namespace": namespace,
	})

	nsDeletedCondition := func() (bool, error) {
		_, err := client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
		if kubeerror.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}

	err := client.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
	if err := logDelete(log, err); err != nil {
		return nsDeletedCondition, errors.Wrap(err, "couldn't delete namespace")
	}

	return nsDeletedCondition, nil
}

func deleteRBAC(client kubernetes.Interface, cfg *DeleteConfig) (wait.ConditionFunc, error) {
	// ClusterRole and ClusterRoleBindings aren't namespaced, so delete them seperately
	selector := metav1.AddLabelToSelector(
		&metav1.LabelSelector{},
		clusterRoleFieldName,
		clusterRoleFieldValue,
	)
	if cfg != nil {
		selector = metav1.AddLabelToSelector(
			selector,
			clusterRoleFieldNamespace,
			cfg.Namespace,
		)
	}

	deleteOpts := &metav1.DeleteOptions{}
	listOpts := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(selector),
	}

	rbacDeleteCondition := func() (bool, error) {
		bindingList, err := client.RbacV1().ClusterRoleBindings().List(listOpts)
		if err != nil || len(bindingList.Items) > 0 {
			return false, err
		}

		roleList, err := client.RbacV1().ClusterRoles().List(listOpts)
		if err != nil || len(roleList.Items) > 0 {
			return false, err
		}

		return true, nil
	}

	err := client.RbacV1().ClusterRoleBindings().DeleteCollection(deleteOpts, listOpts)
	if err := logDelete(logrus.WithField("kind", "clusterrolebindings"), err); err != nil {
		return rbacDeleteCondition, errors.Wrap(err, "failed to delete cluster role binding")
	}

	// ClusterRole and ClusterRole bindings aren't namespaced, so delete them manually
	err = client.RbacV1().ClusterRoles().DeleteCollection(deleteOpts, listOpts)
	if err := logDelete(logrus.WithField("kind", "clusterroles"), err); err != nil {
		return rbacDeleteCondition, errors.Wrap(err, "failed to delete cluster role")
	}

	return rbacDeleteCondition, nil
}

func cleanupE2E(client kubernetes.Interface) (wait.ConditionFunc, error) {
	// Delete any dangling E2E namespaces
	e2eNamespaceCondition := func() (bool, error) {
		namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return false, errors.Wrap(err, "failed to list namespaces")
		}
		for _, namespace := range namespaces.Items {
			if strings.HasPrefix(namespace.Name, e2eNamespacePrefix) {
				return false, nil
			}
		}
		return true, nil
	}

	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return e2eNamespaceCondition, errors.Wrap(err, "failed to list namespaces")
	}

	for _, namespace := range namespaces.Items {
		if strings.HasPrefix(namespace.Name, e2eNamespacePrefix) {
			log := logrus.WithFields(logrus.Fields{
				"kind":      "namespace",
				"namespace": namespace.Name,
			})
			err := client.CoreV1().Namespaces().Delete(namespace.Name, &metav1.DeleteOptions{})
			if err := logDelete(log, err); err != nil {
				return e2eNamespaceCondition, errors.Wrap(err, "couldn't delete namespace")
			}
		}
	}

	return e2eNamespaceCondition, nil
}

func logDelete(log logrus.FieldLogger, err error) error {
	switch {
	case err == nil:
		log.Info("deleted")
	case kubeerror.IsNotFound(err):
		log.Info("already deleted")
	case kubeerror.IsConflict(err):
		log.WithError(err).Info("delete in progress")
	case err != nil:
		return err
	}
	return nil
}
