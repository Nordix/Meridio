# Deploy

We will now deploy a new attractor in `trench-a` called `attractor-a-1`:
```
cat <<EOF | kubectl apply -f -
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
  - gateway-a-1-v4
  - gateway-a-1-v6
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
EOF
```{{exec}}
The attractor will serve conduit `conduit-a-1` from the 2 gateways `gateway-a-1-v4` and `gateway-a-1-v6` for the 2 VIPs `vip-a-1-v4` and `vip-a-1-v6`. It will receive an nsm-vlan interface based on eth0 with 100 as VLAN ID and will get an IPv4 in the `169.254.100.0/24` subnet and an IPv6 in the `100:100::/64` subnet. 2 replicas of this attractor will be deployed.

# Verify

The attractor is now deployed
```
kubectl get attractors -n red
```{{exec}}

When the attractor has been applied, the operator has deployed 2 new deployments:
* nse-vlan-attractor-a-1
* stateless-lb-frontend-attractor-a-1
```
kubectl get -n red deployments
kubectl get -n red pods
```{{exec}}

And a new PDB
* pdb-attractor-a-1: Pod disruption budget for the `stateless-lb-frontend-attractor-a-1` deployment
```
kubectl get -n red pdb
```{{exec}}

This configmap has been reconfigured with the new configuration.
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-5.svg)
