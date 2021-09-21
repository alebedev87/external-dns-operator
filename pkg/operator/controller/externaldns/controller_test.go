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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/openshift/external-dns-operator/api/v1alpha1"
)

const (
	testExtDNSNamespace = "external-dns"
	testExtDNSName      = "test"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	if err := clientgoscheme.AddToScheme(testScheme); err != nil {
		panic(err)
	}
	if err := operatorv1alpha1.AddToScheme(testScheme); err != nil {
		panic(err)
	}
}

func TestReconcile(t *testing.T) {
	managedTypesList := []client.ObjectList{
		&corev1.NamespaceList{},
		&appsv1.DeploymentList{},
		&corev1.ServiceAccountList{},
		&rbacv1.ClusterRoleList{},
		&rbacv1.ClusterRoleBindingList{},
	}
	eventWaitTimeout := time.Duration(1 * time.Second)

	testCases := []struct {
		name            string
		existingObjects []runtime.Object
		inputConfig     Config
		inputRequest    ctrl.Request
		expectedResult  reconcile.Result
		expectedEvents  []testEvent
		errExpected     bool
	}{
		{
			name:            "Bootstrap",
			existingObjects: []runtime.Object{testExtDNSInstance()},
			inputConfig:     testConfig(),
			inputRequest:    testRequest(),
			expectedResult:  reconcile.Result{},
			expectedEvents: []testEvent{
				{
					eventType: watch.Added,
					objType:   "clusterrole",
					NamespacedName: types.NamespacedName{
						Name: "external-dns",
					},
				},
				{
					eventType: watch.Added,
					objType:   "deployment",
					NamespacedName: types.NamespacedName{
						Namespace: testExtDNSNamespace,
						Name:      "external-dns-test",
					},
				},
				{
					eventType: watch.Added,
					objType:   "serviceaccount",
					NamespacedName: types.NamespacedName{
						Namespace: testExtDNSNamespace,
						Name:      "external-dns-test",
					},
				},
				{
					eventType: watch.Added,
					objType:   "clusterrolebinding",
					NamespacedName: types.NamespacedName{
						Name: "external-dns-test",
					},
				},
				{
					eventType: watch.Added,
					objType:   "namespace",
					NamespacedName: types.NamespacedName{
						Name: "external-dns",
					},
				},
			},
		},
		{
			name:            "Deleted ExternalDNS",
			existingObjects: []runtime.Object{},
			inputConfig:     testConfig(),
			inputRequest:    testRequest(),
			expectedResult:  reconcile.Result{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cl := fake.NewClientBuilder().WithScheme(testScheme).WithRuntimeObjects(tc.existingObjects...).Build()
			r := &reconciler{
				client: cl,
				config: tc.inputConfig,
				log:    zap.New(zap.UseDevMode(true)),
			}
			// get watch interfaces from all the type managed by the operator
			watches := []watch.Interface{}
			for _, list := range managedTypesList {
				w, err := cl.Watch(context.TODO(), list)
				if err != nil {
					t.Fatalf("failed to start the watch for %T: %v", list, err)
				}
				watches = append(watches, w)
			}

			// TEST FUNCTION
			gotResult, err := r.Reconcile(context.TODO(), tc.inputRequest)

			// error check
			if err != nil {
				if !tc.errExpected {
					t.Fatalf("got unexpected error: %v", err)
				}
			}
			if tc.errExpected {
				t.Fatalf("error expected but not received")
			}
			// result check
			if !reflect.DeepEqual(gotResult, tc.expectedResult) {
				t.Fatalf("expected result %v, got %v", tc.expectedResult, gotResult)
			}
			// events check
			if len(tc.expectedEvents) == 0 {
				return
			}
			allEventsCh := make(chan watch.Event, len(watches))
			for _, w := range watches {
				go func(c <-chan watch.Event) {
					for e := range c {
						allEventsCh <- e
					}
				}(w.ResultChan())
			}
			idxExpectedEvents := indexTestEvents(tc.expectedEvents)
			for {
				select {
				case e := <-allEventsCh:
					key := watch2test(e).key()
					if _, exists := idxExpectedEvents[key]; !exists {
						t.Fatalf("unexpected event received: %v", e)
					}
					delete(idxExpectedEvents, key)
					if len(idxExpectedEvents) == 0 {
						return
					}
				case <-time.After(eventWaitTimeout):
					t.Fatalf("timed out waiting for all expected events")
				}
			}
		})
	}
}

type testEvent struct {
	eventType watch.EventType
	objType   string
	types.NamespacedName
}

func (e testEvent) key() string {
	return string(e.eventType) + "/" + e.objType + "/" + e.Namespace + "/" + e.Name
}

func indexTestEvents(events []testEvent) map[string]testEvent {
	m := map[string]testEvent{}
	for _, e := range events {
		m[e.key()] = e
	}
	return m
}

func watch2test(we watch.Event) testEvent {
	te := testEvent{
		eventType: we.Type,
	}

	switch obj := we.Object.(type) {
	case *appsv1.Deployment:
		te.objType = "deployment"
		te.Namespace = obj.Namespace
		te.Name = obj.Name
	case *corev1.ServiceAccount:
		te.objType = "serviceaccount"
		te.Namespace = obj.Namespace
		te.Name = obj.Name
	case *rbacv1.ClusterRole:
		te.objType = "clusterrole"
		te.Name = obj.Name
	case *rbacv1.ClusterRoleBinding:
		te.objType = "clusterrolebinding"
		te.Name = obj.Name
	case *corev1.Namespace:
		te.objType = "namespace"
		te.Name = obj.Name
	}
	return te
}

func testConfig() Config {
	return Config{
		Namespace: testExtDNSNamespace,
		Image:     "quay.io/test/external-dns:0.0.1",
	}
}

func testRequest() ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "",
			Name:      testExtDNSName,
		},
	}
}

func testExtDNSInstance() *operatorv1alpha1.ExternalDNS {
	return &operatorv1alpha1.ExternalDNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: testExtDNSName,
		},
		Spec: operatorv1alpha1.ExternalDNSSpec{
			Provider: operatorv1alpha1.ExternalDNSProvider{
				Type: operatorv1alpha1.ProviderTypeAWS,
			},
			Source: operatorv1alpha1.ExternalDNSSource{
				ExternalDNSSourceUnion: operatorv1alpha1.ExternalDNSSourceUnion{
					Type: operatorv1alpha1.SourceTypeService,
					Service: &operatorv1alpha1.ExternalDNSServiceSourceOptions{
						ServiceType: []corev1.ServiceType{
							corev1.ServiceTypeLoadBalancer,
						},
					},
				},
			},
			Zones: []string{"public-zone"},
		},
	}
}
