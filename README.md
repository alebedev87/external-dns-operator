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
3. Run `kubectl apply -k config/default`
4. Now you can deploy an instance of ExternalDNS:
    * Run the following command to create the credentials secret on AWS:
        ```bash
        kubectl -n external-dns-operator create secret generic aws-access-key \
                --from-literal=aws_access_key_id=${ACCESS_KEY_ID}
                --from-literal=aws_secret_access_key=${ACCESS_SECRET_KEY}
        ```
        *Note*: other provider's options can be found in `api/v1alpha1/externaldns_types.go`, i.e. `ExternalDNSAWSProviderOptions` structure for AWS
    * Run `kubectl apply -k config/samples/aws` for AWS    
        *Note*: other providers available in `config/samples/`

## TODO

### 4.10
- Complete deployment logic:
    - Validation/status
        - FQDNTemplate must be there if hostname annotation is allowed
        - Secret is mandatory
        - Put current ExternalDNS values into status (provder, source, etc.)
        - Available status
- Finalization/ownership logic
    - Secondary resources must be deleted when the primary one (ExternalDNS CR) is deleted
- Domain filtering support

### > 4.10
- Metrics
- CRD source
- Synchronization policies (`create-only`, `upsert-only`)
- Zone selection different from ZoneIDs (public/private, `azure-resource-group`, etc.)
