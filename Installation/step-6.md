# Deploy

We will now deploy 2 new gateways in `trench-a`, `gateway-a-1-v4` for IPv4 and `gateway-a-1-v6` for IPv6:
```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-a-1-v4
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
  name: gateway-a-1-v6
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
EOF
```{{exec}}
The 2 gateways are using GBP + BFD as routing protocol.

# Verify

The gateways are now deployed
```
kubectl get gateways -n red
```{{exec}}

No new resource has been deployed while deploying the Gateways, but the configmap has been configured
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-6.svg)
