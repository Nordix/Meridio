# Demo - Kind - vlan - multi-trenches

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

Configure Spire
```
./docs/demo/scripts/spire-config-trenches.sh
```

Install Meridio trench-a
```
# ipv4
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-a --set vlan.id=100 --set vlan.ipv4Prefix=169.254.100.0/24
# ipv6
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-a --set vlan.id=100 --set vlan.ipv6Prefix=100:100::/64 --set ipFamily=ipv6 
# dualstack
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-a --set vlan.id=100 --set vlan.ipv4Prefix=169.254.100.0/24 --set vlan.ipv6Prefix=100:100::/64 --set ipFamily=dualstack 
```

Install Meridio trench-b
```
# ipv4
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-b --set vlan.id=101 --set vlan.ipv4Prefix=169.254.101.0/24
# ipv6
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-b --set vlan.id=101 --set vlan.ipv6Prefix=100:101::/64 --set ipFamily=ipv6 
# dualstack
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-b --set vlan.id=101 --set vlan.ipv4Prefix=169.254.101.0/24 --set vlan.ipv6Prefix=100:101::/64 --set ipFamily=dualstack 
```

### External host / External connectivity

Deploy a external host
```
./docs/demo/scripts/vlan/external-host-ns.sh
```

## Traffic

Connect to the external host
```
docker exec -it ubuntu-ext bash
```

### Trench-a

Generate traffic on trench-a
```
# ipv4
ip netns exec trench-a ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic-a.json
# ipv6
ip netns exec trench-a ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic-a.json
```

Verification of trench-a
```
ctraffic -analyze hosts -stat_file v4traffic-a.json
```

Generate traffic on trench-b
```
# ipv4
ip netns exec trench-b ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic-b.json
# ipv6
ip netns exec trench-b ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic-b.json
```

Verification of trench-b
```
ctraffic -analyze hosts -stat_file v4traffic-b.json
```

## Scaling

Scale-In/Scale-Out target of trench-a
```
kubectl scale deployment target -n trench-a --replicas=5
```

Scale-In/Scale-Out load-balancer of trench-a
```
kubectl scale deployment load-balancer -n trench-a --replicas=1
```
