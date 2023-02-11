# Deploy

We will now deploy a new stateless-lb conduit in `trench-a` called `conduit-a-1`:
```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Conduit
metadata:
  name: conduit-a-1
  namespace: red
  labels:
    trench: trench-a
spec:
  type: stateless-lb
EOF
```{{exec}}

# Verify

The conduit is now deployed
```
kubectl get conduits -n red
```{{exec}}

When the conduit has been applied, the operator has deployed a new daemonset: 
* proxy-conduit-a-1: 
```
kubectl get -n red daemonsets
kubectl get -n red pods
```{{exec}}

This configmap has been reconfigured with the new configuration.
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-4.svg)
