# Demo - Kind - vlan

* [Kind - VLAN](readme.md) - Demo running on [Kind](https://kind.sigs.k8s.io/) using a vlan-forwarder to link the network service to an external host.
* [xcluster - VLAN](xcluster.md) - Demo running on [xcluster](https://github.com/Nordix/xcluster) using a vlan-forwarder to link the network service to an external host.


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

Configure Spire for trenches in namespace red
```
./docs/demo/scripts/spire.sh default red
```

Configure Spire for trench-a
```
./docs/demo/scripts/spire.sh meridio-trench-a red
```

Install Meridio trench-a
```
# ipv4
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-a --set vlan.id=100 --set --set vlan.fe.gateway[0]="169.254.100.150/24"
# ipv6
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-a --set vlan.id=100 --set ipFamily=ipv6 --set vlan.fe.gateway[0]="100:100::150/64"
# dualstack
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-a --set ipFamily=dualstack ---set vlan.fe.gateway[0]="169.254.100.150/24" --set vlan.fe.gateway[1]="100:100::150/64"
```

Configure Spire for trench-b
```
./docs/demo/scripts/spire.sh meridio-trench-b red
```

Install Meridio trench-b
```
# ipv4
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-b --set vlan.id=101 --set vlan.fe.gateway[0]="169.254.100.150/24"
# ipv6
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-b --set vlan.id=101 --set ipFamily=ipv6 --set vlan.fe.gateway[0]="100:100::150/64"
# dualstack
helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-b --set vlan.id=101 --set ipFamily=dualstack --set vlan.fe.gateway[0]="169.254.100.150/24" --set vlan.fe.gateway[1]="100:100::150/64"
```

### target

Deploy common resources for the targets
```
helm install examples/target/common/ --generate-name --create-namespace --namespace red
```

Configure Spire for the targets
```
./docs/demo/scripts/spire.sh meridio red
```

Install targets connected to trench-a
```
helm install examples/target/helm/ --generate-name --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a
```

Install targets connected to trench-b
```
helm install examples/target/helm/ --generate-name --create-namespace --namespace red --set applicationName=target-b --set default.trench.name=trench-b
```

### External host / External connectivity

Deploy a external host
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

## Scaling

Scale-In/Scale-Out target
```
kubectl scale deployment target -n red --replicas=5
```

Scale-In/Scale-Out load-balancer
```
kubectl scale deployment load-balancer-trench-a -n red --replicas=1
```

## Ambassador

From a target:

Connect to a conduit/trench (Conduit/Network Service: load-balancer, Trench: trench-a)
```
./target-client connect -ns load-balancer -t trench-a
```

Disconnect from a conduit/trench (Conduit/Network Service: load-balancer, Trench: trench-a)
```
./target-client disconnect -ns load-balancer -t trench-a
```

Request a stream (Conduit/Network Service: load-balancer, Trench: trench-a)
```
./target-client request -ns load-balancer -t trench-a
```

Close a stream (Conduit/Network Service: load-balancer, Trench: trench-a)
```
./target-client close -ns load-balancer -t trench-a
```
