# Demo

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
helm install docs/demo/deployments/nsm/ --generate-name
```

### Meridio

Install Meridio
```
helm install deployments/helm/ --generate-name
```

### Demo environment

Deploy a external host
```
./docs/demo/scripts/external-host.sh
```

Add Routes to the load balancer service
```
docker exec -it ubuntu-ext ip route replace 20.0.0.1/32 nexthop via 192.168.1.1 nexthop via 192.168.2.1
```

Attach LBs to the external host
```
./docs/demo/scripts/external-link.sh
```

## Traffic

Connect to the external host
```
docker exec -it ubuntu-ext bash
```

Generate traffic
```
ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
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
```
