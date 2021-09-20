#! /bin/bash

set -e

oc new-project publish-external-dns && sleep 5
quay-alebedev87 publish-external-dns default
oc run httpserver --image=quay.io/alebedev87/http-server:latest && sleep 5
oc expose pod httpserver --name=alebedev-httpserver --port=80 --target-port=8080 --type=LoadBalancer
oc annotate svc alebedev-httpserver external-dns.mydomain.org/publish=yes
oc expose pod httpserver --name=alebedev-httpserver2 --port=80 --target-port=8080
oc annotate svc alebedev-httpserver2 external-dns.mydomain.org/publish=yes
oc expose pod httpserver --name=alebedev-httpserver3 --port=80 --target-port=8080 --type=NodePort
oc annotate svc alebedev-httpserver3 external-dns.mydomain.org/publish=yes
