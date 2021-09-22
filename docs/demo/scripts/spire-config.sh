#!/bin/sh

kubectl -n spire exec spire-server-0 -- \
/opt/spire/bin/spire-server entry create \
-ttl 7200 \
-spiffeID spiffe://example.org/ns/spire/sa/spire-agent \
-selector k8s_sat:cluster:nsm-cluster \
-selector k8s_sat:agent_ns:spire \
-selector k8s_sat:agent_sa:spire-agent \
-node

kubectl -n spire exec spire-server-0 -- \
/opt/spire/bin/spire-server entry create \
-ttl 7200 \
-spiffeID spiffe://example.org/ns/default/sa/default \
-parentID spiffe://example.org/ns/spire/sa/spire-agent \
-selector k8s:ns:default \
-selector k8s:sa:default
