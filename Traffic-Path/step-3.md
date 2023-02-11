# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-3.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# Netfilter

Get stateless-lb-frontend on first node:
```
STATELESS_LB_FRONTEND=$(kubectl get pods -l app=stateless-lb-frontend-attractor-a-1 -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}')
```{{exec}}

List the nftables tables, sets and chains:
* chain nfqlb: brings all traffic with any destination IP in the `ipv4-vips` and `ipv6-vips` sets to the userspace via the nfqueue 0-3
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- nft list ruleset
```{{exec}}
if nft command is not available, it is also possible to these commands:
* Get the nftables: `cat /etc/nftables.nft`

# nfqueue-loadbalancer (nfqlb)

Show nfqlb userspace program running and listening on nfqueue 0-3:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ps
```{{exec}}

List flows:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- nfqlb flow-list
```{{exec}}
This flow will be serving in `stream-a-i`. It will accept only TCP, any source IP and source port, only `vip-a-1-v4` and `vip-a-1-v6` as destination IP and only `4000` as destination port.

List registered targets in `stream-a-i`:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- nfqlb show --shm=tshm-stream-a-i
```{{exec}}
The 4 Targets are registered (4 identifiers).

Traffic can be captured thanks to nfqlb:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- nfqlb trace-flow-set --name=tshm-stream-a-flow-a
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- nfqlb trace --mask=0xffffffff
```
For more details, see these links:
* https://github.com/Nordix/nfqueue-loadbalancer/blob/1.1.3/log-trace.md#trace
* https://github.com/Nordix/Meridio/tree/v1.0.0/docs/trouble-shooting#flow-trace

NFQLB will then choose a target for the 5-tuple (source/destination ip/port + protocol) and will add a fwmark corresponding to the target identifier to the packet. Then the traffic will go back to the kernel space.

# Policy Routing

List the ip rules for IPv4:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ip rule
```{{exec}}
There are 4 rules matching the fwmark, each rule correspond to a target and have a corresponding table.

List the route for a table for IPv4:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ip route show table 1
```{{exec}}
The route correspond to the internal IP of the target

List the ip rules for IPv6:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ip -6 rule
```{{exec}}

List the route for a table for IPv6:
```
kubectl exec -it $STATELESS_LB_FRONTEND -n red -- ip -6 route show table 1
```{{exec}}

# Netfilter

Another prerouting table called `meridio-nat` exists in nftables. It corresponds to the port NATting but this feature is not part of this course.

# Capture traffic

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
