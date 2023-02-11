# Deploy

We will now deploy a new flow in `trench-a` called `flow-a-z-tcp`:
```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a-z-tcp
  namespace: red
  labels:
    trench: trench-a
spec:
  stream: stream-a-i
  priority: 1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  source-subnets:
  - 0.0.0.0/0
  - 0:0:0:0:0:0:0:0/0
  source-ports:
  - any
  destination-ports:
  - "4000"
  protocols:
  - tcp
EOF
```{{exec}}
This flow will be serving in `stream-a-i`. It will accept only TCP, any source IP and source port, only `vip-a-1-v4` and `vip-a-1-v6` as destination IP and only `4000` as destination port.

# Verify

The flow is now deployed
```
kubectl get flows -n red
```{{exec}}

No new resource has been deployed while deploying the Flow, but the configmap has been configured
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-8.svg)
