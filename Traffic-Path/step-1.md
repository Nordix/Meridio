# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-1.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# Trench

The trench, called `trench-a` works with dualstack, it has 2 gateways (169.254.100.150 and 100:100::150), 2 VIPs (20.0.0.1/32 and 2000::1/128), 1 attractor `attractor-a-1`, 1 conduit `conduit-a-1`, 1 stream `stream-a-i` and a flow accepting `4000` as destination port.

Here is the trench that has been applied:
```
---
apiVersion: meridio.nordix.org/v1
kind: Trench
metadata:
  name: trench-a
  namespace: red
spec:
  ip-family: dualstack
---
apiVersion: meridio.nordix.org/v1
kind: Attractor
metadata:
  name: attractor-a-1
  namespace: red
  labels:
    trench: trench-a
spec:
  replicas: 2
  composites:
  - conduit-a-1
  gateways:
  - gateway-v4-a
  - gateway-v6-a
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
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v4-a
  namespace: red
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
---
apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  name: gateway-v6-a
  namespace: red
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
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  name: vip-a-1-v4
  namespace: red
  labels:
    trench: trench-a
spec:
  address: "20.0.0.1/32"
---
apiVersion: meridio.nordix.org/v1
kind: Vip
metadata:
  name: vip-a-1-v6
  namespace: red
  labels:
    trench: trench-a
spec:
  address: "2000::1/128"
---
apiVersion: meridio.nordix.org/v1
kind: Conduit
metadata:
  name: conduit-a-1
  namespace: red
  labels:
    trench: trench-a
spec:
  type: stateless-lb
---
apiVersion: meridio.nordix.org/v1
kind: Stream
metadata:
  name: stream-a-i
  namespace: red
  labels:
    trench: trench-a
spec:
  conduit: conduit-a-1
---
apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  name: flow-a-z-tcp
  namespace: red
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