/*
Copyright 2021.

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

package externaldnscontroller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	operatorv1alpha1 "github.com/openshift/external-dns-operator/api/v1alpha1"
	controller "github.com/openshift/external-dns-operator/pkg/operator/controller"
)

const (
	clusterRoleName = "external-dns"
)

// ensureExternalDNSClusterRole ensures that the externalDNS cluster role exists.
// Returns a boolean if the cluster role exists, and an error when relevant.
func (r *reconciler) ensureExternalDNSClusterRole(ctx context.Context) (bool, *rbacv1.ClusterRole, error) {
	desired := desiredExternalDNSClusterRole()

	exists, current, err := r.currentExternalDNSClusterRole(ctx)
	if err != nil {
		return false, nil, err
	}

	if !exists {
		if err := r.createExternalDNSClusterRole(ctx, desired); err != nil {
			return false, nil, err
		}
		// getting API in case a third party updated it after the creation
		return r.currentExternalDNSClusterRole(ctx)
	}

	// update the cluster role
	if _, err := r.updateExternalDNSClusterRole(ctx, current, desired); err != nil {
		return true, current, err
	}

	// getting from API in case a third party updated it after the update
	return r.currentExternalDNSClusterRole(ctx)
}

// desiredExternalDNSClusterRole returns the desired cluster role definition for externalDNS
func desiredExternalDNSClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"extensions", "networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"endpoints", "services", "pods", "nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
}

// currentExternalDNSClusterRole returns true if cluster role exists
func (r *reconciler) currentExternalDNSClusterRole(ctx context.Context) (bool, *rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: clusterRoleName}, cr); err != nil {
		if errors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, cr, nil
}

// createExternalDNSClusterRole creates the given deployment
func (r *reconciler) createExternalDNSClusterRole(ctx context.Context, desired *rbacv1.ClusterRole) error {
	if err := r.client.Create(ctx, desired); err != nil {
		return fmt.Errorf("failed to create externalDNS cluster role %s: %v", desired.Name, err)
	}
	r.log.Info("created externalDNS cluster role", "name", desired.Name)
	return nil
}

// updateExternalDNSClusterRole updates the cluster role with the desired state if the rules differ
func (r *reconciler) updateExternalDNSClusterRole(ctx context.Context, current, desired *rbacv1.ClusterRole) (bool, error) {
	if reflect.DeepEqual(current.Rules, desired.Rules) {
		return false, nil
	}
	updated := current.DeepCopy()
	updated.Rules = desired.Rules
	// Diff before updating because the client may mutate the object.
	diff := cmp.Diff(current, updated, cmpopts.EquateEmpty())
	if err := r.client.Update(ctx, updated); err != nil {
		if errors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}
	r.log.Info("updated externalDNS cluster role", "name", updated.Name, "diff", diff)
	return true, nil
}

// ensureExternalDNSClusterRoleBinding ensures that externalDNS cluster role binding exists.
func (r *reconciler) ensureExternalDNSClusterRoleBinding(ctx context.Context, namespace string, externalDNS *operatorv1alpha1.ExternalDNS) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: controller.ExternalDNSResourceName(externalDNS),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      controller.ExternalDNSResourceName(externalDNS),
				Namespace: namespace,
			},
		},
	}
	if err := r.client.Get(ctx, types.NamespacedName{Name: crb.Name}, crb); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("failed to get externalDNS cluster role binding %s: %v", crb.Name, err)
		}
		if err := r.client.Create(ctx, crb); err != nil {
			return fmt.Errorf("failed to create externalDNS cluster role binding %s: %v", crb.Name, err)
		}
		r.log.Info("created externalDNS cluster role binding", "name", crb.Name)
	}
	return nil
}
