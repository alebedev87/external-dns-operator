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
	"reflect"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestEnsureExternalDNSClusterRole(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []runtime.Object
		expectedExist   bool
		expectedRole    rbacv1.ClusterRole
	}{
		{
			name:            "Does not exist",
			existingObjects: []runtime.Object{},
			expectedExist:   true,
			expectedRole: rbacv1.ClusterRole{
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
			},
		},
		{
			name: "Exist and as expected",
			existingObjects: []runtime.Object{
				&rbacv1.ClusterRole{
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
				},
			},
			expectedExist: true,
			expectedRole: rbacv1.ClusterRole{
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
			},
		},
		{
			name: "Exist and needs to be updated",
			existingObjects: []runtime.Object{
				&rbacv1.ClusterRole{
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
							Resources: []string{"endpoints", "services", "pods"},
							Verbs:     []string{"get", "list", "watch"},
						},
					},
				},
			},
			expectedExist: true,
			expectedRole: rbacv1.ClusterRole{
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
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl := fake.NewFakeClient(tc.existingObjects...)
			r := &reconciler{
				client: cl,
				log:    zap.New(zap.UseDevMode(true)),
			}
			gotExist, gotRole, _ := r.ensureExternalDNSClusterRole(context.TODO())
			if gotExist != tc.expectedExist {
				t.Errorf("expected cluster roles's exist to be %t, got %t", tc.expectedExist, gotExist)
			}
			if reflect.DeepEqual(*gotRole, tc.expectedRole) {
				t.Errorf("expected cluster roles %v, got %v", tc.expectedRole, gotRole)
			}
		})
	}
}
