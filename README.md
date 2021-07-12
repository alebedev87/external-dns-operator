# ExternalDNS Operator

The ExternalDNS Operator deploys and manages [ExternalDNS](https://github.com/kubernetes-sigs/external-dns), which dynamically manages
external DNS records in specific DNS Providers for specific Kubernetes resources.

This Operator is in the early stages of implementation. For the time being, please reference the
[ExternalDNS Operator OpenShift Enhancement Proposal](https://github.com/openshift/enhancements/pull/786).

## Deploy operator

### Quick development
1. Build and push the operator image to a registry:
   ```sh
   $ podman build -t <registry>/<username>/external-dns-operator:latest -f Dockerfile .
   $ podman push <registry>/<username>/external-dns-operator:latest
   ```
2. Make sure to uncomment the `image` in `config/manager/kustomization.yaml` and set it to the operator image you pushed
3. Run `oc apply -k config/default`

### OperatorHub install with custom index image

This process refers to building the operator in a way that it can be installed locally via the OperatorHub with a custom index image.

1. Build and push the bundle image to a registry:
   ```sh
   $ podman build -t <registry>/<username>/external-dns-operator-bundle:latest -f Dockerfile.bundle .
   $ podman push <registry>/<username>/external-dns-operator-bundle:latest
   ```

2. Build and push the image index for operator-registry:
   ```sh
   # to get opm: https://github.com/operator-framework/operator-registry
   $ opm index add -c podman --bundles <registry>/<username>/external-dns-operator-bundle:latest --tag <registry>/<username>/external-dns-operator-bundle-index:1.0.0
   $ podman push <registry>/<username>/external-dns-operator-bundle-index:1.0.0
   ```

3. Create and apply catalogsource manifest:
   ```yaml
   apiVersion: operators.coreos.com/v1alpha1
   kind: CatalogSource
   metadata:
     name: external-dns-operator
     namespace: openshift-marketplace
   spec:
     sourceType: grpc
     image: <registry>/<username>/external-dns-operator-bundle-index:1.0.0
   ```

4. Create `external-dns-operator` namespace:
   ```sh
   $ oc create ns external-dns-operator
   ```

5. Open the console Operators -> OperatorHub, search for `ExternalDNS operator` and install the operator
