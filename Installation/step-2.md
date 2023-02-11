# Deploy

We will now deploy a new trench `trench-a` accepting dualstack:
```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Trench
metadata:
  name: trench-a
  namespace: red
spec:
  ip-family: dualstack
EOF
```{{exec}}

# Verify

The trench is now deployed
```
kubectl get trench -n red
```{{exec}}

When the trench has been applied, the operator has deployed:

2 Services:
* ipam-service-trench-a
* nsp-service-trench-a
```
kubectl get -n red service
```{{exec}}

2 statefulsets with 1 replica in each
* ipam-trench-a
* nsp-trench-a
```
kubectl get -n red statefulsets
kubectl get -n red pods
```{{exec}}

1 configmap
* meridio-configuration-trench-a: Contains all configurations for the custom resources in `trench-a`
```
kubectl get -n red configmap
```{{exec}}

This configmap has been configured with the current trench configuration.
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-2.svg)
