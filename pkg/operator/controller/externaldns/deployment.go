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
	"sort"

	"github.com/google/go-cmp/cmp"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	operatorv1alpha1 "github.com/openshift/external-dns-operator/api/v1alpha1"
	controller "github.com/openshift/external-dns-operator/pkg/operator/controller"
)

const (
<<<<<<< HEAD
	startPortMetrics    = 7979
	appNameLabel        = "app.kubernetes.io/name"
	appInstanceLabel    = "app.kubernetes.io/instance"
	masterNodeRoleLabel = "node-role.kubernetes.io/master"
	osLabel             = "kubernetes.io/os"
	linuxOS             = "linux"
=======
	startPortMetrics = 7979
>>>>>>> 41263c5... begin implementing deployment logic
)

// providerStringTable maps ExternalDNSProviderType values from the
// ExternalDNS operator API to the provider string argument expected by ExternalDNS.
var providerStringTable = map[operatorv1alpha1.ExternalDNSProviderType]string{
	operatorv1alpha1.ProviderTypeAWS:      "aws",
	operatorv1alpha1.ProviderTypeGCP:      "google",
	operatorv1alpha1.ProviderTypeAzure:    "azure",
	operatorv1alpha1.ProviderTypeBlueCat:  "bluecat",
	operatorv1alpha1.ProviderTypeInfoblox: "infoblox",
}

// sourceStringTable maps ExternalDNSSourceType values from the
// ExternalDNS operator API to the source string argument expected by ExternalDNS.
var sourceStringTable = map[operatorv1alpha1.ExternalDNSSourceType]string{
	operatorv1alpha1.SourceTypeRoute:   "openshift-route",
	operatorv1alpha1.SourceTypeService: "service",
	// TODO: Add CRD source support
}

// ensureExternalDNSDeployment ensures that the externalDNS deployment exists.
// Returns a boolean if the deployment exists, and an error when relevant.
func (r *reconciler) ensureExternalDNSDeployment(ctx context.Context, namespace, image string, serviceAccount *corev1.ServiceAccount, externalDNS *operatorv1alpha1.ExternalDNS) (bool, *appsv1.Deployment, error) {
	desired, err := desiredExternalDNSDeployment(namespace, image, serviceAccount, externalDNS)
	if err != nil {
		return false, nil, fmt.Errorf("failed to build externalDNS deployment: %w", err)
	}

	exist, current, err := r.currentExternalDNSDeployment(ctx, namespace, externalDNS)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get externalDNS deployment: %w", err)
	}

	// create the deployment
	if !exist {
		if err := r.createExternalDNSDeployment(ctx, desired); err != nil {
			return false, nil, err
		}
		// getting the deployment from API in case a third party updated it after the creation
		return r.currentExternalDNSDeployment(ctx, namespace, externalDNS)
	}

	// update the deployment
	if _, err := r.updateExternalDNSDeployment(ctx, current, desired); err != nil {
		return true, current, err
	}

	// getting the deployment from API in case a third party updated it after the update
	return r.currentExternalDNSDeployment(ctx, namespace, externalDNS)
}

// currentExternalDNSDeployment gets the current externalDNS deployment resource.
func (r *reconciler) currentExternalDNSDeployment(ctx context.Context, namespace string, externalDNS *operatorv1alpha1.ExternalDNS) (bool, *appsv1.Deployment, error) {
	depl := &appsv1.Deployment{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: externalDNS.Name}, depl); err != nil {
		if errors.IsNotFound(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, depl, nil
}

// desiredExternalDNSDeployment returns the desired deployment resource.
func desiredExternalDNSDeployment(namespace, image string, serviceAccount *corev1.ServiceAccount, externalDNS *operatorv1alpha1.ExternalDNS) (*appsv1.Deployment, error) {
	replicas := int32(1)

	matchLbl := map[string]string{
		appNameLabel:     controller.ExternalDNSBaseName,
		appInstanceLabel: externalDNS.Name,
	}

	nodeSelectorLbl := map[string]string{
		osLabel:             linuxOS,
		masterNodeRoleLabel: "",
	}

    controlPlaneTolerations := []corev1.Toleration{
        {
            Key:      masterNodeRoleLabel,
            Operator: corev1.TolerationOpExists,
            Effect:   corev1.TaintEffectNoSchedule,
        },
    }

	depl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controller.ExternalDNSResourceName(externalDNS),
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLbl,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: matchLbl,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount.Name,
					NodeSelector:       nodeSelectorLbl,
					Tolerations: controlPlaneTolerations,
				},
			},
		},
	}

	for i, zone := range externalDNS.Spec.Zones {
		container, err := populateExternalDNSContainer(image, startPortMetrics+i, zone, externalDNS)
		if err != nil {
			return nil, err
		}
		depl.Spec.Template.Spec.Containers = append(depl.Spec.Template.Spec.Containers, *container)
	}

	return depl, nil
}

// WIP function for generating container specs for one container at a time.
func populateExternalDNSContainer(image string, metricsPort int, zone string, externalDNS *operatorv1alpha1.ExternalDNS) (*corev1.Container, error) {
	name := "externaldns-" + zone

	args := []string{
		fmt.Sprintf("--metrics-address=127.0.0.1:%d", metricsPort),
		fmt.Sprintf("--txt-owner-id=externaldns-%s", externalDNS.Name),
		fmt.Sprintf("--zone-id-filter=%s", zone),
		"--policy=sync",
		"--registry=txt",
		"--log-level=debug",
	}

	provider, ok := providerStringTable[externalDNS.Spec.Provider.Type]
	if !ok {
		return nil, fmt.Errorf("%s is not a supported provider", externalDNS.Spec.Provider.Type)
	}
	args = append(args, fmt.Sprintf("--provider=%s", provider))

	//TODO: Add provider credentials logic

	source, ok := sourceStringTable[externalDNS.Spec.Source.Type]
	if !ok {
		return nil, fmt.Errorf("%s is not a supported source type", externalDNS.Spec.Source.Type)
	}
	args = append(args, fmt.Sprintf("--source=%s", source))

	if externalDNS.Spec.Source.Namespace != nil && len(*externalDNS.Spec.Source.Namespace) > 0 {
		args = append(args, fmt.Sprintf("--namespace=%s", *externalDNS.Spec.Source.Namespace))
	}

	if externalDNS.Spec.Source.Service != nil && len(externalDNS.Spec.Source.Service.ServiceType) > 0 {
		serviceTypeFilter, publishInternal := "", false
		for _, serviceType := range externalDNS.Spec.Source.Service.ServiceType {
			serviceTypeFilter += string(serviceType) + ","
			if serviceType == corev1.ServiceTypeClusterIP {
				publishInternal = true
			}
		}

		// avoid having a trailing comma
		args = append(args, fmt.Sprintf("--service-type-filter=%s", serviceTypeFilter[0:len(serviceTypeFilter)-1]))

		if publishInternal {
			args = append(args, "--publish-internal-services")
		}
	}

	if externalDNS.Spec.Source.HostnameAnnotationPolicy == operatorv1alpha1.HostnameAnnotationPolicyIgnore {
		args = append(args, "--ignore-hostname-annotation")
	}

	//TODO: Add logic for the CRD source.

	return &corev1.Container{
		Name:                     name,
		Image:                    image,
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
		Args:                     args,
	}, nil
}

// createExternalDNSDeployment creates the given deployment using the reconciler's client.
func (r *reconciler) createExternalDNSDeployment(ctx context.Context, depl *appsv1.Deployment) error {
	if err := r.client.Create(ctx, depl); err != nil {
		return fmt.Errorf("failed to create externalDNS deployment %s/%s: %w", depl.Namespace, depl.Name, err)
	}
	r.log.Info("created externalDNS deployment", "namespace", depl.Namespace, "name", depl.Name)
	return nil
}

// updateExternalDNSDeployment updates the in-cluster externalDNS deployment.
// Returns a boolean if an update was made, and an error when relevant.
func (r *reconciler) updateExternalDNSDeployment(ctx context.Context, current, desired *appsv1.Deployment) (bool, error) {
	// don't always update (or do simple DeepEqual with) the operand's deployment
	// as this may result into a "fight" between API/admission webhooks and this controller
	// example:
	//  - this controller creates a deployment with the desired fields A, B, C
	//  - API adds some default fields D, E, F (i.e. metadata, imagePullPullPolicy, dnsPolicy)
	//  - mutating webhooks add default values to fields E, F
	//  - this controller gets into reconcile loop and starts all from step 1
	//  - checking that fields A, B, C are the same as desired would save us the before mentioned round trips:
	changed, updated := externalDNSDeploymentChanged(current, desired)
	if !changed {
		return false, nil
	}

	if err := r.client.Update(ctx, updated); err != nil {
		return false, fmt.Errorf("failed to update externalDNS deployment %s/%s: %w", desired.Namespace, desired.Name, err)
	}
	r.log.Info("updated externalDNS deployment", "namespace", desired.Namespace, "name", desired.Name)
	return true, nil
}

// externalDNSDeploymentChange evaluates whether or not a deployment update is necessary.
// Returns a boolean if an update is necessary, and the deployment resource to update to.
func externalDNSDeploymentChanged(current, expected *appsv1.Deployment) (bool, *appsv1.Deployment) {
	changed := false
	updated := current.DeepCopy()

	if len(current.Spec.Template.Spec.Containers) > 0 && len(expected.Spec.Template.Spec.Containers) > 0 {
		if current.Spec.Template.Spec.Containers[0].Image != expected.Spec.Template.Spec.Containers[0].Image {
			updated.Spec.Template.Spec.Containers[0].Image = expected.Spec.Template.Spec.Containers[0].Image
			changed = true
		}
		currArgs := append([]string{}, current.Spec.Template.Spec.Containers[0].Args...)
		expArgs := append([]string{}, expected.Spec.Template.Spec.Containers[0].Args...)
		sort.Strings(currArgs)
		sort.Strings(expArgs)
		if !cmp.Equal(currArgs, expArgs) {
			updated.Spec.Template.Spec.Containers[0].Args = expected.Spec.Template.Spec.Containers[0].Args
			changed = true
		}
	}

	if !changed {
		updated = nil
	}

	return changed, updated
}
