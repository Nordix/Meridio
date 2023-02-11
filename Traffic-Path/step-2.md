# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-2.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# Gateway / Traffic Generator

List the IPs on the Gateway / Traffic Generator:
* vlan0: VLAN interface (ID: 100) based on `eth0`
* eth0: interface on the same network as the Kubernetes worker nodes
```
docker exec -it trench-a ip a
```{{exec}}

List the IPv4 routes on the Gateway / Traffic Generator:
```
docker exec -it trench-a ip route
```{{exec}}

List the IPv6 routes on the Gateway / Traffic Generator:
```
docker exec -it trench-a ip -6 route
```{{exec}}

You can capture traffic:
```
docker exec -it trench-a tcpdump -nn -i any port 4000
```{{exec}}

# VPP

Get forwarder vpp on first node:
```
FORWARDER=$(kubectl get pods -l app=forwarder-vpp -n nsm --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the VPP interfaces:
* host-eth0: VPP interface representing the Linux kernel base interface `eth0` of the host
* host-eth0.100: VPP VLAN (ID: 100) interface based on `eth0`
* 565487: VPP bridge-domain interface bridging `host-eth0.100` and `tap0`
* tap0: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `stateless-lb-frontend` (ext-vlan0)
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show interface addr
```{{exec}}
Note: `vppctl show mode` can also be used.

List the VPP bridges:
```
kubectl exec -it $FORWARDER -n nsm -- vppctl show bridge-domain
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

# stateless-lb-frontend

Get stateless-lb-frontend on first node:
```
STATELESS_LB_FRONTEND=$(kubectl get pods -l app=stateless-lb-frontend-attractor-a-1 -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the interfaces in the stateless-lb-frontend:
* ext-vlan0: peer of VPP `tap0` interface
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ip a
```{{exec}}
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/ext-vlan0/address`
* Get the ARP table: `cat /proc/net/arp`

You can capture traffic:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- tcpdump -nn -i any port 4000
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
