# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-4.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# stateless-lb-frontend

Get stateless-lb-frontend on first node:
```
STATELESS_LB_FRONTEND=$(kubectl get pods -l app=stateless-lb-frontend-attractor-a-1 -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the interfaces in the stateless-lb-frontend:
* conduit-a.-b89d: peer of VPP `tap1` interface
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ip a
```{{exec}}

# proxy

Get proxy on first node:
```
PROXY=$(kubectl get pods -l app=proxy-conduit-a-1 -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the interfaces in the proxy:
* conduit-a.-3150: peer of VPP `tap2` interface
* proxy.cond-0a85: peer of VPP `tap3` interface
* bridge0: Linux kernel bridge interface bridging `conduit-a.-3150` and `proxy.cond-0a85`
```
kubectl exec -it $PROXY -n red -- ip a
```{{exec}}

# target

Get a target on first node:
```
TARGET=$(kubectl get pods -l app=target-a -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}' | head -n 1)
```{{exec}}

List the interfaces in the target:
* nsm-0: peer of VPP `tap4` interface
```
kubectl exec -it $TARGET -n red -- ip a
```{{exec}}

# VPP

Get forwarder vpp on first node:
```
FORWARDER=$(kubectl get pods -l app=forwarder-vpp -n nsm --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the VPP interfaces:
* tap1: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `stateless-lb-frontend` (conduit-a.-b89d). It is cross connected (l2 xconnect) with `tap2`
* tap2: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (conduit-a.-3150). It is cross connected (l2 xconnect) with `tap1`
* tap3: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (proxy.cond-0a85). It is cross connected (l2 xconnect) with `tap4`
* tap4: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `target` (nsm-0). It is cross connected (l2 xconnect) with `tap3`
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show interface addr
```{{exec}}

To find the peer interface of a TAP interface in VPP, you can do it by listing the TAP interfaces and finding the one that has the same `host-mac-addr` property as the MAC address on the Linux Kernel interface.
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show tap
```{{exec}}

It is also possible to get metrics for the VPP interfaces:
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show interface
```{{exec}}

You can capture VLAN traffic with VLAN 100 and 4000 as destination port with:
```
kubectl exec -it $FORWARDER -n nsm -- tcpdump -nn -i eth0 -e 'dst port 4000 and (vlan 100)'
```{{exec}}

# Traffic

Send IPv4 traffic with `20.0.0.1` as destination IP and `4000` as destination port:
```
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
```{{exec}}

Create a single TCP connection with `20.0.0.1` as destination IP, `4000` as destination port and `35000` as source port:
```
docker exec -it trench-a timeout 0.5s nc 20.0.0.1 4000 -p 35000
```{{exec}}
