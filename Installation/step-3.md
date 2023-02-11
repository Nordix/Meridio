# Deploy

We will now deploy 2 new VIPs in `trench-a`, `vip-a-1-v4` for IPv4 (`20.0.0.1/32`) and `vip-a-1-v6` for IPv6 (`2000::1/128`):
```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  name: vip-a-1-v4
  namespace: red
  labels:
    trench: trench-a
spec:
  address: "20.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  name: vip-a-1-v6
  namespace: red
  labels:
    trench: trench-a
spec:
  address: "2000::1/128"
EOF
```{{exec}}

# Verify

The vips are now deployed
```
kubectl get vips -n red
```{{exec}}

No new resource has been deployed while deploying the VIPs, but the configmap has been configured
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-3.svg)
