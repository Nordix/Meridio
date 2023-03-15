# Flow

The Flow classifies incoming traffic based on the 5-tuple in the IP-header containing: Source and destination IP-addresses, source and destination ports and protocol type.

The traffic that match the classification are mapped to a logically stream.

It is allowed to configure flows having overlapping IP-addresses and port ranges. In this case the priority setting is used to decide which flow will be tried matched first. 

Traffic that do not match the classification for any flow will be dropped.

Notice that destination IP-address is provided by referencing one or more Vip.

This resource must be created with label `metadata.labels.trench` to specify its owner reference trench.

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/flow_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/flow_types.go)

## Example

Here is an example of a Flow object:

```yaml
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a-z-tcp
  labels:
    trench: trench-a
spec:
  stream: stream-a-i
  priority: 1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  source-subnets:
  - 0.0.0.0/0
  - 0:0:0:0:0:0:0:0/0
  source-ports:
  - any
  destination-ports:
  - "4000"
  protocols:
  - tcp
```

The stream will be applied for `stream-conduit-a-1`. Any source IP and source port will be accepted, and only TCP with `vip-a-1-v4` or `vip-a-1-v6` as destination IP and 4000 as destination port can go through this flow.

## Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get flows
NAME           VIPS                          DST-PORTS   SRC-SUBNETS                         SRC-PORTS   PROTOCOLS   BYTE-MATCHES   STREAM       TRENCH
flow-a-z-tcp   ["vip-a-1-v4","vip-a-1-v6"]   ["4000"]    ["0.0.0.0/0","0:0:0:0:0:0:0:0/0"]   ["any"]     ["tcp"]                    stream-a-i   trench-a
```

No new resource has been deployed while deploying the VIPs, but the `meridio-configuration-<trench-name>` configmap has been configured.

The picture below represents a Kubernetes cluster with Flow applied and highlighted in red:
![Installation-Flow](../resources/Installation-Flow.svg)

## Limitations

* `.metadata.labels.trench` property is mandatory and immutable.

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
Name | string | Name of the Flow | yes |
source-subnets | []string |  | yes | 
destination-port-ranges | []string |  | yes | 
source-port-ranges | []string |  | yes | 
protocols | []string |  | yes | 
vips | []string |  | yes | 
priority | int |  | yes | 
stream | string | Name of the Stream the Flow belongs to | yes | 
