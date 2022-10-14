# Meridio-Operator

[Meridio](https://github.com/Nordix/Meridio)

## Build

The following parameters can be configured when building the operator image:
* IMG: specify the image tag, default is "controller:0.0.1"
* BUILDER: specifiy the image used in the binary compling phase, default is "golang:1.16"
* BASE_IMG: the final base image, default is "ubuntu:18.04"

```bash
# build the image
make docker-build
# Make the image available for the cluster to use. The following commands are alternative
make docker-push  # Push the image to a registry
make kind-load  # Load the image to kind cluster
```

## Deploy

### Deploy spire

Spire is used by Meridio-Operator for provisioning tHe TLS certificates for the kubeapi server.

```bash
kubectl apply -k <Meridio>/docs/demo/deployments/spire
```

Spire will be deployed in "spire" namespace. Wait until the spire server and agent pods are running to continue deploying Meridio-Operator.

**Note**: certmanager can also be used for the same purpose. To switch from spire to certmanager, you need to comment out all the sections with "[SPIRE]" comment and uncomment all the sections with "[CERTMANAGER]" comment in *config* directory.

### Deploy Meridio-Operator

Meridio-Operator is a namespace-scoped operator, which watches and manages custom resources in a single namespace where the operator is deployed.

The following parameters can be set when you deploy Meridio-Operator:
* NAMESPACE: specify the namespace where the Meridio-Operator resources will be deployed, default is "meridio-operator-system".
* ENABLE_MUTATING_WEBHOOK: enable the mutating webhook to mutate the Cutstom Resources which will be created to configure Meridio-Operator, so that specific fields can be left out and use default values, default is true.
* IMG: specify the image that Meridio-Operator deployment will use, default is controller:0.0.1

```bash
make deploy
```

### Run tests

Running e2e tests requires the Meridio-Operator already deployed in the cluster.

The following parameters are expected to be the same as what are configured when Meridio-Operator is deployed:
* NAMESPACE: specify the namespace where the Meridio-Operator resources will be deployed, default is "meridio-operator-system".
* ENABLE_MUTATING_WEBHOOK: enable the mutating webhook to mutate the Cutstom Resources which will be created to configure Meridio-Operator, so that specific fields can be left out and use default values, default is true.

```bash
# Run e2e test in the namespace "red"
make e2e NAMESPACE="red"
```

## Configuration

**The Meridio-Operator is deployed in the "default" namespace in the examples below.**

### Trench

A [trench](https://github.com/Nordix/Meridio-Operator/blob/master/config/samples/meridio_v1alpha1_trench.yaml) spawns the IPAM, NSP pods, and needed role, role-binding and service accounts, and the ConfigMap storing configuration of the trench. The resources created by a trench will be suffixed with the trench's name.

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

NAME                            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/ipam-service-trench-a   ClusterIP   10.96.6.85     <none>        7777/TCP   1m
service/nsp-service-trench-a    ClusterIP   10.96.71.187   <none>        7778/TCP   1m

NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/ipam-trench-a   1/1     1            1           1m
deployment.apps/nsp-trench-a    1/1     1            1           1m
...
```

### Attractor

An attractor resource is responsible for the creation of NSE-VLAN and stateless-lb-frontend. The resources created will be suffixed with either attractor or trench's name.

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
pod/stateless-lb-frontend-attr-1-57c865cf4c-mzwpj     3/3     Running   0          5s
pod/nse-vlan-attr1-7844574dc-dlgkr    1/1     Running   0          5s

NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/stateless-lb-frontend-attr-1     1/1     1            1           5s
deployment.apps/nse-vlan-attr1   1/1     1            1           5s

$ kubectl get attractor
NAME    INTERFACE-NAME   INTERFACE-TYPE   GATEWAYS                  VIPS              COMPOSITES              REPLICAS   TRENCH
attr1   eth0.100         nsm-vlan         ["gateway1","gateway2"]   ["vip1","vip2"]   ["conduit-stateless"]   1          trench-a
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
pod/proxy-trench-a-dq7pn                                   1/1     Running   0          22m
pod/proxy-trench-a-xzx8m                                   1/1     Running   0          22m
...

NAME                            DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
...
daemonset.apps/proxy-trench-a   2         2         2       2            2           <none>          22m
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

## Resource Management

The resource requirements of containers making up the PODs to be spawned by the Operator can be controlled by annotating the respective custom resource.
As of now, annotation of __Trench__, __Attractor__ and __Conduit__ resources are supported, because these are responsible for creating POD resources.

A Trench can be annotated to set resource requirements by following the example below.
```
apiVersion: meridio.nordix.org/v1alpha1
kind: Trench
metadata:
  name: trench-a
  annotations:
    resource-template: "small"
spec:
  ip-family: dualstack
```

### How it works

For each container making up a specific custom resource (e.g. Trench) the annotation value for key _resource-template_ is interpreted as the name of a resource requirements template. Such templates are defined per container, and are to be specified before building the Operator.

As an example some [templates](https://github.com/Nordix/Meridio-Operator/tree/master/config/manager/resource_requirements/) are included for each container out-of-the-box. But they are not verified to fit any production use cases, and can be overridden at will. (A template is basically a kubernetes [core v1 ResourceRequirements](https://pkg.go.dev/k8s.io/api@v0.22.2/core/v1#ResourceRequirements) block with name.)

The Operator looks up the templates based on the annotation value for each container contributing to the particular custom resource. If a template is missing for a container, then deployment proceeds without setting resource requirements for the container at issue. Otherwise the related resources will be deployed by importing the respective resource requirements from the matching templates.

Updating the annotation of a custom resource is possible. Changes will be applied by kubernetes according to the
Update Strategy of the related resources. Service disturbances and outages are to be expected.
