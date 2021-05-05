#!/bin/sh

kubectl -n spire exec spire-server-0 -- \
/opt/spire/bin/spire-server entry create \
-ttl 72000 \
-spiffeID spiffe://example.org/ns/trench-a/sa/default \
-parentID spiffe://example.org/ns/spire/sa/spire-agent \
-selector k8s:ns:trench-a \
-selector k8s:sa:default

kubectl -n spire exec spire-server-0 -- \
/opt/spire/bin/spire-server entry create \
-ttl 72000 \
-spiffeID spiffe://example.org/ns/trench-b/sa/default \
-parentID spiffe://example.org/ns/spire/sa/spire-agent \
-selector k8s:ns:trench-b \
-selector k8s:sa:default