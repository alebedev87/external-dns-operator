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
	"fmt"
	"hash"
	"hash/fnv"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	operatorv1alpha1 "github.com/openshift/external-dns-operator/api/v1alpha1"
	operatorconfig "github.com/openshift/external-dns-operator/pkg/operator/config"
)

const (
	// ExternalDNSBaseName is a universal base name for any ExternalDNS instance
	ExternalDNSBaseName = "external-dns"
)

// ExternalDNSResourceName returns the name of ExternalDNS instance.
// It can be used for ExternalDNS resources related to this instance: deployment, servicename, etc.
func ExternalDNSResourceName(externalDNS *operatorv1alpha1.ExternalDNS) string {
	return ExternalDNSBaseName + "-" + externalDNS.Name
}

// ExternalDNSContainerName returns a name for the container of ExternalDNS deployment.
// The name will contain the hashed zone given as parameter.
func ExternalDNSContainerName(zone string) string {
	return ExternalDNSBaseName + "-" + hashString(zone)
}

// ExternalDNSDestCredentialsSecretName returns the namespaced name of the destination (operand) credentials secret
func ExternalDNSDestCredentialsSecretName(operandNamespace, extdnsName string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: operandNamespace,
		Name:      ExternalDNSBaseName + "-credentials-" + extdnsName,
	}
}

func ExternalDNSCredentialsSourceNamespace(cfg *operatorconfig.Config) string {
	// TODO: use openshift-config namespace for OpenShift
	return cfg.OperatorNamespace
}

func hashString(str string) string {
	hasher := getHasher()
	hasher.Write([]byte(str))
	return rand.SafeEncodeString(fmt.Sprint(hasher.Sum(nil)))
}

func getHasher() hash.Hash {
	return fnv.New32a()
}
