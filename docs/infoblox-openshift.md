# Use ExternalDNS Operator on Openshift with Infoblox provider

- [Steps](#steps)
- [Provision Infoblox on AWS](#provision-infoblox-on-aws)
    - [Prerequisites](#prerequisites)
    - [Provision manually](#provision-manually)
    - [Infoblox configuration](#infoblox-configuration)
- [Links](#links)

## Steps

**Note**: This guide assumes that Infoblox is setup and ready to be used for the DNS queries. You can follow the instructions from [Provision Infoblox on AWS chapter](#provision-infoblox-on-aws) if you want to setup Infoblox on AWS using the marketplace products. ExternalDNS Operator though can be deployed on any environment (AWS, Azure, GCP, locally)

1. _Optional_: In case Infoblox uses a self signed certificate, add its CA as trusted to ExternalDNS Operator:
```sh
oc -n external-dns-operator create configmap trusted-ca-infoblox --from-file=ca-bundle.crt=/path/to/pem/encoded/infoblox/ca
oc -n external-dns-operator patch subscription external-dns-operator --type='json' -p='[{"op": "add", "path": "/spec/config", "value":{"env":[{"name":"TRUSTED_CA_CONFIGMAP_NAME","value":"trusted-ca-infoblox"}]}}]'
```

2. Create a secret with Infoblox credentials:
```sh
oc -n external-dns-operator create secret generic infoblox-credentials --from-literal=EXTERNAL_DNS_INFOBLOX_WAPI_USERNAME=${INFOBLOX_USERNAME} --from-literal=EXTERNAL_DNS_INFOBLOX_WAPI_PASSWORD=${INFOBLOX_PASSWORD}
```

3. Get the routes to check your cluster's domain (everything after `apps.`):
```sh
$ oc get routes --all-namespaces | grep console
openshift-console          console             console-openshift-console.apps.aws.devcluster.openshift.com                       console             https   reencrypt/Redirect     None
openshift-console          downloads           downloads-openshift-console.apps.aws.devcluster.openshift.com                     downloads           http    edge/Redirect          None
```

4. Add an authoritative DNS zone for your cluster's domain to Infoblox using its WebUI:
    - `Data Management` top tab -> `DNS` subtab -> `Add` right panel -> `Zone` -> `Authoritative Zone` -> `Forward mapping`
    - Put cluster's domain as zone name (e.g. `aws.devcluster.openshift.com`)
    - Add Grid Primary as nameserver (`+` button on top of the table)
    - `Save & Close`

5. Create a [ExternalDNS CR](../../config/samples/infoblox/operator_v1alpha1_infoblox_detailed.yaml) as follows:
    ```sh
    cat <<EOF | oc create -f -
    apiVersion: externaldns.olm.openshift.io/v1alpha1
    kind: ExternalDNS
    metadata:
      name: sample-infoblox
    spec:
      provider:
        type: Infoblox
        infoblox:
          credentials:
            name: infoblox-credentials
          gridHost: ${INFOBLOX_GRID_PUBLIC_IP}
          wapiPort: 443
          wapiVersion: "2.3.1"
      domains:
      - filterType: Include
        matchType: Exact
        name: aws.devcluster.openshift.com
      source:
        type: OpenShiftRoute
        openshiftRouteOptions:
          routerName: default
    EOF
    ```

6. Check the records created for `console` routes:
    ```sh
    $ dig @${INFOBLOX_GRID_PUBLIC_IP} $(oc -n openshift-console get route console --template='{{range .status.ingress}}{{if eq "default" .routerName}}{{.host}}{{end}}{{end}}') +short
    router-default.apps.aws.devcluster.openshift.com
    $ dig @${INFOBLOX_GRID_PUBLIC_IP} $(oc -n openshift-console get route downloads --template='{{range .status.ingress}}{{if eq "default" .routerName}}{{.host}}{{end}}{{end}}') +short
    router-default.apps.aws.devcluster.openshift.com
    ```

## Provision Infoblox on AWS

### Prerequisites
Make sure your AWS account is subscribed to the following product:
- [Infoblox vNIOS for DDI](https://aws.amazon.com/marketplace/pp/prodview-opxe3p2cgudwe)

### Provision manually
**Note**: all the steps described in this chapter are well detailed in Infoblox guide which you can find in the [links](#links)

- Create VPC with public subnets to host NIOS VM:
    ```sh
    export AWS_PROFILE=MY_AWS_PROFILE
    aws cloudformation create-stack --stack-name infoblox-test --template-body file://${PWD}/scripts/cloud-formation-infoblox.yaml --parameters ParameterKey=EnvironmentName,ParameterValue=infoblox-test
    ```
- Launch NIOS instance:
    - Go to `AWS Marketplace/Subscriptions`, search for `Infoblox` and click on `Launch instance`
    - Choose the machine type from the list of recommended (enabled) ones
    - Choose the previously created VPC and one of the subnets
    - NIOS needs 2 network interfaces, primary one (`eth0`) is for the management, add another device for the secondary network interface (`eth1`). Make sure `eth1` is not from the subnet of `eth0` interface.
    - Add the user data to enable SSH and setup the password and the temporary license:
    ```text
    #infoblox-config
    remote_console_enabled: y
    default_admin_password: MY_COMPLEX_PASSWORD
    temp_license: enterprise dns dhcp cloud nios IB-V825
    ```
    - Add the storage: default (preconfigured) is fine
    - Add security group:
        - Use the default (preconfigured) one
        - Add TCP DNS port: `53`
        - Add UDP DHCP ports: `67-68`
    - Use existing or create a new keypair
    - Launch the instance and wait for all the status checks to pass (takes ~ 10 mins)
- Setup Elastic IP:
    - _Optional_: Rename `eth1` network interface so that it can be easily found when attached to Elastic IP
    - Allocate an Elastic IP (Amazon's pool is fine)
    - Associate this Elastic IP with `eth1` network interface (`Actions` dropdown list)
    - Choose the proposed private IP
- Now you can try to connect to the Grid Manager WebUI using the Elastic IP and `admin/PASSWORD_FROM_USER_DATA` as credentials. Use HTTPS scheme and accept the self signed certificate.

### Infoblox configuration

- Follow `Use vNIOS Instance for New grid` chapter from the guide which can be found in the [links](#links):
    - Reset the shared password (not sure whether it matters or not)
    - Admin password is better to be reset
    - Restart will be needed at the end of the setup of the new grid
- Start DNS service: `Grid` top tab -> `Grid manager` subtab -> `DNS` -> Select `infoblox.locald` and button `Start`
- Add name server group with the grid server: `Data Management` top tab -> `DNS` subtab -> `Add` right panel -> `Group` -> `Authorative` -> Put a name -> `+` button -> `Add Grid Primary` -> `Select` -> `Add` -> `Save & Close`
- Default self signed certificate uses the private IP in SAN. There can be multiple options of how to cope with that:
    - Regenerate the self signed certificate with the Elastic IP:
        - `Grid` -> `Grid Manager` -> `DNS` -> `Certificates` right panel -> `HTTPS Cert` -> `Generate Self Signed Certificate` -> Put Elastic IP in `Subject Alternative name` -> Accept the restart of the service
    - Generate a certificate signed by the CA known to your OpenShift cluster, then upload it to Infoblox:
        - `Grid` -> `Grid Manager` -> `DNS` -> `Certificates` right panel -> `HTTPS Cert` -> `Upload Certificate`

## Links
- [Deploy Infoblox vNIOS instances for AWS](https://www.infoblox.com/wp-content/uploads/infoblox-deployment-guide-deploy-infoblox-vnios-instances-for-aws.pdf)
- [Grid Manager. Managing certificates](https://docs.infoblox.com/display/NAG8/Managing+Certificates)
