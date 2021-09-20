## Demo

* Follow the instructions from [main README](https://github.com/alebedev87/external-dns-operator/blob/functional-operator/README.md) to deploy the operator

* Show the operator deployment:
    ```bash
    oc -n external-dns-operator get pods
    oc -n external-dns-operator logs deploy/external-dns-operator -c external-dns-operator -f
    ```

* Use this to get AWS credentials:
    ```bash
    ACCESS_KEY_ID=$(base64 -d <(oc -n kube-system get secret aws-creds --template='{{.data.aws_access_key_id}}'))
    ACCESS_SECRET_KEY=$(base64 -d <(oc -n kube-system get secret aws-creds --template='{{.data.aws_secret_access_key}}'))
    ```

* Follow again the instruction from [main README](https://github.com/alebedev87/external-dns-operator/blob/functional-operator/README.md) to create the secret

* Explain the contents of [sample-extdns-mycluster.yaml](./sample-extdns-mycluster.yaml) and make sure it contains the right zone ID

* Create ExternalDNS which targets `alebedev-dev.devcluster.openshift.com` domain in `openshift-dev` AWS account:
    ```bash
    oc apply -f sample-extdns-mycluster.yaml
    ```

* Show what was created:
    ```bash
    oc get ns external-dns
    oc get clusterrole external-dns
    oc get clusterrolebinding external-dns-sample
    oc -n external-dns get pod,deploy,sa,secret
    oc -n external-dns logs deploy/external-dns-sample -f
    ```

* Create the source for DNS records:
    ```bash
    ./create-source.sh
    ```

* Show the source services:
    ```bash
    oc -n publish-external-dns get svc,pod
    ```

* Check the DNS records in the hosted zone (WebUI or CMD):
    * Show the difference between LoadBalancer/ClusterIP/NodePort services
    * Show the TXT records
    * Show that the other records from the zone are left untouched

* Remove the NodePort service, wait for 1 minute and show that corresponding A and TXT records were removed:
    ```bash
    oc -n publish-external-dns delete svc alebedev-httpserver3
    ```

* Update ExternalDNS to target the public `devcluster.openshift.com` domain:
    ```bash
    oc apply -f sample-extdns.yaml
    ```

* Show that the whole chain is working:
    ```bash
    curl http://alebedev-httpserver.devcluster.openshift.com
    ```

* Remove all the services and **wait for 1 minute** to let ExternalDNS do the cleanup. We don't want the public `devcluster.openshift.com` domain to stay with our records:
    ```bash
    oc delete ns publish-external-dns
    ```

* Remove ExternalDNS instance:
    ```bash
    oc delete externaldns sample
    ```

* Show that all the resources are cleaned up:
    ```bash
    oc -n external-dns get pod,deploy,sa,secret
    oc get clusterrolebinding external-dns-sample
    ```

## Where we are

The shown functionality is only possible with the following PR:
* [PR22](https://github.com/openshift/external-dns-operator/pull/22) (needs to be merged)

The other tasks are broken down in [the epic](https://issues.redhat.com/browse/NE-303).

## Helper commands

* Run this to see ExternalDNS logs:
    ```bash
    oc -n external-dns logs deploy/external-dns-sample -f
    ```

* Run this to disable operator:
    ```bash
    oc -n external-dns-operator scale deploy/external-dns-operator --replicas=0
    ```

* Hostanme annotation:
    ```bash
    oc annotate svc httpserver external-dns.alpha.kubernetes.io/hostname=httpserver.devcluster.openshift.com
    oc annotate svc httpserver2 external-dns.alpha.kubernetes.io/hostname=httpserver2.devcluster.openshift.com
    ```

* List A records:
    ```bash
    aws route53 list-resource-record-sets --output json --hosted-zone-id "/hostedzone/Z06076472WHIZX9L24NVR" --query "ResourceRecordSets[?Type == 'A']"
    ```

* Create a hosted zone:
    ```bash
    aws route53 create-hosted-zone --name "extdnstest.devcluster.openshift.com" --caller-reference "external-dns-test-$(date +%s)"
    ```
