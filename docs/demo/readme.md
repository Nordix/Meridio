# Demo - Kind - vlan

* [Kind - VLAN](readme.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a vlan-forwarder to link the network service to an external host.
* [xcluster - VLAN](xcluster.md) - Demo running on [xcluster](https://github.com/Nordix/xcluster) using a vlan-forwarder to link the network service to an external host.

This demo deploys a Kubernetes with 2 workers and 1 master running Spire, Network Service Mesh and a single Meridio trench. The Meridio trench has 1 conduit (Stateless load-balancer) with 2 instances, 1 stream, 1 flow (any source IP/Port, TCP, 5000 as destination port and 2 VIP: 20.0.0.1/32 and 2000::1/128). The traffic is attracted by a vlan connected to a gateway (also used as traffic generator).

![Overview](../resources/Overview.svg)

## Installation

### Kubernetes cluster

Deploy a Kubernetes cluster with Kind
```
kind create cluster --config docs/demo/kind.yaml
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

### Meridio

Install Meridio
```
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-a --set ipFamily=dualstack
```

### Target

Install targets
```
helm install examples/target/helm/ --generate-name --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a
```

### External host / External connectivity

Deploy a external host (Gateway-Router)
```
./docs/demo/scripts/kind/external-host.sh
```

## Traffic

Connect to a external host (trench-a, trench-b or trench-c)
```
docker exec -it trench-a bash
```

Generate traffic
```
# ipv4
ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
# ipv6
ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
```

Verification
```
ctraffic -analyze hosts -stat_file v4traffic.json
```
