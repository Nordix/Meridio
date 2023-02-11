#!/bin/bash

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh
./get_helm.sh

git clone https://github.com/Nordix/Meridio.git

cd Meridio

docker pull registry.gitlab.com/lionelj/meridio/kind-host:latest
docker tag registry.gitlab.com/lionelj/meridio/kind-host:latest registry.nordix.org/cloud-native/meridio/kind-host:latest

make -s -C docs/demo/scripts/kind/ KUBERNETES_VERSION="v1.26" KUBERNETES_IP_FAMILY="v1.26" NSM_VERSION="v1.7.1" 

helm install meridio-crds https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-CRDs-v1.0.0.tgz --create-namespace --namespace red
helm install meridio https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-v1.0.0.tgz --create-namespace --namespace red --set registry="registry.gitlab.com" --set repository="lionelj/meridio" --set nsm.repository="lionelj/meridio"

sleep 10

kubectl wait --for=condition=Ready pods --all -n red --timeout=5m

sleep 30

cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Trench
metadata:
  name: trench-a
  namespace: red
spec:
  ip-family: dualstack
---
apiVersion: meridio.nordix.org/v1
kind: Attractor
metadata:
  name: attractor-a-1
  namespace: red
  labels:
    trench: trench-a
spec:
  replicas: 2
  composites:
  - conduit-a-1
  gateways:
  - gateway-v4-a
  - gateway-v6-a
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  interface:
    name: ext-vlan0
    ipv4-prefix: 169.254.100.0/24
    ipv6-prefix: 100:100::/64
    type: nsm-vlan
    nsm-vlan:
      vlan-id: 100
      base-interface: eth0
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v4-a
  namespace: red
  labels:
    trench: trench-a
spec:
  address: 169.254.100.150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v6-a
  namespace: red
  labels:
    trench: trench-a
spec:
  address: 100:100::150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  name: vip-a-1-v4
  namespace: red
  labels:
    trench: trench-a
spec:
  address: "20.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  name: vip-a-1-v6
  namespace: red
  labels:
    trench: trench-a
spec:
  address: "2000::1/128"
---
apiVersion: meridio.nordix.org/v1
kind: Conduit
metadata:
  name: conduit-a-1
  namespace: red
  labels:
    trench: trench-a
spec:
  type: stateless-lb
---
apiVersion: meridio.nordix.org/v1
kind: Stream
metadata:
  name: stream-a-i
  namespace: red
  labels:
    trench: trench-a
spec:
  conduit: conduit-a-1
---
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a-z-tcp
  namespace: red
  labels:
    trench: trench-a
spec:
  stream: stream-a-i
  priority: 1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  source-subnets:
  - 0.0.0.0/0
  - 0:0:0:0:0:0:0:0/0
  source-ports:
  - any
  destination-ports:
  - "4000"
  protocols:
  - tcp
EOF

sleep 10

kubectl wait --for=condition=Ready pods --all -n red --timeout=5m

helm install meridio-target-a https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-Target-v1.0.0.tgz --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a --set default.conduit.name=conduit-a-1 --set default.stream.name=stream-a-i --set registry="registry.gitlab.com" --set repository="lionelj/meridio"

cd ..
