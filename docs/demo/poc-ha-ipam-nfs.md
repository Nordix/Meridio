# Demo - Kind - HA IPAM with NFS

This proof-of-concept (PoC) demonstrates a highly-available (HA) Meridio IPAM service in a Kubernetes environment. The primary goal is to validate the resilience and failover capabilities of the IPAM service when a node is abruptly removed.  

The demo uses a KinD Kubernetes cluster with one controller and six worker nodes. To provide a shared filesystem for the IPAM service, the controller node runs an [NFS Ganesha dynamic provisioner](https://github.com/kubernetes-sigs/nfs-ganesha-server-and-external-provisioner).  

Ganesha provides a standard NFS interface and serves data from a hostPath volume. This volume is mounted on the controller node and is used as the backing storage for the NFS exports.  
Note: For stability, the backing storage must be a non-`overlayfs` filesystem, such as `ext4`.  

The IPAM service is designed for high availability using a shared SQLite database and relies on the [leaderelection package](https://pkg.go.dev/k8s.io/client-go/tools/leaderelection) to ensure that only a single replica acts as the leader at any given time. This leader is responsible for managing the database and exposing the IPAM service to Meridio clients.  

To test the failover mechanism, two of the six KinD workers are dedicated to running the IPAM Pods. This allows for controlled testing of a sudden node removal, simulating a real-world failure scenario and observing how the remaining IPAM replica takes over leadership.

## Installation

### Kubernetes cluster

Deploy a Kind Kubernetes cluster (with support for local private docker registry)
```bash
#!/bin/bash

# You can set this variable before running the script, e.g.,
# export GANESHA_HOST_PATH="/my/custom/path"
# The script will use this value, or fall back to the default if not set.
GANESHA_HOST_PATH=${GANESHA_HOST_PATH:-"~/work/poc/nfs-meridio-ipam/ganesha-data/"}

# Check if the directory exists and create it if not.
if [ ! -d "$GANESHA_HOST_PATH" ]; then
    echo "Creating hostPath directory: $GANESHA_HOST_PATH"
    mkdir -p "$GANESHA_HOST_PATH"
fi

# Define the YAML configuration as a here-document
KIND_CONFIG=$(cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry-1.docker.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."gcr.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry.nordix.org"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."quay.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."ghcr.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."projects.registry.vmware.com"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry.k8s.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."cr.fluentbit.io"]
    endpoint = ["http://registry:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.configs."docker.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."registry-1.docker.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."k8s.gcr.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."gcr.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."registry.nordix.org".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."quay.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."ghcr.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."projects.registry.vmware.com".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."registry.k8s.io".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."cr.fluentbit.io".tls]
    insecure_skip_verify = true
nodes:
  - role: control-plane
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    extraMounts:
    - hostPath: ${GANESHA_HOST_PATH}
      containerPath: /mnt/ganesha-host-mount
  - role: worker
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          container-log-max-size: "100Mi"
  - role: worker
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          container-log-max-size: "100Mi"
  - role: worker
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          container-log-max-size: "100Mi"
  - role: worker
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          container-log-max-size: "100Mi"
  - role: worker
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          container-log-max-size: "100Mi"
          register-with-taints: "dedicated=ipam-worker:NoSchedule"
          node-labels: "ipam-worker=true"
  - role: worker
    image: kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865
    kubeadmConfigPatches:
    - |
      kind: JoinConfiguration
      nodeRegistration:
        kubeletExtraArgs:
          container-log-max-size: "100Mi"
          register-with-taints: "dedicated=ipam-worker:NoSchedule"
          node-labels: "ipam-worker=true"
networking:
  kubeProxyMode: "ipvs"
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
EOF
)

# Create a temporary file and write the config to it
TEMP_CONFIG_FILE=$(mktemp)
echo "${KIND_CONFIG}" > "${TEMP_CONFIG_FILE}"

# Run the kind command with the temporary file
kind create cluster --config "${TEMP_CONFIG_FILE}"

# Clean up the temporary file
rm "${TEMP_CONFIG_FILE}"
```

Optional; Connect local private registry to the KinD network
```sh
#!/bin/sh
set -o errexit

# Note: User must create a registry container unless it already exists
reg_name='registry'
reg_port='80'

# connect the registry to the cluster network
# (the network may already be connected)
docker network connect "kind" "${reg_name}" || true

# Document the local registry
# https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/generic/1755-communicating-a-local-registry
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${reg_port}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF
```

### Setup the KinD cluster

```bash
make -C ./docs/demo/scripts/kind/ KUBERNETES_WORKERS=4 KUBERNETES_VERSION=1.31 install-spire kind-network multus cni-plugins whereabouts kind-gateways
```

### Deploy Ganesha dynamic provisioner
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nfs-provisioner
  namespace: default # Or your preferred namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nfs-provisioner-runner
rules:
  - apiGroups: [""]
    resources: ["persistentvolumes"]
    verbs: ["get", "list", "watch", "create", "delete"]
  - apiGroups: [""]
    resources: ["persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "update"]
  - apiGroups: ["storage.k8s.io"]
    resources: ["storageclasses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "update", "patch"]
  - apiGroups: [""]
    resources: ["services", "endpoints"] # For the provisioner to manage its own service/endpoints
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["extensions", "apps"]
    resources: ["deployments"] # For the provisioner to manage its own deployment (if needed, though typically not for itself)
    verbs: ["get", "list", "watch", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: run-nfs-provisioner
subjects:
  - kind: ServiceAccount
    name: nfs-provisioner
    namespace: default # Match the ServiceAccount namespace
roleRef:
  kind: ClusterRole
  name: nfs-provisioner-runner
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: nfs-provisioner-leader-lock
  namespace: default # Match the ServiceAccount namespace
rules:
  - apiGroups: [""]
    resources: ["endpoints"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: nfs-provisioner-leader-lock
  namespace: default # Match the ServiceAccount namespace
subjects:
  - kind: ServiceAccount
    name: nfs-provisioner
    namespace: default # Match the ServiceAccount namespace
roleRef:
  kind: Role
  name: nfs-provisioner-leader-lock
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nfs-provisioner
  labels:
    app: nfs-provisioner
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nfs-provisioner
  strategy:
    type: Recreate # Use Recreate to ensure clean restarts for a single replica
  template:
    metadata:
      labels:
        app: nfs-provisioner
    spec:
      serviceAccountName: nfs-provisioner
      terminationGracePeriodSeconds: 10
      # Add nodeSelector and tolerations to schedule on the control plane
      nodeSelector:
        kubernetes.io/hostname: kind-control-plane
      tolerations:
      - key: "node-role.kubernetes.io/control-plane"
        operator: "Exists"
        effect: "NoSchedule"
      containers:
        - name: nfs-provisioner
          image: registry.k8s.io/sig-storage/nfs-provisioner:v4.0.8
          imagePullPolicy: IfNotPresent
          ports:
            # Standard NFS ports
            - name: nfs
              containerPort: 2049
            - name: nfs-udp
              containerPort: 2049
              protocol: UDP
            - name: nlockmgr # Added nlockmgr ports from official example
              containerPort: 32803
            - name: nlockmgr-udp
              containerPort: 32803
              protocol: UDP
            - name: mountd
              containerPort: 20048 # Common port for mountd
            - name: mountd-udp
              containerPort: 20048
              protocol: UDP
            - name: rquotad # Added rquotad ports from official example
              containerPort: 875
            - name: rquotad-udp
              containerPort: 875
              protocol: UDP
            - name: rpcbind
              containerPort: 111
            - name: rpcbind-udp
              containerPort: 111
              protocol: UDP
            - name: statd
              containerPort: 662
            - name: statd-udp
              containerPort: 662
              protocol: UDP
          securityContext:
            capabilities:
              add:
                - DAC_READ_SEARCH
                - SYS_RESOURCE
          args:
            - "-provisioner=example.com/nfs"
          env:
            - name: POD_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
            - name: SERVICE_NAME
              value: nfs-provisioner
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          volumeMounts:
            - name: export-volume
              mountPath: /export # This is where Ganesha will store the actual PV data
      volumes:
        - name: export-volume
          hostPath: # Using hostPath for persistence in KinD. For production, use a real PV.
            path: /mnt/ganesha-host-mount
            type: DirectoryOrCreate
---
apiVersion: v1
kind: Service
metadata:
  name: nfs-provisioner
  labels:
    app: nfs-provisioner
spec:
  selector:
    app: nfs-provisioner
  ports:
    - name: nfs
      port: 2049
      protocol: TCP
    - name: nfs-udp
      port: 2049
      protocol: UDP
    - name: nlockmgr # Added nlockmgr ports from official example
      port: 32803
    - name: nlockmgr-udp
      port: 32803
      protocol: UDP
    - name: mountd
      port: 20048
      protocol: TCP
    - name: mountd-udp
      port: 20048
      protocol: UDP
    - name: rquotad # Added rquotad ports from official example
      port: 875
    - name: rquotad-udp
      port: 875
      protocol: UDP
    - name: rpcbind
      port: 111
      protocol: TCP
    - name: rpcbind-udp
      port: 111
      protocol: UDP
    - name: statd
      port: 662
      protocol: TCP
    - name: statd-udp
      port: 662
      protocol: UDP
  type: ClusterIP # Standard ClusterIP for internal access
EOF
```

Create a Storage Class that will serve as a template for dynamic provisioning. When a POD requests a PersistentVolumeClaim (PVC) with the storageClassName `nfs-ganesha`, Kubernetes hands off the request to the provisioner `example.com/nfs` that is to Ganesha server to create the actual storage. Use `mountOptions` to control the NFS client's behavour.
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-ganesha
provisioner: example.com/nfs # Match the provisioner arg in Deployment
reclaimPolicy: Delete # PVs created by this SC will be deleted when PVC is deleted
volumeBindingMode: Immediate # PV is bound as soon as PVC is created
mountOptions:
  - vers=4.1
  - soft # avoid kind-worker hanging indefinitely when shutting down KinD cluster with running IPAM PODs
  - timeo=600 # 60 seconds
EOF
```

### NSM

```bash
helm install docs/demo/deployments/nsm-k8s --generate-name --create-namespace --namespace nsm --set pullPolicy=Always --set tag=v1.14.5
```

### Meridio

Build the POC images. A private local docker registry is used in this case.
```bash
make REGISTRY=localhost:80/cloud-native/meridio
```

Deploy Meridio Operator
```bash
make deploy REGISTRY=registry.nordix.org/cloud-native/meridio VERSION_OPERATOR=latest OPERATOR_NAMESPACE="default"
```

Define a PersistentVolumeClaim that requests a `ReadWriteMany` volume from the `nfs-ganesha` StorageClass. The Ganesha provisioner will automatically create the corresponding PersistentVolume and the NFS share.
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ipam-rwx-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteMany # This is key for RWX
  storageClassName: nfs-ganesha # Reference the StorageClass created by Ganesha
  resources:
    requests:
      storage: 20Mi # Request a small amount of storage for the PoC
EOF
```

Because the referenced Storage Class has `volumeBindingMode: Immediate` a PV must be bound as soon as PVC is created.
```
$ kubectl get pvc ipam-rwx-pvc
NAME           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   VOLUMEATTRIBUTESCLASS   AGE
ipam-rwx-pvc   Bound    pvc-d9353872-b677-472a-a568-4cd20883c201   20Mi       RWX            nfs-ganesha    <unset>                 36m
$ kubectl get pv |grep ipam-rwx-pvc
pvc-d9353872-b677-472a-a568-4cd20883c201   20Mi      RWX      Delete      Bound    default/ipam-rwx-pvc      nfs-ganesha  <unset>      38m
```

Install Meridio via Meridio Operator
```yaml
cat <<EOF | kubectl apply -f -
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: mynad
  namespace: default
spec:
  config: '{
      "cniVersion": "0.4.0",
      "name": "mynet",
      "plugins": [
        {
          "type":"bridge",
          "name": "mybridge",
          "bridge": "br-meridio",
          "vlan": 100,
          "ipam": {
            "enable_overlapping_ranges": false,
            "log_file": "/tmp/whereabouts.log",
            "type": "whereabouts",
            "ipRanges": [{
              "range": "169.254.100.0/24",
              "enable_overlapping_ranges": false,
              "exclude": [
                "169.254.100.150/32",
                "169.254.100.253/32",
                "169.254.100.254/32"
              ]
            }, {
              "range": "100:100::/64",
              "exclude": [
                "100:100::150/128"
              ]
            }]
          }
        }
      ]
  }'
---
apiVersion: meridio.nordix.org/v1
kind: Trench
metadata:
  name: trench-a
  namespace: default
  annotations:
    resource-template: "medium"
spec:
  ip-family: dualstack
---
apiVersion: meridio.nordix.org/v1
kind: Attractor
metadata:
  name: attr-a1
  namespace: default
  labels:
    trench: trench-a
spec:
  replicas: 2
  gateways:
    - gateway-a1
    - gateway-a2
  composites:
    - load-balancer-a1
  vips:
    - vip-a1
    - vip-a2
    - vip-a3
  interface:
    name: eth0.100
    type: network-attachment
    network-attachments:
      - name: mynad
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  labels:
    trench: trench-a
  name: gateway-a1
  namespace: default
spec:
  address: 169.254.100.150
  protocol: bgp
  bgp:
    bfd:
      switch: true
      min-rx: 300ms
      min-tx: 300ms
      multiplier: 3
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: "3s"
    local-port: 10179
    remote-port: 10179
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  labels:
    trench: trench-a
  name: gateway-a2
  namespace: default
spec:
  address: 100:100::150
  protocol: bgp
  bgp:
    bfd:
      switch: true
      min-rx: 300ms
      min-tx: 300ms
      multiplier: 3
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: "3s"
    local-port: 10179
    remote-port: 10179
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  labels:
    trench: trench-a
  name: vip-a1
  namespace: default
spec:
  address: "20.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  labels:
    trench: trench-a
  name: vip-a2
  namespace: default
spec:
  address: "10.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  labels:
    trench: trench-a
  name: vip-a3
  namespace: default
spec:
  address: "2000::1/128"
---
apiVersion: meridio.nordix.org/v1
kind: Conduit
metadata:
  name: load-balancer-a1
  namespace: default
  annotations:
    resource-template: "medium"
  labels:
    trench: trench-a
spec:
  type: stateless-lb
---
apiVersion: meridio.nordix.org/v1
kind: Stream
metadata:
  name: stream-a1
  namespace: default
  labels:
    trench: trench-a
spec:
  conduit: load-balancer-a1
  max-targets: 145
---
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a1
  namespace: default
  labels:
    trench: trench-a
spec:
  stream: stream-a1
  priority: 2
  vips:
  - vip-a1
  - vip-a2
  - vip-a3
  source-subnets:
  - 0.0.0.0/0
  - ::/0
  source-ports:
  - any
  destination-ports:
  - 500-8000
  protocols:
  - udp
  - tcp
---
apiVersion: meridio.nordix.org/v1
kind: Stream
metadata:
  name: stream-a2
  namespace: default
  labels:
    trench: trench-a
spec:
  conduit: load-balancer-a1
---
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a2
  namespace: default
  labels:
    trench: trench-a
spec:
  stream: stream-a2
  priority: 2
  vips:
  - vip-a1
  - vip-a2
  - vip-a3
  source-subnets:
  - 0.0.0.0/0
  - ::/0
  source-ports:
  - any
  destination-ports:
  - 9000-10000
  protocols:
  - udp
  - tcp
EOF
```

A StatefulSet will not create the next POD in its sequence until the previous POD has reached a Ready state. In this POC only the leader IPAM POD is passing readiness probe. Hence, using more than two IPAM replicas only makes sense to be protected against persisting node failures.
```
kubectl get pods -o wide
NAME                                            READY   STATUS    RESTARTS   AGE    IP           NODE                 NOMINATED NODE   READINNAME                                            READY   STATUS    RESTARTS   AGE     IP           NODE                 NOMINATED NODE   READINESS GATES
ipam-trench-a-0                                 1/1     Running   0          6m26s   10.244.4.2   kind-worker6         <none>           <none>
ipam-trench-a-1                                 0/1     Running   0          6m15s   10.244.1.2   kind-worker5         <none>           <none>
meridio-operator-54f9c6765f-x9xsm               1/1     Running   0          13m     10.244.2.3   kind-worker          <none>           <none>
nfs-provisioner-59db44d7ff-jvj9g                1/1     Running   0          16m     10.244.0.5   kind-control-plane   <none>           <none>
nsp-trench-a-0                                  1/1     Running   0          6m26s   10.244.6.7   kind-worker4         <none>           <none>
proxy-load-balancer-a1-qr7wv                    1/1     Running   0          6m26s   10.244.5.4   kind-worker3         <none>           <none>
proxy-load-balancer-a1-vnld7                    1/1     Running   0          6m26s   10.244.3.4   kind-worker2         <none>           <none>
proxy-load-balancer-a1-z5j6t                    1/1     Running   0          6m26s   10.244.6.5   kind-worker4         <none>           <none>
proxy-load-balancer-a1-zwhsx                    1/1     Running   0          6m26s   10.244.2.5   kind-worker          <none>           <none>
stateless-lb-frontend-attr-a1-5c747c4fd-4qgbb   2/2     Running   0          6m26s   10.244.2.4   kind-worker          <none>           <none>
stateless-lb-frontend-attr-a1-5c747c4fd-ggcjp   2/2     Running   0          6m26s   10.244.5.3   kind-worker3         <none>           <none>
target-a-6fc4d776b7-9m9g9                       2/2     Running   0          5m35s   10.244.5.5   kind-worker3         <none>           <none>
target-a-6fc4d776b7-gcd5f                       2/2     Running   0          5m35s   10.244.2.6   kind-worker          <none>           <none>
target-a-6fc4d776b7-k9qtc                       2/2     Running   0          5m35s   10.244.3.5   kind-worker2         <none>           <none>
target-a-6fc4d776b7-m4v8s                       2/2     Running   0          5m35s   10.244.6.8   kind-worker4         <none>           <none>

```

### Target

Install targets
```
helm install examples/target/deployments/helm/ --generate-name --create-namespace --set applicationName=target-a     --set default.trench.name=trench-a --set default.conduit.name=load-balancer-a1 --set default.stream.name=stream-a1 --set pullPolicy=Always
```

## Traffic

Connect to a external host (trench-a)
```
docker exec -it trench-a bash
```

Generate traffic
```
# ipv4
ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
# ipv6
ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v6traffic.json
```

Verification
```
ctraffic -analyze hosts -stat_file v4traffic.json
ctraffic -analyze hosts -stat_file v6traffic.json
```
## Failover Test

Start ctraffic for a longer period of time
```
ctraffic -timeout 5m -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
```

Stop the KinD worker running the active IPAM replica
```
docker container stop kind-worker6
```

The standby IPAM replica should take over the leader role shortly
```
kubectl get pods -o wide
NAME                                            READY   STATUS    RESTARTS   AGE     IP           NODE                 NOMINATED NODE   READINESS GATES
ipam-trench-a-0                                 1/1     Running   0          6m53s   10.244.4.2   kind-worker6         <none>           <none>
ipam-trench-a-1                                 1/1     Running   0          6m42s   10.244.1.2   kind-worker5         <none>           <none>
meridio-operator-54f9c6765f-x9xsm               1/1     Running   0          13m     10.244.2.3   kind-worker          <none>           <none>
nfs-provisioner-59db44d7ff-jvj9g                1/1     Running   0          17m     10.244.0.5   kind-control-plane   <none>           <none>
nsp-trench-a-0                                  1/1     Running   0          6m53s   10.244.6.7   kind-worker4         <none>           <none>
proxy-load-balancer-a1-qr7wv                    1/1     Running   0          6m53s   10.244.5.4   kind-worker3         <none>           <none>
proxy-load-balancer-a1-vnld7                    1/1     Running   0          6m53s   10.244.3.4   kind-worker2         <none>           <none>
proxy-load-balancer-a1-z5j6t                    1/1     Running   0          6m53s   10.244.6.5   kind-worker4         <none>           <none>
proxy-load-balancer-a1-zwhsx                    1/1     Running   0          6m53s   10.244.2.5   kind-worker          <none>           <none>
stateless-lb-frontend-attr-a1-5c747c4fd-4qgbb   2/2     Running   0          6m53s   10.244.2.4   kind-worker          <none>           <none>
stateless-lb-frontend-attr-a1-5c747c4fd-ggcjp   2/2     Running   0          6m53s   10.244.5.3   kind-worker3         <none>           <none>
target-a-6fc4d776b7-9m9g9                       2/2     Running   0          6m2s    10.244.5.5   kind-worker3         <none>           <none>
target-a-6fc4d776b7-gcd5f                       2/2     Running   0          6m2s    10.244.2.6   kind-worker          <none>           <none>
target-a-6fc4d776b7-k9qtc                       2/2     Running   0          6m2s    10.244.3.5   kind-worker2         <none>           <none>
target-a-6fc4d776b7-m4v8s                       2/2     Running   0          6m2s    10.244.6.8   kind-worker4         <none>           <none>

kubectl get endpoints ipam-service-trench-a
NAME                    ENDPOINTS         AGE
ipam-service-trench-a   10.244.1.2:7777   6m51s

kubectl get nodes
NAME                 STATUS     ROLES           AGE   VERSION
kind-control-plane   Ready      control-plane   29m   v1.31.0
kind-worker          Ready      <none>          29m   v1.31.0
kind-worker2         Ready      <none>          29m   v1.31.0
kind-worker3         Ready      <none>          29m   v1.31.0
kind-worker4         Ready      <none>          29m   v1.31.0
kind-worker5         Ready      <none>          29m   v1.31.0
kind-worker6         NotReady   <none>          29m   v1.31.0
```

Verification
```
ctraffic -analyze hosts -stat_file v4traffic.json
```

## Clean-up

Delete the Trench before deleting the KinD cluster
```
kubectl delete trench trench-a
```