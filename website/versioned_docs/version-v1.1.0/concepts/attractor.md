# Attractor

The Attractor defines a group of Front-Ends (FEs) sharing commonalities, for example; external interface to utilize, IP-address assignment method, gateway(s) to establish connection with, VIP-addresses to announce etc. 

This resource must be created with label `metadata.labels.trench` to specify its owner reference trench.

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/attractor_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/attractor_types.go)

## Types

TODO

### nsm-vlan

TODO

#### vlan-id 1 to 4094

TODO

The data plane of this type of attractor depends on the NSM version. This is documented here:
* [NSM v1.7.0 to latest](../dataplane/gateway-frontend.md#vlan-id-1-to-4094---nsm-vpp-forwarder-v170-to-latest)
* [NSM v1.1.0 to v1.6.1](../dataplane/gateway-frontend.md#vlan-id-1-to-4094---nsm-vpp-forwarder-v110-to-v161)

Here is an example of an Attractor object with nsm-vlan 1 to 4094:

```yaml
apiVersion: meridio.nordix.org/v1alpha1
kind: Attractor
metadata:
  name: attractor-a-1
  labels:
    trench: trench-a
spec:
  replicas: 2
  composites:
  - conduit-a-1
  gateways:
  - gateway-v4-a-1
  - gateway-v6-a-1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  interface:
    name: ext-vlan0
    ipv4-prefix: 169.254.100.0/24
    ipv6-prefix: 100:100::/64
    type: nsm-vlan
    nsm-vlan:
      vlan-id: 100
      base-interface: eth0
```

#### vlan-id 0

In case the base interface needs to be used without adding any additional vlan tagging, the vlan ID property (`.spec.interface.nsm-vlan.vlan-id`) can be set to 0. This approach may be useful in environments that use vlan in the underlay to provide interfaces for the worker-nodes.

<!-- https://networkservicemesh.io/docs/roadmap/v1.4.0/#connecting-a-remote-interface-without-creating-a-vlan-on-top -->

The data plane of this type of attractor depends on the NSM version. This is documented here:
* [NSM v1.7.0 to latest](../dataplane/gateway-frontend.md#vlan-id-0---nsm-vpp-forwarder-v170-to-latest)
* [NSM v1.4.0 to v1.6.1](../dataplane/gateway-frontend.md#vlan-id-0---nsm-vpp-forwarder-v140-to-v161)

Here is an example of an Attractor object with nsm-vlan 0:

```yaml
apiVersion: meridio.nordix.org/v1alpha1
kind: Attractor
metadata:
  name: attractor-a-1
  labels:
    trench: trench-a
spec:
  replicas: 2
  composites:
  - conduit-a-1
  gateways:
  - gateway-v4-a-1
  - gateway-v6-a-1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  interface:
    name: ext-vlan0
    ipv4-prefix: 169.254.100.0/24
    ipv6-prefix: 100:100::/64
    type: nsm-vlan
    nsm-vlan:
      vlan-id: 0
      base-interface: eth0
```

#### Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get attractors
NAME            INTERFACE-NAME   INTERFACE-TYPE   GATEWAYS                              VIPS                          COMPOSITES        REPLICAS   TRENCH
attractor-a-1   ext-vlan0        nsm-vlan         ["gateway-a-1-v4","gateway-a-1-v6"]   ["vip-a-1-v4","vip-a-1-v6"]   ["conduit-a-1"]   2          trench-a
```

2 deployments:
* `nse-vlan-<attractor-name>`
* `stateless-lb-frontend-<attractor-name>`
```sh
$ kubectl get deployments
NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
nse-vlan-attractor-a-1                1/1     1            1           3m2s
stateless-lb-frontend-attractor-a-1   2/2     2            2           3m2s

$ kubectl get pods
NAME                                                  READY   STATUS    RESTARTS   AGE
nse-vlan-attractor-a-1-5cf67947d5-jfg4m               1/1     Running   0          3m2s
stateless-lb-frontend-attractor-a-1-d8db96c8f-p9g29   3/3     Running   0          3m2s
stateless-lb-frontend-attractor-a-1-d8db96c8f-x4zjh   3/3     Running   0          3m2s
```

A PDB
* `pdb-<attractor-name>`: Pod disruption budget for the `stateless-lb-frontend-attractor-a-1` deployment
```sh
$ kubectl get pdb
NAME                MIN AVAILABLE   MAX UNAVAILABLE   ALLOWED DISRUPTIONS   AGE
pdb-attractor-a-1   75%             N/A               0                     13s
```

The `meridio-configuration-<trench-name>` configmap has also been configured.

The picture below represents a Kubernetes cluster with Attractor applied and highlighted in red:
![Installation-Attractor](../resources/Installation-Attractor.svg)

### network-attachment

TODO

```yaml
apiVersion: meridio.nordix.org/v1
kind: Attractor
metadata:
  name: attractor-a-1
  labels:
    trench: trench-a
spec:
  replicas: 2
  composites:
  - conduit-a-1
  gateways:
  - gateway-v4-a-1
  - gateway-v6-a-1
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
  interface:
    name: eth-ext
    type: network-attachment
    network-attachments:
      - name: ovs-cni-nad
        namespace: default
```

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: ovs-cni-nad
  namespace: default
spec:
  config: '{
      "cniVersion": "0.4.0",
      "name": "myovsnet",
      "plugins": [
        {
          "type":"ovs",
          "name": "myovs",
          "bridge": "br-meridio",
          "vlan": 100,
          "ipam": {
            "log_file": "/tmp/whereabouts.log",
            "type": "whereabouts",
            "ipRanges": [{
              "range": "169.254.100.0/24",
              "exclude": [
                "169.254.100.150/32"
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
```

#### Deployment

TODO

## Limitations

* `.metadata.name` has a limit of `41` (`63 - RESOURCE_NAME_PREFIX - 22`) characters.
   * `63`: The maximum length for `.metadata.name` in Kubernetes.
   * `RESOURCE_NAME_PREFIX`: An environemnt variable in the operator adding a prefix to the resources being deployed.
   * `22`: Due to the pods names in the `stateless-lb-frontend` deployment.
* As described in the [data plane documentation](../dataplane/gateway-frontend.md), using NSM < v1.7.0, deploying multiple attractors with the same VLAN ID will not work.
* An attractor can serve only 1 conduit with `.spec.composites`.
* Using `nsm-vlan`, the based interface (`.spec.interface.nsm-vlan.vlan-id`) must be configured in the device-selector configuration of the NSM forwarder.
* `.spec.interface.*` properties are mandatory and immutable.
* `.metadata.labels.trench` property is mandatory and immutable.

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Attractor | yes | 
vips | []string |  | yes | 
gateways | []string |  | yes | 
trench | string | Name of the Trench the Attractor belongs to | yes | 