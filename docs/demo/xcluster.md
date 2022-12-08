# Demo - xcluster - vlan

* [Kind - VLAN](readme.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a vlan-forwarder to link the network service to an external host.
* [xcluster - VLAN](xcluster.md) - Demo running on [xcluster](https://github.com/Nordix/xcluster) using a vlan-forwarder to link the network service to an external host.

## Installation
Guide to deploy Merido and Meridio-Operator into k8s namespace "default"

### Kubernetes cluster

Deploy a Kubernetes cluster with xcluster into a dedicated network namespace and use meridio ovl  
Refer to: [xcluster-quick-start](https://github.com/Nordix/xcluster#quick-start),
[xcluster-netns](https://github.com/Nordix/xcluster/blob/master/doc/netns.md), 
[xcluster-ovl](https://github.com/Nordix/xcluster/blob/master/doc/overlays.md),
[private-docker-reg](https://github.com/Nordix/xcluster/blob/master/ovl/private-reg/README.md)
```
# consider creating a dedicated resolv.conf for the network namespace e.g.:
cat /etc/netns/[NAMESPACE_NAME]/resolv.conf 
nameserver 8.8.8.8
# in the dedicated network namespace
export XCLUSTER_OVLPATH="$XCLUSTER_OVLPATH:docs/demo/deployments/xcluster/ovl"
unset __mem1
unset __mem
# start xcluster with 2 workers and 2 routers (at least 1 worker is required)
xc mkcdrom meridio; xc starts --nets_vm=0,1,2 --nvm=2 --mem=4096 --smp=4
# or using private docker registry
xc mkcdrom private-reg meridio; xc starts --nets_vm=0,1,2 --nvm=2 --mem=4096 --smp=4
```

### External host / External connectivity

Deploy external gateway and traffic generator  
prerequisite; Multus is ready (deployed by meridio ovl)
```
# default interface setup
helm install docs/demo/deployments/xcluster/ovl/meridio/helm/gateway --generate-name
# eth1:meridio<-->gateways, eth2:gateways<-->tg
helm install docs/demo/deployments/xcluster/ovl/meridio/helm/gateway --generate-name --set masterItf=eth1,tgMasterItf=eth2 --create-namespace --namespace tg1
```

### NSM

Deploy Spire
```
kubectl apply -k docs/demo/deployments/spire
```

Deploy NSM
```
helm install docs/demo/deployments/nsm --generate-name --create-namespace --namespace nsm
```

### Meridio-Operator

Refer to the [description](https://github.com/Nordix/Meridio-Operator/blob/master/readme.md) on how to take hold of a Meridio-Operator image
if non available.  

```
# if using a private docker registry with xcluster
make docker-build docker-push IMG="localhost:80/meridio/meridio-operator:v0.0.1"
# NOT using a private registry; upload the built image to a registry of your choice
make docker-build docker-push IMG="registry.nordix.org/meridio/meridio-operator:v0.0.1"
```

Deploy Meridio-Operator
```
# deploy operator image from the Operator repository (irrespective of the registry type)
make deploy IMG="registry.nordix.org/meridio/meridio-operator:v0.0.1" NAMESPACE="default"
```

### Meridio

Install Meridio for trench-a through Meridio-Operator  
Note: vlan interface config in the Attractor Custom Resource must match the one used by external gateway PODs to connect Meridio
```
# add Trench
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Trench
metadata:
  name: trench-a
spec:
  ip-family: dualstack
EOF

# add Attractor
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Attractor
metadata:
  name: attr1
  labels:
    trench: trench-a
spec:
  replicas: 1
  gateways:
    - gateway1
    - gateway2
    - gateway3
    - gateway4
  composites:
    - load-balancer
  vips:
    - vip1
    - vip2
    - vip3
  interface:
    name: eth1.100
    ipv4-prefix: 169.254.100.0/24
    ipv6-prefix: 100:100::/64
    type: nsm-vlan
    nsm-vlan:
      vlan-id: 100
      base-interface: eth1
EOF

# add Gateways (bgp and static+bfd can be mixed)
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Gateway
metadata:
  labels:
    trench: trench-a
  name: gateway1
spec:
  address: 169.254.100.254
  protocol: bgp
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: "3s"
    local-port: 10179
    remote-port: 10179
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Gateway
metadata:
  labels:
    trench: trench-a
  name: gateway2
spec:
  address: 100:100::254
  protocol: bgp
  bgp:
    bfd:
      switch: true
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: "3s"
    local-port: 10179
    remote-port: 10179
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Gateway
metadata:
  labels:
    trench: trench-a
  name: gateway3
spec:
  address: 169.254.100.253
  protocol: bgp
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: "3s"
    local-port: 10179
    remote-port: 10179
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Gateway
metadata:
  labels:
    trench: trench-a
  name: gateway4
spec:
  address: 100:100::254
  protocol: static
EOF

# add Vips
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  labels:
    trench: trench-a
  name: vip1
spec:
  address: "20.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  labels:
    trench: trench-a
  name: vip2
spec:
  address: "10.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  labels:
    trench: trench-a
  name: vip3
spec:
  address: "2000::1/128"
EOF

# add Conduit
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Conduit
metadata:
  name: load-balancer
  labels:
    trench: trench-a
EOF

# add Stream to previously defined Conduit
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Stream
metadata:
  name: stream-a
  labels:
    trench: trench-a
spec:
  conduit: load-balancer
EOF

# add Flow to previously defined Stream
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Flow
metadata:
  name: flow-1
  labels:
    trench: trench-a
spec:
  stream: stream-a
  priority: 2
  vips:
  - vip1
  - vip2
  - vip3
  source-subnets:
  - 0.0.0.0/0
  - ::/0
  source-ports:
  - any
  destination-ports:
  - 2000-8000
  protocols:
  - tcp
  - udp
EOF
```

## Target

Install targets connected to trench-a and conduit "load-balancer"
```
helm install examples/target/deployments/helm/ --generate-name --namespace default --set applicationName=target-a --set default.trench.name=trench-a --set default.conduit.name=load-balancer
```

## Traffic

Connect to the Traffic Generator POD
```
# exec into traffic generator POD
kubectl exec -ti tg -n tg1 -- bash
```

Generate traffic
```
# ipv4
./ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 91/91/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 180/180/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 297/297/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 400/400/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 491/491/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 582/582/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 699/698/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 800/800/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 891/891/0
# ipv6
./ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v6traffic.json
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 91/91/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 180/180/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 297/297/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 400/400/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 491/491/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 582/582/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 699/698/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 800/800/0
Conn act/fail/connecting: 400/0/0, Packets send/rec/dropped: 891/891/0
```

Verification
```
./ctraffic -analyze hosts -stat_file v4traffic.json
Lost connections: 0
Lasting connections: 400
  target-a-7659b8d89f-2s795 98
  target-a-7659b8d89f-ppnks 104
  target-a-7659b8d89f-sg6nb 98
  target-a-7659b8d89f-vg2nw 100
./ctraffic -analyze hosts -stat_file v6traffic.json
Lost connections: 0
Lasting connections: 400
  target-a-7659b8d89f-2s795 93
  target-a-7659b8d89f-ppnks 105
  target-a-7659b8d89f-sg6nb 97
  target-a-7659b8d89f-vg2nw 105

```

## Scale Attractor

Update the replicas field of the Attractor
```
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Attractor
metadata:
  name: attr1
  labels:
    trench: trench-a
spec:
  replicas: 2
  gateways:
    - gateway1
    - gateway2
    - gateway3
    - gateway4
  composites:
    - load-balancer
  vips:
    - vip1
    - vip2
    - vip3
  interface:
    name: eth1.100
    ipv4-prefix: 169.254.100.0/24
    ipv6-prefix: 100:100::/64
    type: nsm-vlan
    nsm-vlan:
      vlan-id: 100
      base-interface: eth1
EOF

```

## Ambassador

Refer to the [description](https://github.com/Nordix/Meridio/blob/master/docs/demo/readme.md#ambassador).
