# Meridio-Operator

[Meridio](https://github.com/Nordix/Meridio)

Run tests

The e2e tests expect the meridio operator is already running in the cluster. 
The e2e test suite can be run in an arbitrary namespace by specifying NAMESPACE=<desired-namespace>
as the following example shows.

```bash
# If meridio operator is not deployed yet, run the following commented command first:
# make deploy NAMESPACE="red"
make e2e NAMESPACE="red"
```

Build image

```bash
# build the image
make docker-build IMG="localhost:5000/meridio/meridio-operator:v0.0.1"

# change:
# - image tag by setting IMG
# - builder image by setting BUILDER
# - base image by setting BASE_IMG
make docker-build IMG="localhost:5000/meridio/meridio-operator:v0.0.1" BUILDER="golang:1.16" BASE_IMG="ubuntu:18.04"

# Push the image to a registry
make docker-push
```

## Deploy

### Deploy cert manager

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

### Deploy Meridio-Operator

Meridio-Operator is a namespace-scoped operator, which watches and manages custom resources in a single namespace where the operator is deployed.

```bash
make deploy \
IMG="localhost:5000/meridio/meridio-operator:v0.0.1" \ # If the image is built with a specific tag
NAMESPACE="default" # specifies the namespace where the operator will be deployed, "meridio-operator-system" is used by default
```

## Configuration

The meridio operator is deployed in the "default" namespace in the examples below.

### Trench

A [trench](https://github.com/Nordix/Meridio-Operator/blob/master/config/samples/meridio_v1alpha1_trench.yaml) spawns the IPAM, NSP, Proxy pods, and needed role, role-binding and service accounts. The resources created by a trench will be suffixed with the trench's name.

To see how to configure a trench, please refer to [trench spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#TrenchSpec).

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_trench.yaml

#After applying, you should able to see the following instances in the cluster

$ kubectl get trench
NAME                                 IP-FAMILY
trench.meridio.nordix.org/trench-a   dualstack

$ kubectl get all
NAME                                  READY   STATUS        RESTARTS   AGE
pod/ipam-trench-a-67474f4bf8-nt964    1/1     Running       0          1m
pod/nsp-trench-a-799984cfb5-r4g8f     1/1     Running       0          1m
pod/proxy-trench-a-dlc5c              1/1     Running       0          1m

NAME                            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/ipam-service-trench-a   ClusterIP   10.96.6.85     <none>        7777/TCP   1m
service/nsp-service-trench-a    ClusterIP   10.96.71.187   <none>        7778/TCP   1m

NAME                              DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/proxy-trench-a     1         1         1       1            1           <none>          1m

NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/ipam-trench-a   1/1     1            1           1m
deployment.apps/nsp-trench-a    1/1     1            1           1m
```

### Attractor

An attractor resource is responsible for  the creation of LB-FE, NSE-VLAN. The resources created will be suffixed with either attractor or trench's name.

To be noted, meridio-operator currently have a limitation to have one attractor each trench.

> There are some resources needs to be created with labels. The label referred resources are always need to be created **before** the resource itself. There is currently a limitation that if the resource itself is created first, the system may not function as expected, and will not recover if the labeled resources are created afterwards.
> **The labels in the resources are immutable**.

An attractor is a resource needs to be created with label. `metadata.labels.trench` specifies the owner trench of the attractor.

To see how to configure and read the status of an attractor, please refer to [attractor spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#AttractorSpec) and [attractor status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#AttractorStatus).

```bash
kubectl apply -f ./config/samples/meridio_v1alpha1_attractor.yaml

#After applying, you should able to see the following instances added in the cluster

$ kubectl get all
NAME                                  READY   STATUS    RESTARTS   AGE
pod/lb-fe-trench-a-786c5979b8-8vdjh   3/3     Running   0          5s
pod/nse-vlan-attr1-7844574dc-dlgkr    1/1     Running   0          5s

NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/lb-fe-trench-a   1/1     1            1           5s
deployment.apps/nse-vlan-attr1   1/1     1            1           5s

$ kubectl get attractor
NAME                                 VLANID   VLANITF   GATEWAYS                  GW-IN-USE   VIPS              VIPS-IN-USE   TRENCH     LB-FE
attractor.meridio.nordix.org/attr1   100      eth0      ["gateway1","gateway3"]               ["vip1","vip2"]                 trench-a   engaged

```

### Gateway

A gateway is a resource that must be created with label. It specifies its owner reference attractor by `metadata.labels.attractor`.

To see how to configure and read the status of a gateway, please refer to [gateway spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#GatewaySpec) and [gateway status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#GatewayStatus).

In the example below, two gateways will be created.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_gateway.yaml

#The following resources should be found

$ kubectl get gateways
NAME                                  ADDRESS   PROTOCOL   BFD     ATTRACTOR   STATUS    MESSAGE
gateway.meridio.nordix.org/gateway1   2.3.4.5   bgp        false   attr1       engaged
gateway.meridio.nordix.org/gateway2   1000::1   bgp        false   attr1       engaged

# The *GW-IN-USE* column of the attractor should be updated with the existing expected gateways. Shown as below

$ kubectl get attractor
NAME                                 VLANID   VLANITF   GATEWAYS                  GW-IN-USE                 VIPS              VIPS-IN-USE   TRENCH     LB-FE
attractor.meridio.nordix.org/attr1   100      eth0      ["gateway1","gateway2"]   ["gateway1","gateway2"]   ["vip1","vip2"]                 trench-a   engaged
```

### Vip

A vip is a resource that must be created with the label `metadata.labels.trench`, by which specifies its owner trench.

To see how to configure and read the status of a vip, please refer to [vip spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#VipSpec) and [vip status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#VipStatus).

In the example below, two vips will be created.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_vip.yaml

# The following resources should be found

$ kubectl get vip
NAME                          ADDRESS       STATUS
vip.meridio.nordix.org/vip1   20.0.0.1/32   engaged
vip.meridio.nordix.org/vip2   10.0.0.1/32   engaged

# The status *VIPS-IN-USE* of gateway will also be updated with the existing expected vips.

$ kubectl get attractor
NAME                                 VLANID   VLANITF   GATEWAYS                  GW-IN-USE                 VIPS              VIPS-IN-USE       TRENCH     LB-FE
attractor.meridio.nordix.org/attr1   100      eth0      ["gateway1","gateway2"]   ["gateway1","gateway2"]   ["vip1","vip2"]   ["vip1","vip2"]   trench-a   engaged
```
