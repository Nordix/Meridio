# Demo - xcluster - vlan

* [Kind - VLAN](readme.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a vlan-forwarder to link the network service to an external host.
* [xcluster - VLAN](xcluster.md) - Demo running on [xcluster](https://github.com/Nordix/xcluster) using a vlan-forwarder to link the network service to an external host.

## Installation

### Kubernetes cluster

Deploy a Kubernetes cluster with xcluster while using meridio ovl  
(Note: also adjust XCLUSTER_OVLPATH env variable to include ovls)
```
unset __mem1
export __mem201=1024
export __mem202=1024
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
# eth1:meridio--gateways, eth2:gateways--tg
helm install docs/demo/deployments/xcluster/ovl/meridio/helm/gateway --generate-name --set masterItf=eth1,tgMasterItf=eth2
```

### NSM

Deploy Spire
```
helm install docs/demo/deployments/spire/ --generate-name
```

Configure Spire
```
docs/demo/scripts/spire-config.sh
```

Deploy NSM
```
helm install docs/demo/deployments/nsm-vlan/ --generate-name
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
# install most recent cert-manager
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
# alternatively install a specific version of cert-manager (useful when running with private docker registry)
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.5.1/cert-manager.yaml
# deploy operator image (irrespective of the registry type)
make deploy IMG="registry.nordix.org/meridio/meridio-operator:v0.0.1" NAMESPACE="default"
```

### Meridio

Configure Spire for trench-a
```
docs/demo/scripts/spire.sh meridio-trench-a default
```

Install Meridio for trench-a through Meridio-Operator  
Note: vlan-interface in Custom Resource (CR) Attractor must match the one used by external gateway PODs
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
  gateways:
  - gateway1
  - gateway2
  - gateway3
  - gateway4
  vips:
  - vip1
  - vip2
  - vip3
  replicas: 2
  vlan-id: 100
  vlan-interface: eth1
  vlan-ipv4-prefix: 169.254.100.0/24
  vlan-ipv6-prefix: 100:100::/64
EOF

# add Gateways
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Gateway
metadata:
  labels:
    trench: trench-a
    attractor: attr1
  name: gateway1
spec:
  address: 169.254.100.254
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
    attractor: attr1
  name: gateway2
spec:
  address: fe80::beef
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
    attractor: attr1
  name: gateway3
spec:
  address: 169.254.100.253
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
    attractor: attr1
  name: gateway4
spec:
  address: fe80::beee
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: "3s"
    local-port: 10179
    remote-port: 10179
EOF

# add Vips
cat <<EOF | kubectl apply -f -
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  labels:
    trench: trench-a
    attractor: attr1
  name: vip1
spec:
  address: "20.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  labels:
    trench: trench-a
    attractor: attr1
  name: vip2
spec:
  address: "10.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  labels:
    trench: trench-a
    attractor: attr1
  name: vip3
spec:
  address: "2000::1/128"
EOF
```

## Target

Configure Spire for the targets
```
./docs/demo/scripts/spire.sh meridio default
```

Deploy common resources for the targets
```
helm install examples/target/common/ --generate-name
```

Install targets connected to trench-a
```
helm install examples/target/helm/ --generate-name --set applicationName=target-a --set default.trench.name=trench-a --set configMapName=meridio-configuration --set networkService=lb-fe
```

## Traffic

Connect to the Traffic Generator POD
```
# exec into traffic generator POD
kubectl exec -ti tg -- bash
```

Ping
```
ping 20.0.0.1 -c 3
ping 10.0.0.1 -c 3
ping 2000::1 -c 3

```

Generate traffic
```
# ipv4
./ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
# ipv6
./ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
```

Verification
```
./ctraffic -analyze hosts -stat_file v4traffic.json
```

## Ambassador

From a target:

Connect to a conduit/trench (Conduit/Network Service: lb-fe, Trench: trench-a)
```
./target-client connect -ns lb-fe -t trench-a
```

Disconnect from a conduit/trench (Conduit/Network Service: lb-fe, Trench: trench-a)
```
./target-client disconnect -ns lb-fe -t trench-a
```

Request a stream (Conduit/Network Service: lb-fe, Trench: trench-a)
```
./target-client request -ns lb-fe -t trench-a
```

Close a stream (Conduit/Network Service: load-balancer, Trench: trench-a)
```
./target-client close -ns lb-fe -t trench-a
```
