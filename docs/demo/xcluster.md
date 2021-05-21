# Demo - xcluster - vlan

* [Kind - VLAN](readme.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a vlan-forwarder to link the network service to an external host.
* [Kind - Static](static.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a script to link the network service to an external host.
* [xcluster - VLAN](xcluster.md) - Demo running on [xcluster](https://github.com/Nordix/xcluster) using a vlan-forwarder to link the network service to an external host.
* [Kind - VLAN - Multi-trenches](multi-trenches.md) - Demo running 2 trenches on [Kind](https://kind.sigs.k8s.io/) using vlan-forwarders to link the network services to an external host.


## Installation

### Kubernetes cluster

Deploy a Kubernetes cluster with xcluster
```
xc mkcdrom; xc starts --nets_vm=0,1,2 --nvm=2
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
helm install deployments/helm/ --generate-name --set vlanInterface=eth1
# ipv6
helm install deployments/helm/ --generate-name --set ipFamily=ipv6 
# dualstack
helm install deployments/helm/ --generate-name --set ipFamily=dualstack 
```

### External host / External connectivity

Deploy a external host
```
./docs/demo/scripts/xcluster/external-host.sh
```

## Traffic

Connect to the external host
```
ssh root@192.168.0.202
# or
vm 202
# or
ssh root@localhost -p 12502
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
```
