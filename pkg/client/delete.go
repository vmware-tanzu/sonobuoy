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
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	clusterRoleFieldName      = "component"
	clusterRoleFieldValue     = "sonobuoy"
	clusterRoleFieldNamespace = "namespace"
	spinnerMode               = "Spinner"
	progressMode              = "Progress"
	pollFreq                  = 5 * time.Second
)

// ConditionFuncWithProgress is like wait.ConditionFunc but the extra string allows us
// to capture status information.
type ConditionFuncWithProgress func() (string, bool, error)

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

	conditions := []ConditionFuncWithProgress{}
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
		lastProgress := ""
		allConditions := func() (bool, error) {
			for _, condition := range conditions {
				progress, done, err := condition()
				if cfg.WaitOutput == progressMode {
					if lastProgress == progress {
						fmt.Println("...")
					} else {
						fmt.Println()
						fmt.Println(progress)
						lastProgress = progress
					}
				}

				if !done || err != nil {
					return done, err
				}
			}
			return true, nil
		}

		switch cfg.WaitOutput {
		case spinnerMode:
			var s = getSpinnerInstance()
			s.Start()
			defer s.Stop()
		}
		if err := wait.Poll(pollFreq, cfg.Wait, allConditions); err != nil {
			return errors.Wrap(err, "waiting for delete conditions to be met")
		}
	}

	return nil
}

func cleanupNamespace(namespace string, client kubernetes.Interface) (ConditionFuncWithProgress, error) {
	// Delete the namespace
	log := logrus.WithFields(logrus.Fields{
		"kind":      "namespace",
		"namespace": namespace,
	})

	nsDeletedCondition := func() (string, bool, error) {
		ns, err := client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
		if kubeerror.IsNotFound(err) {
			return fmt.Sprintf("Namespace %q has been deleted", namespace), true, nil
		}
		if err != nil {
			status := fmt.Sprintf("Error encountered while trying to delete namespace %q: %v", ns.Name, err)
			return status, false, err
		}
		if ns != nil {
			status := fmt.Sprintf("Namespace %q has status %+v", ns.Name, ns.Status)
			return status, false, err
		}
		return "Requested namespace returned nil but was not recognized as having been deleted. Report this as a bug.", false, err
	}

	err := client.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err := logDelete(log, err); err != nil {
		return nsDeletedCondition, errors.Wrap(err, "couldn't delete namespace")
	}

	return nsDeletedCondition, nil
}

func deleteRBAC(client kubernetes.Interface, cfg *DeleteConfig) (ConditionFuncWithProgress, error) {
	// ClusterRole and ClusterRoleBindings aren't namespaced, so delete them separately.
	selector := metav1.AddLabelToSelector(
		&metav1.LabelSelector{},
		clusterRoleFieldName,
		clusterRoleFieldValue,
	)

	// Just delete the one for the target namespace unless using DeleteAll.
	if cfg != nil && !cfg.DeleteAll {
		selector = metav1.AddLabelToSelector(
			selector,
			clusterRoleFieldNamespace,
			cfg.Namespace,
		)
	}

	deleteOpts := metav1.DeleteOptions{}
	listOpts := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(selector),
	}

	rbacDeleteCondition := func() (string, bool, error) {
		clusterBindingList, err := client.RbacV1().ClusterRoleBindings().List(context.TODO(), listOpts)
		if err != nil {
			return fmt.Sprintf("Error encountered when checking for ClusterRoleBindings: %v", err), false, err
		}
		if len(clusterBindingList.Items) > 0 {
			names := []string{}
			for _, i := range clusterBindingList.Items {
				names = append(names, i.Name)
			}
			return fmt.Sprintf("Still found %v ClusterRoleBindings to delete: %v", len(names), names), false, nil
		}

		clusterRoleList, err := client.RbacV1().ClusterRoles().List(context.TODO(), listOpts)
		if err != nil {
			return fmt.Sprintf("Error encountered when checking for ClusterRoleBindings: %v", err), false, err
		}
		if len(clusterRoleList.Items) > 0 {
			names := []string{}
			for _, i := range clusterRoleList.Items {
				names = append(names, i.Name)
			}
			return fmt.Sprintf("Still found %v ClusterRoles to delete: %v", len(names), names), false, nil
		}

		// Roles and role bindings (not cluster-wide) will be deleted as the namespace is deleted.

		return "Deleted all ClusterRoles and ClusterRoleBindings.", true, nil
	}

	err := client.RbacV1().ClusterRoleBindings().DeleteCollection(context.TODO(), deleteOpts, listOpts)
	if err := logDelete(logrus.WithField("kind", "clusterrolebindings"), err); err != nil {
		return rbacDeleteCondition, errors.Wrap(err, "failed to delete cluster role binding")
	}

	// ClusterRole and ClusterRole bindings aren't namespaced, so delete them manually
	err = client.RbacV1().ClusterRoles().DeleteCollection(context.TODO(), deleteOpts, listOpts)
	if err := logDelete(logrus.WithField("kind", "clusterroles"), err); err != nil {
		return rbacDeleteCondition, errors.Wrap(err, "failed to delete cluster role")
	}

	return rbacDeleteCondition, nil
}

func isE2ENamespace(ns v1.Namespace) bool {
	// E2E namespaces are identified by looking for the "e2e-framework" and "e2e-run" labels.
	_, hasE2EFrameworkLabel := ns.Labels["e2e-framework"]
	_, hasE2ERunLabel := ns.Labels["e2e-run"]

	return hasE2EFrameworkLabel && hasE2ERunLabel
}

func cleanupE2E(client kubernetes.Interface) (ConditionFuncWithProgress, error) {
	// Delete any dangling E2E namespaces
	// There currently isn't a way to associate namespaces with a particular test run so this
	// will delete namespaces associated with any e2e test run by checking the namespace's labels.
	e2eNamespaceCondition := func() (string, bool, error) {
		namespaces, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "Error encountered when checking namespaces for deletion.", false, errors.Wrap(err, "failed to list namespaces")
		}
		names := []string{}
		for _, namespace := range namespaces.Items {
			if isE2ENamespace(namespace) {
				names = append(names, namespace.Name)
			}
		}
		if len(names) > 0 {
			return fmt.Sprintf("Found %v namespaces that still need to be deleted: %v", len(names), names), false, nil
		}
		return "All E2E namespaces deleted", true, nil
	}

	namespaces, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return e2eNamespaceCondition, errors.Wrap(err, "failed to list namespaces")
	}

	for _, namespace := range namespaces.Items {
		if isE2ENamespace(namespace) {
			log := logrus.WithFields(logrus.Fields{
				"kind":      "namespace",
				"namespace": namespace.Name,
			})
			err := client.CoreV1().Namespaces().Delete(context.TODO(), namespace.Name, metav1.DeleteOptions{})
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
		log.Info("delete request issued")
	case kubeerror.IsNotFound(err):
		log.Info("already deleted")
	case kubeerror.IsConflict(err):
		log.WithError(err).Info("delete already in progress")
	case err != nil:
		return err
	}
	return nil
}
