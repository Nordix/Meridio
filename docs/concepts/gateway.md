# Gateway

The Gateway defines how to establish connectivity with an external gateway such as; which IP-address, routing- and supervision-protocol(s) to use. The Gateway can also define specific protocol settings that differ from the default etc.

Notice that it normally is good practice and often required have "mirrored" settings shared between the external gateway and the Meridio FrontEnds to get the BGP and BFD sessions established. The used "retry" and "time-out" settings will dictate the time it takes traffic to fail-over in case of a link failure.

Notice that when static is chosen as routing protocol, BFD link-supervision is by default turned on with default settings.

Note: In the Alpha release BGP with BFD is not a supported option.

This resource must be created with label `metadata.labels.trench` to specify its owner reference trench.

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/gateway_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/gateway_types.go)

## Types

TODO

### BGP

TODO

IPv4
```yaml
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v4-a-1
  labels:
    trench: trench-a
spec:
  address: 169.254.100.150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
```

IPv6:
```yaml
apiVersion: meridio.nordix.org/v1alpha1
kind: Gateway
metadata:
  name: gateway-v6-a-1
  labels:
    trench: trench-a
spec:
  address: 100:100::150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
```

#### BFD

TODO

#### BGP Authentication

TODO

<!-- https://github.com/Nordix/Meridio/issues/266
https://github.com/Nordix/Meridio/pull/292
https://github.com/Nordix/Meridio-Operator/pull/125 -->

```yaml
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  name: my-bgp-secret
stringData:
  gateway-v4-a-1-key: MYPASSWORD
```

```yaml
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v4-a-1
  labels:
    trench: trench-a
spec:
  address: 169.254.100.150
  bgp:
    local-asn: 8103
    remote-asn: 4248829953
    hold-time: 24s
    local-port: 10179
    remote-port: 10179
    bfd:
      switch: true
      min-tx: 300ms
      min-rx: 300ms
      multiplier: 5
    auth:
      key-name: gateway-v4-a-1-key
      key-source: my-bgp-secret
```

#### Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get gateways
NAME             ADDRESS           PROTOCOL   TRENCH
gateway-a-1-v4   169.254.100.150   bgp        trench-a
gateway-a-1-v6   100:100::150      bgp        trench-a
```

No new resource has been deployed while deploying the VIPs, but the `meridio-configuration-<trench-name>` configmap has been configured.

The picture below represents a Kubernetes cluster with Gateways applied and highlighted in red:
![Installation-Gateways](../resources/Installation-Gateways.svg)

### Static Routing

TODO

```yaml
```

#### BFD

TODO

#### Deployment

TODO

## Limitations

TODO

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Gateway | yes | 
address | string |  | yes | 
remote-asn | int |  | yes | 
local-asn | int |  | yes | 
remote-port | int |  | yes | 
local-port | int |  | yes | 
ip-family | string |  | yes | 
bfd | bool |  | yes | 
protocol | string |  | yes | 
hold-time | int |  | yes | 
min-tx | int |  | yes | 
min-rx | int |  | yes | 
multiplier | int |  | yes | 
trench | string | Name of the Trench the Gateway belongs to | yes | 
bgp-auth | BgpAuth | Enables BGP authentication. | no | 

### BgpAuth

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
source | string | Name of the kubernetes Secret object containing the pre-shard key to be used for BGP authentication. | yes|
key | string | The key in the kubernetes Secret object's data section identifying the pre-shared key to be used for BGP authentication. | yes|

Note: Adding the kubernetes Secret object is outside the scope of Meridio, but it must share the kubernetes namespace
with the Trench.
