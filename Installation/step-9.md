# Deploy

```
helm install meridio-target-a https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-Target-v1.0.0.tgz --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a --set default.conduit.name=conduit-a-1 --set default.stream.name=stream-a-i --set registry="registry.gitlab.com" --set repository="lionelj/meridio"
```{{exec}}

# Verify

They are now deployed:
* target-a
```
kubectl get deployment -n red 
kubectl get pods -n red
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-9.svg)
