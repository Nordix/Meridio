# Demo - Kind - vlan

* [Kind - VLAN](readme.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a vlan-forwarder to link the network service to an external host.
* [Kind - Static](static.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a script to link the network service to an external host.
* [xcluster - VLAN](xcluster.md) - Demo running on [xcluster](https://github.com/Nordix/xcluster) using a vlan-forwarder to link the network service to an external host.
* [Kind - VLAN - Multi-trenches](multi-trenches.md) - Demo running 2 trenches on [Kind](https://kind.sigs.k8s.io/) using vlan-forwarders to link the network services to an external host.


## Installation

### Kubernetes cluster

Deploy a Kubernetes cluster with Kind
```
kind create cluster --config docs/demo/kind.yaml
```

### NSM

Deploy Spire
```
helm install docs/demo/deployments/spire/ --generate-name
```

Configure Spire
```
./docs/demo/scripts/spire-config.sh
```

Deploy NSM
```
helm install docs/demo/deployments/nsm-vlan/ --generate-name
```

### Meridio

Install Meridio
```
# ipv4
helm install deployments/helm/ --generate-name
# ipv6
helm install deployments/helm/ --generate-name --set ipFamily=ipv6 
# dualstack
helm install deployments/helm/ --generate-name --set ipFamily=dualstack 
```

### External host / External connectivity

Deploy a external host
```
./docs/demo/scripts/vlan/external-host.sh
```

## Traffic

Connect to the external host
```
docker exec -it ubuntu-ext bash
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

## Scaling

Scale-In/Scale-Out target
```
kubectl scale deployment target --replicas=5
```

Scale-In/Scale-Out load-balancer
```
kubectl scale deployment load-balancer --replicas=1

# TODO: reconfigure links between the external host and the LBs

# ipv4
docker exec -it ubuntu-ext ip route replace 20.0.0.1/32 nexthop via 192.168.1.1 nexthop via 192.168.2.1
# ipv6
docker exec -it ubuntu-ext ip route replace 2000::1/128 nexthop via 1500:1::1 nexthop via 1500:2::1
```
