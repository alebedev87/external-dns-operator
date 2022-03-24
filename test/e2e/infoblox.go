//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	ibclient "github.com/infobloxopen/infoblox-go-client"

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/openshift/external-dns-operator/api/v1alpha1"
)

const (
	infobloxGridHostEnvVar     = "INFOBLOX_GRID_HOST"
	infobloxWAPIUsernameEnvVar = "INFOBLOX_WAPI_USERNAME"
	infobloxWAPIPasswordEnvVar = "INFOBLOX_WAPI_PASSWORD"
	trustedCAConfigMapEnvVar   = "TRUSTED_CA_CONFIGMAP_NAME"
	defaultWAPIPort            = "443"
	defaultWAPIVersion         = "2.3.1"
	defaultTLSVerify           = "false"
	defaultHTTPRequestTimeout  = 20
	defaultHTTPConnPool        = 10
	defaultGridMasterHostname  = "infoblox.localdomain"
	operatorContainerName      = "operator"
)

type infobloxTestHelper struct {
	client       *enhancedIBClient
	gridHost     string
	wapiUsername string
	wapiPassword string
}

var _ providerTestHelper = &infobloxTestHelper{}

func newInfobloxHelper(kubeClient client.Client) (*infobloxTestHelper, error) {
	helper := &infobloxTestHelper{}

	if err := helper.prepareConfigurations(kubeClient); err != nil {
		return nil, fmt.Errorf("failed to prepare infoblox helper: %w", err)
	}

	hostConfig := ibclient.HostConfig{
		Host:     helper.gridHost,
		Version:  defaultWAPIVersion,
		Port:     defaultWAPIPort,
		Username: helper.wapiUsername,
		Password: helper.wapiPassword,
	}
	transportConfig := ibclient.NewTransportConfig(defaultTLSVerify, defaultHTTPRequestTimeout, defaultHTTPConnPool)

	client, err := newEnhancedIBClient(hostConfig, transportConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to intiantiate enhanced infoblox client: %w", err)
	}
	helper.client = client

	return helper, nil
}

func (h *infobloxTestHelper) ensureHostedZone(zoneDomain string) (string, []string, error) {
	zones := []ibclient.ZoneAuth{}
	err := h.client.GetObject(ibclient.NewZoneAuth(ibclient.ZoneAuth{}), "", &zones)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list authoritative zone: %w", err)
	}
	for _, zone := range zones {
		if zone.Fqdn == zoneDomain {
			return zone.Ref, []string{h.gridHost}, nil
		}
	}

	authZone := ibclient.NewZoneAuth(ibclient.ZoneAuth{Fqdn: zoneDomain})
	ref, err := h.client.CreateObject(authZone)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create authoritative zone: %w", err)
	}
	authZone.Ref = ref

	// NS record is not added automatically with the zone creation
	if err = h.client.addNameServer(authZone.Ref, defaultGridMasterHostname); err != nil {
		return "", nil, fmt.Errorf("failed to add nameserver to authoritative zone: %w", err)
	}

	// creation of an authoritative zone needs a restart of DNS service
	if err = h.client.restartServices(); err != nil {
		return "", nil, fmt.Errorf("failed to restart grid services: %w", err)
	}

	return authZone.Ref, []string{h.gridHost}, nil
}

func (h *infobloxTestHelper) deleteHostedZone(zoneID, zoneDomain string) error {
	if _, err := h.client.DeleteObject(zoneID); err != nil {
		return err
	}

	// deletion of an authoritative zone needs a restart of DNS service
	if err := h.client.restartServices(); err != nil {
		return err
	}

	return nil
}

func (h *infobloxTestHelper) platform() string {
	return infobloxDNSProvider
}

func (h *infobloxTestHelper) makeCredentialsSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("infoblox-credentials-%s", randomString(16)),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"EXTERNAL_DNS_INFOBLOX_WAPI_USERNAME": []byte(h.wapiUsername),
			"EXTERNAL_DNS_INFOBLOX_WAPI_PASSWORD": []byte(h.wapiPassword),
		},
	}
}

func (h *infobloxTestHelper) buildExternalDNS(name, zoneID, zoneDomain string, credsSecret *corev1.Secret) operatorv1alpha1.ExternalDNS {
	resource := defaultExternalDNS(name, zoneID, zoneDomain)
	wapiPort, _ := strconv.Atoi(defaultWAPIPort)
	resource.Spec.Provider = operatorv1alpha1.ExternalDNSProvider{
		Type: operatorv1alpha1.ProviderTypeInfoblox,
		Infoblox: &operatorv1alpha1.ExternalDNSInfobloxProviderOptions{
			Credentials: operatorv1alpha1.SecretReference{
				Name: credsSecret.Name,
			},
			GridHost:    h.gridHost,
			WAPIPort:    wapiPort,
			WAPIVersion: defaultWAPIVersion,
		},
	}
	return resource
}

func (h *infobloxTestHelper) buildOpenShiftExternalDNS(name, zoneID, zoneDomain, routerName string, credsSecret *corev1.Secret) operatorv1alpha1.ExternalDNS {
	resource := routeExternalDNS(name, zoneID, zoneDomain, routerName)
	wapiPort, _ := strconv.Atoi(defaultWAPIPort)
	resource.Spec.Provider = operatorv1alpha1.ExternalDNSProvider{
		Type: operatorv1alpha1.ProviderTypeInfoblox,
		Infoblox: &operatorv1alpha1.ExternalDNSInfobloxProviderOptions{
			Credentials: operatorv1alpha1.SecretReference{
				Name: credsSecret.Name,
			},
			GridHost:    h.gridHost,
			WAPIPort:    wapiPort,
			WAPIVersion: defaultWAPIVersion,
		},
	}
	return resource
}

func (h *infobloxTestHelper) prepareConfigurations(kubeClient client.Client) error {
	h.gridHost = mustGetEnv(infobloxGridHostEnvVar)
	h.wapiUsername = mustGetEnv(infobloxWAPIUsernameEnvVar)
	h.wapiPassword = mustGetEnv(infobloxWAPIPasswordEnvVar)

	return h.trustGridTLSCert()
}

// trustGridTLSCert instructs the operator to trust Grid Master's self signed TLS certificate
func (h *infobloxTestHelper) trustGridTLSCert() error {
	// get Grid's TLS certificate as raw PEM encoded data
	certRaw, err := readServerTLSCert(net.JoinHostPort(h.gridHost, defaultWAPIPort), true)
	if err != nil {
		return fmt.Errorf("failed to get Grid Master's TLS certificate: %w", err)
	}

	operatorNs := "external-dns-operator"
	trustedCAConfigMapName := fmt.Sprintf("infoblox-trustedca-%s", randomString(16))

	// create a config map with the trusted CA bundle
	trustedCAConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trustedCAConfigMapName,
			Namespace: operatorNs,
		},
		Data: map[string]string{
			"ca-bundle.crt": string(certRaw),
		},
	}
	if err := kubeClient.Create(context.TODO(), trustedCAConfigMap); err != nil {
		return fmt.Errorf("failed to create trusted CA configmap %s/%s: %w", trustedCAConfigMap.Namespace, trustedCAConfigMap.Name, err)
	}

	// trusted CA environment variable to be injected
	trustedCAEnvVar := corev1.EnvVar{
		Name:  trustedCAConfigMapEnvVar,
		Value: trustedCAConfigMapName,
	}

	// inject into subscription if there is one
	findOperatorSubscription := func() (*olmv1alpha1.Subscription, error) {
		list := &olmv1alpha1.SubscriptionList{}
		if err := kubeClient.List(context.TODO(), list, client.InNamespace(operatorNs)); err != nil {
			return nil, err
		}
		for i := range list.Items {
			if strings.HasPrefix(list.Items[i].Name, "external-dns-operator") {
				return &list.Items[i], nil
			}
		}
		return nil, nil
	}
	subscription, err := findOperatorSubscription()
	if err != nil {
		return fmt.Errorf("failed to find operator subscription: %w", err)
	}
	if subscription != nil {
		if subscription.Spec.Config == nil {
			subscription.Spec.Config = &olmv1alpha1.SubscriptionConfig{}
		}
		subscription.Spec.Config.Env = ensureEnvVar(subscription.Spec.Config.Env, trustedCAEnvVar)
		if err := kubeClient.Update(context.TODO(), subscription); err != nil {
			return fmt.Errorf("failed to inject trusted CA environment variable into the subscription: %w", err)
		}
		return nil
	}

	// no subscription was found, try to update the deployment directly
	findOperatorDeployment := func() (*appsv1.Deployment, error) {
		list := &appsv1.DeploymentList{}
		if err := kubeClient.List(context.TODO(), list, client.InNamespace(operatorNs)); err != nil {
			return nil, err
		}
		for i := range list.Items {
			if strings.HasPrefix(list.Items[i].Name, "external-dns-operator") {
				return &list.Items[i], nil
			}
		}
		return nil, nil
	}
	deployment, err := findOperatorDeployment()
	if err != nil {
		return fmt.Errorf("failed to find operator deployment: %w", err)
	}
	if deployment == nil {
		return fmt.Errorf("no operator deployment found")
	}

	for i := range deployment.Spec.Template.Spec.Containers {
		if deployment.Spec.Template.Spec.Containers[i].Name == operatorContainerName {
			deployment.Spec.Template.Spec.Containers[i].Env = ensureEnvVar(deployment.Spec.Template.Spec.Containers[i].Env, trustedCAEnvVar)
			break
		}
	}
	if err := kubeClient.Update(context.TODO(), deployment); err != nil {
		return fmt.Errorf("failed to inject trusted CA environment variable into the deployment: %w", err)
	}

	return nil
}
