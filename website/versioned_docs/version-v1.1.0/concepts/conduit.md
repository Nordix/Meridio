# Conduit

The Conduit defines a physical path of chained network services the traffic will pass before it can be consumed by the target application. The Traffic is assigned to the Conduit by Stream. Currently, the only supported network service function is stateless load balancing.

The picture below describes 2 conduits. the left one is a stateless load-balancer and the right one is a SCTP network service. The left user pod has subscribed to one or more stream the stateless load-balancer is handling, so it is connected to it via a new network interface. The right user pod has subscribed to streams the stateless load-balancer and the SCTP network services are handling, so it is connected to them via two new network interfaces, one connected to the stateless load-balancer network service, and the other one connected to the SCTP network service. 

This resource must be created with label `metadata.labels.trench` to specify its owner reference trench.

![Overview-Conduit](../resources/Overview-Conduit.svg)

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/conduit_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/conduit_types.go)

## Types

TODO

### Stateless load balancer 

The Stateless load balancer type (`.spec.type: stateless-lb`) provides stateless load-balancing network function via the Maglev algorithm. 

Here is an example of a stateless-lb conduit object:

```yaml
apiVersion: meridio.nordix.org/v1
kind: Conduit
metadata:
  name: conduit-a-1
  labels:
    trench: trench-a
spec:
  type: stateless-lb
```

#### Port-NAT

As defined in the [use cases](../use-cases.md), NATting might require it in order to expose their service with certain ports. By default, privileged ports under 1024 require root or CAP_NET_BIND_SERVICE in order to bind to them. Therefore, applications must NAT service port to an unprivileged port open on their host.

Here is an example of destination port NATting in a conduit object:

```yaml
apiVersion: meridio.nordix.org/v1
kind: Conduit
metadata:
  name: conduit-a-1
  labels:
    trench: trench-a
spec:
  type: stateless-lb
  destination-port-nats:
  - port: 80
    target-port: 4000
    vips:
    - vip-a-1-v4
    - vip-a-1-v6
    protocol: "tcp"
```

All TCP traffic with `vip-a-1-v4` or `vip-a-1-v6` as desination IP and 80 as destination port will get the destination port translated to 4000.

## Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get conduits
NAME          TYPE           TRENCH
conduit-a-1   stateless-lb   trench-a
```

A daemonset: 
* `proxy-<conduit-name>`
```sh
$ kubectl get daemonsets
NAME                DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
proxy-conduit-a-1   2         2         2       2            2           <none>          67s

$ kubectl get pods
NAME                      READY   STATUS    RESTARTS   AGE
proxy-conduit-a-1-tbjgg   1/1     Running   0          67s
proxy-conduit-a-1-w4qbk   1/1     Running   0          67s
```

The `meridio-configuration-<trench-name>` configmap has also been configured.

The picture below represents a Kubernetes cluster with Conduit applied and highlighted in red:
![Installation-Conduit](../resources/Installation-Conduit.svg)

## Limitations

* `.metadata.name` has a limit of `57` (`63 - RESOURCE_NAME_PREFIX - 6`) characters.
   * `63`: The maximum length for `.metadata.name` in Kubernetes.
   * `RESOURCE_NAME_PREFIX`: An environemnt variable in the operator adding a prefix to the resources being deployed.
   * `6`: Due to the pods names in the `proxy` deployment.
* `.metadata.labels.trench` property is mandatory and immutable.
* `.spec.type` property is mandatory and immutable.

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Conduit | yes |
trench | string | Name of the Trench the Conduit belongs to | yes | 
destination-port-nats | []PortNat | List of destination ports to NAT. | no | 
