# Meridio-Operator

[Meridio](https://github.com/Nordix/Meridio)

## Build

```bash
# build the image
# use IMG to specify the image tag, default is "controller:0.0.1"
make docker-build IMG="<meridio-operator-tag>"

# change:
# - image tag by setting IMG
# - builder image by setting BUILDER, default is "golang:1.16"
# - base image by setting BASE_IMG, default is "ubuntu:18.04"
make docker-build IMG="<meridio-operator-tag>" BUILDER="<builder>" BASE_IMG="<base-image>"

# Make the image available for the cluster to use. The following commands are alternative
make docker-push IMG="<meridio-operator-tag>"  # Push the image to a registry
make kind-load IMG="<meridio-operator-tag>"  # Load the image to kind cluster
```

### Run tests

The e2e tests expect the meridio operator is already running in the cluster.
The e2e test suite can be run in an arbitrary namespace by specifying NAMESPACE
as the following example shows.

```bash
# If meridio operator is not deployed yet, run the following commented command first:
# make deploy NAMESPACE="red"
make e2e NAMESPACE="red"
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

**The meridio operator is deployed in the "default" namespace in the examples below.**

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
...
```

### Attractor

An attractor resource is responsible for the creation of NSE-VLAN. The resources created will be suffixed with either attractor or trench's name.

To be noted, meridio-operator currently have a limitation to have one attractor each trench.

> There are some resources needs to be created with labels. The label referred resources are always need to be created **before** the resource itself. Otherwise these resource will fail to be created.
> **The labels in the resources are immutable**.

An attractor is a resource needs to be created with label. `metadata.labels.trench` specifies the owner trench of the attractor.

To see how to configure and read the status of an attractor, please refer to [attractor spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#AttractorSpec) and [attractor status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#AttractorStatus).

```bash
kubectl apply -f ./config/samples/meridio_v1alpha1_attractor.yaml

#After applying, you should able to see the following instances added in the cluster

$ kubectl get all
NAME                                  READY   STATUS    RESTARTS   AGE
pod/nse-vlan-attr1-7844574dc-dlgkr    1/1     Running   0          5s

NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/nse-vlan-attr1   1/1     1            1           5s

$ kubectl get attractor
NAME    INTERFACE-NAME   INTERFACE-TYPE   GATEWAYS                  VIPS              TRENCH
attr1   eth0.100         nsm-vlan         ["gateway1","gateway2"]   ["vip1","vip2"]   trench-a
```

### Gateway

A gateway is a resource to describe the gateway information for the Meridio Front-ends.It must be created with label `metadata.labels.trench` to specify its owner reference trench.

To see how to configure and read the status of a gateway, please refer to [gateway spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#GatewaySpec) and [gateway status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#GatewayStatus).

In the example below, two gateways will be created.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_gateway.yaml

#The following resources should be found

$ kubectl get gateways
NAME       ADDRESS   PROTOCOL   TRENCH
gateway1   2.3.4.5   bgp        trench-a
gateway2   1000::1   bgp        trench-a
```

### Vip

A Vip is a resource to reserving the destination addresses for the target applications.It must be created with label `metadata.labels.trench` to specify its owner reference trench.

To see how to configure and read the status of a Vip, please refer to [Vip spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#VipSpec) and [Vip status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#VipStatus).

In the example below, two Vips will be created.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_vip.yaml

# The following resources should be found

$ kubectl get vip
NAME   ADDRESS       TRENCH
vip1   20.0.0.0/32   trench-a
vip2   10.0.0.1/32   trench-a
```

### Conduit

A Conduit is for configuring the load balancer type. It must be created with label `metadata.labels.trench` to specify its owner reference trench.
There is a limitation that a conduit must be created when one attractor is created in the same trench. Meridio only supports one conduit per trench now.

To see how to configure and read the status of a Conduit, please refer to [Conduit spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#ConduitSpec) and [Conduit status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#ConduitStatus).

A Conduit can be created by following the example below.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_conduit.yaml

# The following resources should be found
$ kubectl get all
NAME                                                       READY   STATUS    RESTARTS   AGE
...
pod/lb-fe-conduit-stateless-f6774f9b8-4rbst                3/3     Running   0          6m2s
...

NAME                                                  READY   UP-TO-DATE   AVAILABLE   AGE
...
deployment.apps/lb-fe-conduit-stateless               1/1     1            1           6m2s
...

$ kubectl get conduit
NAME                TYPE           TRENCH
conduit-stateless   stateless-lb   trench-a
```

### Stream

A Stream is for grouping different flows, and it can choose how traffic is load balanced by registering for a specific conduit. It must be created with label `metadata.labels.trench` to specify its owner reference trench.

To see how to configure and read the status of a Stream, please refer to [Stream spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#StreamSpec) and [Stream status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#StreamStatus).

A Stream can be created by following the example below.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_stream.yaml

# The following resources should be found

$ kubectl get stream
NAME       CONDUIT             TRENCH
stream-1   conduit-stateless   trench-a
```

### Flow

A Flow enables the traffic to a selection of pods by specifying the 5-tuples and the Stream the traffic go through. It must be created with label `metadata.labels.trench` to specify its owner reference trench.

To see how to configure and read the status of a Flow, please refer to [Flow spec](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#FlowSpec) and [Flow status](https://pkg.go.dev/github.com/nordix/meridio-operator/api/v1alpha1#FlowStatus).

A Flow can be created by following the example below.

```bash
$ kubectl apply -f ./config/samples/meridio_v1alpha1_flow.yaml

# The following resources should be found

$ kubectl get flow
NAME     VIPS       DST-PORTS       SRC-SUBNETS          SRC-PORTS         PROTOCOLS   STREAM     TRENCH
flow-1   ["vip1"]   ["2000-3000"]   ["10.20.30.40/30"]   ["20000-21000"]   ["tcp"]     stream-1   trench-a
```
