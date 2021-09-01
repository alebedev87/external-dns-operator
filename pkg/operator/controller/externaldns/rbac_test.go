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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	operatorv1alpha1 "github.com/openshift/external-dns-operator/api/v1alpha1"
	controller "github.com/openshift/external-dns-operator/pkg/operator/controller"
)

const (
	testNamespace = "testns"
)

var (
	testExternalDNS = &operatorv1alpha1.ExternalDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testexternaldns",
		},
	}
	trueVar = true
)

func TestEnsureExternalDNSClusterRole(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []runtime.Object
		expectedExist   bool
		expectedRole    rbacv1.ClusterRole
		errExpected     bool
	}{
		{
			name:            "Does not exist",
			existingObjects: []runtime.Object{},
			expectedExist:   true,
			expectedRole: rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSBaseName,
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
			},
		},
		{
			name: "Exists and as expected",
			existingObjects: []runtime.Object{
				&rbacv1.ClusterRole{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: controller.ExternalDNSBaseName,
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
				},
			},
			expectedExist: true,
			expectedRole: rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSBaseName,
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
			},
		},
		{
			name: "Exists and needs to be updated",
			existingObjects: []runtime.Object{
				&rbacv1.ClusterRole{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: controller.ExternalDNSBaseName,
					},
					Rules: []rbacv1.PolicyRule{
						{
							APIGroups: []string{"extensions", "networking.k8s.io"},
							Resources: []string{"ingresses"},
							Verbs:     []string{"get", "list", "watch"},
						},
						{
							APIGroups: []string{""},
							Resources: []string{"endpoints", "services", "pods"},
							Verbs:     []string{"get", "list", "watch"},
						},
					},
				},
			},
			expectedExist: true,
			expectedRole: rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSBaseName,
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
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl := fake.NewFakeClient(tc.existingObjects...)
			r := &reconciler{
				client: cl,
				scheme: testScheme,
				log:    zap.New(zap.UseDevMode(true)),
			}
			gotExist, gotRole, err := r.ensureExternalDNSClusterRole(context.TODO())
			if err != nil {
				if !tc.errExpected {
					t.Fatalf("unexpected error received: %v", err)
				}
				return
			}
			if tc.errExpected {
				t.Fatalf("Error expected but wasn't received")
			}

			if gotExist != tc.expectedExist {
				t.Errorf("expected cluster roles's exist to be %t, got %t", tc.expectedExist, gotExist)
			}
			diffOpts := cmpopts.IgnoreFields(rbacv1.ClusterRole{}, "ResourceVersion")
			if diff := cmp.Diff(tc.expectedRole, *gotRole, diffOpts); diff != "" {
				t.Errorf("unexpected cluster role (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExternalDNSRoleRulesChanged(t *testing.T) {
	testCases := []struct {
		name          string
		inputCurrent  []rbacv1.PolicyRule
		inputExpected []rbacv1.PolicyRule
		expectChanged bool
	}{
		{
			name: "Same",
			inputCurrent: []rbacv1.PolicyRule{
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
			inputExpected: []rbacv1.PolicyRule{
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
			expectChanged: false,
		},
		{
			name: "All reordered",
			inputCurrent: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"extensions", "networking.k8s.io"},
					Resources: []string{"ingresses"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"extensions"},
					Resources: []string{"ingresses"},
					Verbs:     []string{"get", "watch", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "nodes", "services", "endpoints"},
					Verbs:     []string{"list", "get", "watch"},
				},
			},
			inputExpected: []rbacv1.PolicyRule{
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
			expectChanged: false,
		},
		{
			name: "Changed",
			inputCurrent: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"extensions", "networking.k8s.io"},
					Resources: []string{"ingresses"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"extensions"},
					Resources: []string{"ingresses"},
					Verbs:     []string{"get", "watch", "list"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"pods", "nodes", "services", "endpoints"},
					Verbs:     []string{"list", "get", "watch", "create"},
				},
			},
			inputExpected: []rbacv1.PolicyRule{
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
			expectChanged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotChanged, _ := externalDNSRoleRulesChanged(tc.inputCurrent, tc.inputExpected)
			if gotChanged != tc.expectChanged {
				t.Errorf("expected that the role rules changed %t, got %t", tc.expectChanged, gotChanged)
			}
		})
	}
}

func TestEnsureExternalDNSClusterRoleBinding(t *testing.T) {
	testCases := []struct {
		name                string
		existingObjects     []runtime.Object
		expectedExist       bool
		expectedRoleBinding rbacv1.ClusterRoleBinding
		errExpected         bool
	}{
		{
			name:            "Does not exist",
			existingObjects: []runtime.Object{},
			expectedExist:   true,
			expectedRoleBinding: rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         operatorv1alpha1.GroupVersion.String(),
							Kind:               "ExternalDNS",
							Name:               testExternalDNS.Name,
							Controller:         &trueVar,
							BlockOwnerDeletion: &trueVar,
						},
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
		},
		{
			name: "Exists and as expected",
			existingObjects: []runtime.Object{
				&rbacv1.ClusterRoleBinding{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: controller.ExternalDNSResourceName(testExternalDNS),
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         operatorv1alpha1.GroupVersion.String(),
								Kind:               "ExternalDNS",
								Name:               testExternalDNS.Name,
								Controller:         &trueVar,
								BlockOwnerDeletion: &trueVar,
							},
						},
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     controller.ExternalDNSBaseName,
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      controller.ExternalDNSResourceName(testExternalDNS),
							Namespace: testNamespace,
						},
					},
				},
			},
			expectedExist: true,
			expectedRoleBinding: rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion:         operatorv1alpha1.GroupVersion.String(),
							Kind:               "ExternalDNS",
							Name:               testExternalDNS.Name,
							Controller:         &trueVar,
							BlockOwnerDeletion: &trueVar,
						},
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
		},
		{
			name: "Exists and needs to be updated",
			existingObjects: []runtime.Object{
				&rbacv1.ClusterRoleBinding{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRoleBinding",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: controller.ExternalDNSResourceName(testExternalDNS),
					},
					RoleRef: rbacv1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "otherrole",
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:      "ServiceAccount",
							Name:      "othersa",
							Namespace: "otherns",
						},
					},
				},
			},
			expectedExist: true,
			expectedRoleBinding: rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: "rbac.authorization.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl := fake.NewFakeClient(tc.existingObjects...)
			r := &reconciler{
				client: cl,
				scheme: testScheme,
				log:    zap.New(zap.UseDevMode(true)),
			}
			gotExist, gotRoleBinding, err := r.ensureExternalDNSClusterRoleBinding(context.TODO(), testNamespace, testExternalDNS)
			if err != nil {
				if !tc.errExpected {
					t.Fatalf("unexpected error received: %v", err)
				}
				return
			}
			if tc.errExpected {
				t.Fatalf("Error expected but wasn't received")
			}
			if gotExist != tc.expectedExist {
				t.Errorf("expected cluster roles binding's exist to be %t, got %t", tc.expectedExist, gotExist)
			}
			diffOpts := cmpopts.IgnoreFields(rbacv1.ClusterRoleBinding{}, "ResourceVersion")
			if diff := cmp.Diff(tc.expectedRoleBinding, *gotRoleBinding, diffOpts); diff != "" {
				t.Errorf("unexpected cluster role binding (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExternalDNSRoleBindingChanged(t *testing.T) {
	testCases := []struct {
		name          string
		inputCurrent  *rbacv1.ClusterRoleBinding
		inputExpected *rbacv1.ClusterRoleBinding
		expectChanged bool
	}{
		{
			name: "Same",
			inputCurrent: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
			inputExpected: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
			expectChanged: false,
		},
		{
			name: "Role changed",
			inputCurrent: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "otherrole",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
			inputExpected: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "Subject name changed",
			inputCurrent: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "othersa",
						Namespace: testNamespace,
					},
				},
			},
			inputExpected: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: testNamespace,
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "Subject namespace changed",
			inputCurrent: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "othersa",
						Namespace: testNamespace,
					},
				},
			},
			inputExpected: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: "otherns",
					},
				},
			},
			expectChanged: true,
		},
		{
			name: "All fields changed",
			inputCurrent: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "otherrole",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "othersa",
						Namespace: "otherns",
					},
				},
			},
			inputExpected: &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: controller.ExternalDNSResourceName(testExternalDNS),
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     controller.ExternalDNSBaseName,
				},
				Subjects: []rbacv1.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      controller.ExternalDNSResourceName(testExternalDNS),
						Namespace: namespace,
					},
				},
			},
			expectChanged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updated := tc.inputCurrent.DeepCopy()
			gotChanged, _ := externalDNSRoleBindingChanged(tc.inputCurrent, tc.inputExpected, updated)
			if gotChanged != tc.expectChanged {
				t.Errorf("expected that the role binding changed %t, got %t", tc.expectChanged, gotChanged)
			}
		})
	}
}
