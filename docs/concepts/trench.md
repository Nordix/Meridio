# Trench

A Trench defines the extension of an external external network into the Kubernetes cluster. Inside each trench the traffic can be split to take different logical and physical paths through the cluster.

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/trench_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/trench_types.go)

## Example

Here is an example of a Trench object:

```yaml
apiVersion: meridio.nordix.org/v1
kind: Trench
metadata:
  name: trench-a
spec:
  ip-family: dualstack
```

All resources attached to a trench will get IP addresses that belongs to the ip family defined in `.spec.ip-family`. The resources belonging to this trench (Attractor, Gateways, VIPs...) must also be configured according to the ip family defined by the trench.

## Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get trench
NAME       IP-FAMILY
trench-a   dualstack
```

2 Services:
* `ipam-service-<trench-name>`
* `nsp-service-<trench-name>`
```sh
$ kubectl get service 
NAME                     TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
ipam-service-trench-a    ClusterIP   10.96.192.107   <none>        7777/TCP   9s
nsp-service-trench-a     ClusterIP   10.96.77.144    <none>        7778/TCP   9s
```

2 statefulsets with 1 replica in each
* `ipam-<trench-name>`
* `nsp-<trench-name>`
```sh
$ kubectl get statefulsets
NAME            READY   AGE
ipam-trench-a   1/1     2m4s
nsp-trench-a    1/1     2m4s

$ kubectl get pods
NAME               READY   STATUS    RESTARTS   AGE
ipam-trench-a-0    1/1     Running   0          2m4s
nsp-trench-a-0     1/1     Running   0          2m4s
```

1 configmap
* `meridio-configuration-<trench-name>`: Contains all configurations for the custom resources in `trench-a`
```sh
$ kubectl get configmap
NAME                             DATA   AGE
meridio-configuration-trench-a   7      18s
```
This configmap has also been configured with the current trench configuration.

The picture below represents a Kubernetes cluster with Trench applied and highlighted in red:
![Installation-Trench](../resources/Installation-Trench.svg)

## Limitations

TODO

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
Name | string | Name of the Trench | yes | 