# Deploy

We will now deploy a new stream in `trench-a` called `stream-a-i`:
```
cat <<EOF | kubectl apply -f -
---
apiVersion: meridio.nordix.org/v1
kind: Stream
metadata:
  name: stream-a-i
  namespace: red
  labels:
    trench: trench-a
spec:
  conduit: conduit-a-1
EOF
```{{exec}}
This stream will be serving in `conduit-a-1`.

# Verify

The vips is now deployed
```
kubectl get streams -n red
```{{exec}}

No new resource has been deployed while deploying the Stream, but the configmap has been configured
```
kubectl get configmap -n red meridio-configuration-trench-a -o yaml
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-7.svg)
