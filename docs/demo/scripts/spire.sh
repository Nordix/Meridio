#!/bin/sh

registerWorkload()
{
    local serviceAccount="$1"
    local namespace="$2"
    kubectl -n spire exec spire-server-0 -- \
    /opt/spire/bin/spire-server entry create \
    -ttl 7200 \
    -spiffeID spiffe://example.org/ns/$namespace/sa/$serviceAccount \
    -parentID spiffe://example.org/ns/spire/sa/spire-agent \
    -selector k8s:ns:$namespace \
    -selector k8s:sa:$serviceAccount
}

if [ -z "$1" ]
then
   echo "No trench specified";
   exit 1
fi

if [ -z "$2" ]
then
   echo "No namespace specified";
   exit 1
fi

registerWorkload "$1" "$2"

