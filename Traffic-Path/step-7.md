# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-7.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# proxy

Get proxy on first node:
```
PROXY=$(kubectl get pods -l app=proxy-conduit-a-1 -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the interfaces in the proxy:
* conduit-a.-6496: peer of VPP `tap5` interface on the first worker node
```
kubectl exec -it $PROXY -n red -- ip a
```{{exec}}

# VPP

Get forwarder vpp on first node:
```
FORWARDER=$(kubectl get pods -l app=forwarder-vpp -n nsm --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the VPP interfaces:
* tap5: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (conduit-a.-6496). It is cross connected (l2 xconnect) with `vxlan_tunnel0`
* vxlan_tunnel0: VPP VxLAN. It is cross connected (l2 xconnect) with `tap5`. The VxLAN ID is 9832580, it uses the host IP addresses and port 4789
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show interface addr
```{{exec}}

Get more details (source/destination IP/Port, VxLAN ID...) about the VxLAN tunnels:
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show vxlan tunnel raw
```{{exec}}
Note: You can get the `sw-if-idx` with `vppctl show interface`

Capture the VxLAN traffic (port: 4789 and VxLAN ID: 9832580):
```
kubectl exec -it $FORWARDER -n nsm -- tcpdump -nn -i eth0 'port 4789 and udp[8:2] = 0x0800 & 0x0800 and udp[11:4] = 9832580 & 0x00FFFFFF'
```{{exec}}

# VPP (Second worker node)

Get forwarder vpp on second node:
```
FORWARDER_2=$(kubectl get pods -l app=forwarder-vpp -n nsm --field-selector spec.nodeName=kind-worker2 --no-headers=true | awk '{print $1}')
```{{exec}}

List the VPP interfaces:
* tap5: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (conduit-a.-6496). It is cross connected (l2 xconnect) with `vxlan_tunnel0`
* vxlan_tunnel0: VPP VxLAN. It is cross connected (l2 xconnect) with `tap5`. The VxLAN ID is 9832580, it uses the host IP addresses and port 4789
```
kubectl exec -it $FORWARDER_2 -n nsm -- vppctl show interface addr
```{{exec}}

# stateless-lb-frontend (Second worker node)

Get stateless-lb-frontend on second node:
```
STATELESS_LB_FRONTEND_2=$(kubectl get pods -l app=stateless-lb-frontend-attractor-a-1 -n red --field-selector spec.nodeName=kind-worker2 --no-headers=true | awk '{print $1}')
```{{exec}}

List the interfaces in the proxy:
* conduit-a.-b84a: peer of VPP `tap5` interface on the second worker node
```
kubectl exec -it $STATELESS_LB_FRONTEND_2 -n red -- ip a
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
