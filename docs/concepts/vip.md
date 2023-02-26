# VIP

The Vip defines a virtual IP address towards which external traffic can be attracted and load-balanced over a scaling set of target application pods. 

As soon as a Vip has been applied to a working trench and attractor the VIP address will be announced by all FEs using BGP. 

This resource must be created with label `metadata.labels.trench` to specify its owner reference trench.

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/vip_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/vip_types.go)

## Example

Here is an example of a IPv4 VIP object:

```yaml
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  name: vip-a-1-v4
  labels:
    trench: trench-a
spec:
  address: "20.0.0.1/32"
```

Here is an example of a IPv6 VIP object:

```yaml
apiVersion: meridio.nordix.org/v1alpha1
kind: Vip
metadata:
  name: vip-a-1-v6
  labels:
    trench: trench-a
spec:
  address: "2000::1/128"
```

## Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get vips
NAME         ADDRESS       TRENCH
vip-a-1-v4   20.0.0.1/32   trench-a
vip-a-1-v6   2000::1/128   trench-a
```

No new resource has been deployed while deploying the VIPs, but the `meridio-configuration-<trench-name>` configmap has been configured.

The picture below represents a Kubernetes cluster with VIPs applied and highlighted in red:
![Installation-VIPs](../resources/Installation-VIPs.svg)

## Limitations

* VIP ranges are not supported. The prefix length must be `/32` for IPv4, and `/128` for IPv6.

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the VIP | yes | 
address | string | The virtual IPaddress. Both ipv4 and ipv6 addresses are supported. The VIP address must be a valid network prefix. | yes | 
trench | string | Name of the Trench the VIP belongs to | yes | 
