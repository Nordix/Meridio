# Target

![Dataplane-target](../resources/Dataplane-target.svg)

List the interfaces in the target:
* nsm-0: contains the 2 VIPs: `20.0.0.1/32` and `2000::1/128`
```sh
$ kubectl exec -it target-a-1 -n red -- ip a show dev nsm-0
3: nsm-0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 02:fe:c2:14:ec:dd brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.6/24 brd 172.16.1.255 scope global nsm-0
       valid_lft forever preferred_lft forever
    inet 20.0.0.1/32 scope global nsm-0
       valid_lft forever preferred_lft forever
    inet6 2000::1/128 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::6/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:c2ff:fe14:ecdd/64 scope link 
       valid_lft forever preferred_lft forever
```

List the ip rules for IPv4:
```sh
$ kubectl exec -it target-a-1 -n red -- ip rule
32765:  from 20.0.0.1 lookup 1
```
There is a rule matching the VIP as source IP address, the rule has a corresponding table.

List the route for a table for IPv4:
```sh
$ kubectl exec -it target-a-1 -n red -- ip route show table 1
default via 172.16.1.1 dev nsm-0 onlink
```
The IP of the nexthop corresponds to the IP of the bridge in the proxy.

List the ip rules for IPv6:
```sh
$ kubectl exec -it target-a-1 -n red -- ip -6 rule
32765:  from 2000::1 lookup 2
```

List the route for a table for IPv6:
```sh
$ kubectl exec -it target-a-1 -n red -- ip -6 route show table 2
default via fd00:0:0:1::1 dev nsm-0 metric 1024 onlink pref medium
```

Check the ARP table:
* 172.16.1.1: IPv4 of the bridge in the proxy
* 02:fe:d2:cc:a2:95: MAC address of the interface with `172.16.1.1` as IP
* nsm-0: The MAC address `02:fe:d2:cc:a2:95` can be reached via this interface
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- cat /proc/net/arp
IP address       HW type     Flags       HW address            Mask     Device
172.16.1.1       0x1         0x2         02:fe:d2:cc:a2:95     *        nsm-0
```
note: it is also possible to use `arp -a` or also `ip neighbour`

Check the NDP table:
* fd00:0:0:1::1: IPv6 of the bridge in the proxy
* 02:fe:d2:cc:a2:95: MAC address of the interface with `fd00:0:0:1::1` as IP
* nsm-0: The MAC address `02:fe:d2:cc:a2:95` can be reached via this interface
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 neighbour
fd00:0:0:1::1 dev nsm-0 lladdr 02:fe:d2:cc:a2:95 router REACHABLE
```
