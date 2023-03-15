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

* `.metadata.name` has a limit of `42` (`63 - RESOURCE_NAME_PREFIX - 15`) characters.
   * `63`: The maximum length for `.metadata.name` in Kubernetes.
   * `RESOURCE_NAME_PREFIX`: An environemnt variable in the operator adding a prefix to the resources being deployed.
   * `15`: Due to the pods names in the `nsp` and `ipam` statefulsets.
* `.spec.ip-family` property is immutable.
* The maximum number of Conduit in a trench is defined by the number of possible subnet. In the IPAM, this is defined by the different between `IPAM_CONDUIT_PREFIX_LENGTH` (`_IPV4` / `_IPV6`) and `IPAM_PREFIX` (`_IPV4` / `_IPV6`).
   * The formula to calculate the maximum number of conduit in a trench is: `2^(IPAM_CONDUIT_PREFIX_LENGTH - IPAM_PREFIX)`
   * e.g. for IPv4, with `IPAM_CONDUIT_PREFIX_LENGTH_IPV4` set to 20 (default) and `IPAM_PREFIX_IPV4` set to 169.255.0.0/16 (default), the maximum number of Conduit in a trench is 16 (`2^(20-16)`).
   * e.g. for IPv6, with `IPAM_CONDUIT_PREFIX_LENGTH_IPV6` set to 56 (default) and `IPAM_PREFIX_IPV6` set to fd00::/48 (default), the maximum number of Conduit in a trench is 256 (`2^(56-48)`).
* The maximum number of proxy per conduit in a trench is defined by the number of possible subnet. In the IPAM, this is defined by the different between `IPAM_NODE_PREFIX_LENGTH` (`_IPV4` / `_IPV6`) and `IPAM_CONDUIT_PREFIX_LENGTH` (`_IPV4` / `_IPV6`).
   * The formula to calculate the maximum number of proxy per conduit in a trench is: `2^(IPAM_NODE_PREFIX_LENGTH - IPAM_CONDUIT_PREFIX_LENGTH)`
   * e.g. for IPv4, with `IPAM_NODE_PREFIX_LENGTH_IPV4` set to 24 (default) and `IPAM_CONDUIT_PREFIX_LENGTH_IPV4` set to 20 (default), the maximum number of Conduit in a trench is 16 (`2^(24-20)`).
   * e.g. for IPv6, with `IPAM_NODE_PREFIX_LENGTH_IPV6` set to 64 (default) and `IPAM_CONDUIT_PREFIX_LENGTH_IPV6` set to 56 (default), the maximum number of Conduit in a trench is 256 (`2^(64-56)`).
* The maximum number of target per proxy per conduit in a trench is defined by the number of possible IP. In the IPAM, this is defined by the `IPAM_NODE_PREFIX_LENGTH_IPV4` and `IPAM_NODE_PREFIX_LENGTH_IPV6` environment variables.
   * The formula to calculate the maximum number of target per proxy per conduit in a trench is: `(((2^(ADDRESS_LENGTH - IPAM_NODE_PREFIX_LENGTH) - 2) - attractor-replicas*2 - 1)/2`
      * `ADDRESS_LENGTH`: 32 for IPv4 and 128 for IPv6.
      * `(... - 2)`: The first and last IPs are not allocated.
      * `2`: 1 attractor takes 2 IPs (proxy and statelesss-lb-frontend side)
      * `-1`: A proxy takes 1 IP for the bridge
      * `/2`: 1 target takes 2 IPs (proxy and target side)
    * e.g. for IPv4, with `IPAM_NODE_PREFIX_LENGTH_IPV4` set to 24 (default) and 2 `attractor-replicas`, the maximum number of target attached to a proxy in a conduit in a trench is 124 (`((2^(32-24)-2)-2*2-1)/2`).
    * e.g. for IPv6, with `IPAM_NODE_PREFIX_LENGTH_IPV6` set to 64 (default) and 2 `attractor-replicas`, the maximum number of target attached to a proxy in a conduit in a trench is 9223372036854775804 (`((2^(128-64)-2)-2*2-1)/2`).

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
Name | string | Name of the Trench | yes | 