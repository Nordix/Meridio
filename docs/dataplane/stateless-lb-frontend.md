# Stateless-lb-frontend

## Ingress

![Dataplane-Stateless-lb-frontend-ingress](../resources/Dataplane-Stateless-lb-frontend-ingress.svg)

### Netfilter nfqueue and defragmentation

List the nftables tables, sets and chains:
* chain nfqlb: brings all traffic with any destination IP in the `ipv4-vips` and `ipv6-vips` sets to the userspace via the nfqueue 0-3
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- nft list ruleset
table inet meridio-nfqlb {
        set ipv4-vips {
                type ipv4_addr
                flags interval
                elements = { 20.0.0.1 }
        }

        set ipv6-vips {
                type ipv6_addr
                flags interval
                elements = { 2000::1 }
        }

        chain nfqlb {
                type filter hook prerouting priority filter; policy accept;
                ip daddr @ipv4-vips counter packets 0 bytes 0 queue to 0-3
                ip6 daddr @ipv6-vips counter packets 0 bytes 0 queue to 0-3
        }

        chain nfqlb-local {
                type filter hook output priority filter; policy accept;
                meta l4proto icmp ip daddr @ipv4-vips counter packets 0 bytes 0 queue to 0-3
                meta l4proto ipv6-icmp ip6 daddr @ipv6-vips counter packets 0 bytes 0 queue to 0-3
        }
}
table inet meridio-nat {
}
table inet meridio-defrag {
        chain pre-defrag {
                type filter hook prerouting priority -500; policy accept;
                iifname "conduit-a-*" notrack
        }

        chain in {
                type filter hook prerouting priority raw; policy accept;
                notrack
                ct state untracked accept
        }

        chain out {
                type filter hook output priority raw; policy accept;
                notrack
        }
}
```
if nft command is not available, it is also possible to these commands:
* Get the nftables: `cat /etc/nftables.nft`

### nfqueue-loadbalancer (nfqlb)

Show nfqlb userspace program running and listening on nfqueue 0-3:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ps
PID   USER     TIME  COMMAND
   19 meridio   0:00 nfqlb flowlb --promiscuous_ping --queue=0:3 --qlength=1024
```

List flows:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- nfqlb flow-list
[{
  "name": "tshm-stream-a-i-flow-a-z-tcp",
  "priority": 1,
  "protocols": [ "tcp" ],
  "dests": [
    "::ffff:20.0.0.1/128",
    "2000::1/128"
  ],
  "dports": [
    "4000"
  ],
  "matches_count": 0,
  "user_ref": "tshm-stream-a-i"
}]
```
This flow will be serving in `stream-a-i`. It will accept only TCP, any source IP and source port, only `vip-a-1-v4` and `vip-a-1-v6` as destination IP and only `4000` as destination port.

List registered targets in `stream-a-i`:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- nfqlb show --shm=tshm-stream-a-i
m: tshm-stream-a-i
  Fw: own=0
  Maglev: M=9973, N=100
   Lookup: 68 68 68 72 43 61 43 61 72 61 61 72 43 68 61 61 43 43 43 61 72 68 61 43 43...
   Active: 5044(43) 5062(61) 5069(68) 5073(72)
```
The 4 Targets are registered (4 identifiers).

Traffic can be captured thanks to nfqlb:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- nfqlb trace-flow-set --name=tshm-stream-a-flow-a
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- nfqlb trace --mask=0xffffffff
```
For more details, see these links:
* https://github.com/Nordix/nfqueue-loadbalancer/blob/1.1.3/log-trace.md#trace
* https://github.com/Nordix/Meridio/tree/v1.0.0/docs/trouble-shooting#flow-trace

NFQLB will then choose a target for the 5-tuple (source/destination ip/port + protocol) and will add a fwmark corresponding to the target identifier to the packet. Then the traffic will go back to the kernel space.

### Policy Routing

List the ip rules for IPv4:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip rule
96:     from all fwmark 0x13d1 lookup 5073
97:     from all fwmark 0x13b4 lookup 5044
98:     from all fwmark 0x13c6 lookup 5062
99:     from all fwmark 0x13cd lookup 5069
```
There are 4 rules matching the fwmark, each rule correspond to a target and have a corresponding table.

List the route for a table for IPv4:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip route show table 5073
default via 172.16.1.8 dev conduit-a--f75b 
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip route show table 5044
default via 172.16.1.6 dev conduit-a--f75b 
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip route show table 5062
default via 172.16.0.8 dev conduit-a--d7c4 
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip route show table 5069
default via 172.16.0.6 dev conduit-a--d7c4
```
The route correspond to the internal IP of the target

List the ip rules for IPv6:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 rule
96:     from all fwmark 0x13d1 lookup 5073
97:     from all fwmark 0x13b4 lookup 5044
98:     from all fwmark 0x13c6 lookup 5062
99:     from all fwmark 0x13cd lookup 5069
```

List the route for a table for IPv6:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 route show table 5073
default via fd00:0:0:1::8 dev conduit-a--f75b metric 1024 pref medium
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 route show table 5044
default via fd00:0:0:1::6 dev conduit-a--f75b metric 1024 pref medium
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 route show table 5062
default via fd00::8 dev conduit-a--d7c4 metric 1024 pref medium
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 route show table 5069
default via fd00::6 dev conduit-a--d7c4 metric 1024 pref medium
```

List interfaces:
* conduit-a--d7c4: Linux kernel interface towards a proxy
* conduit-a--f75b: Linux kernel interface towards another proxy
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip a show dev conduit-a--d7c4
4: conduit-a--d7c4: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 02:fe:44:02:74:01 brd ff:ff:ff:ff:ff:ff
    inet 172.16.0.4/24 brd 172.16.0.255 scope global conduit-a--d7c4
       valid_lft forever preferred_lft forever
    inet6 fd00::4/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:44ff:fe02:7401/64 scope link 
       valid_lft forever preferred_lft forever
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip a show dev conduit-a--f75b
5: conduit-a--f75b: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 02:fe:18:32:8d:87 brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.4/24 brd 172.16.1.255 scope global conduit-a--f75b
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::4/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:18ff:fe32:8d87/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/ext-vlan0/address`

Check the ARP table:
* 172.16.1.6: IPv4 of one of the target
* 02:fe:c2:14:ec:dd: MAC address of the target with `172.16.1.6` as IP
* conduit-a--f75b: The MAC address `02:fe:c2:14:ec:dd` can be reached via this interface
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- cat /proc/net/arp
IP address       HW type     Flags       HW address            Mask     Device
172.16.1.6       0x1         0x2         02:fe:c2:14:ec:dd     *        conduit-a--f75b
172.16.1.8       0x1         0x2         02:fe:54:90:a2:4f     *        conduit-a--f75b
172.16.0.8       0x1         0x2         02:fe:72:70:fd:55     *        conduit-a--d7c4
172.16.0.6       0x1         0x2         02:fe:d6:03:3b:84     *        conduit-a--d7c4
```
note: it is also possible to use `arp -a` or also `ip neighbour`

Check the NDP table:
* fd00:0:0:1::6: IPv6 of one of the target
* 02:fe:c2:14:ec:dd: MAC address of the target with `fd00:0:0:1::6` as IP
* conduit-a--f75b: The MAC address `02:fe:c2:14:ec:dd` can be reached via this interface
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 neighbour
fd00:0:0:1::6 dev conduit-a--f75b lladdr 02:fe:c2:14:ec:dd REACHABLE
fd00:0:0:1::8 dev conduit-a--f75b lladdr 02:fe:54:90:a2:4f REACHABLE
fd00::8 dev conduit-a--d7c4 lladdr 02:fe:72:70:fd:55 REACHABLE
fd00::6 dev conduit-a--d7c4 lladdr 02:fe:d6:03:3b:84 REACHABLE
```

### Netfilter NAT

TODO: Port NAT

### Capture traffic

To capture traffic inside the stateless-lb-frontend:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- tcpdump -nn -i any port 4000
```

## Egress

![Dataplane-Stateless-lb-frontend-egress](../resources/Dataplane-Stateless-lb-frontend-egress.svg)

List the ip rules for IPv4:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip rule
100:    from 20.0.0.1 lookup 4096
```

List the route for a table for IPv4:
* The route correspond to the IPv4 of the Gateway (`169.254.100.150`)
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip route show table 4096
default via 169.254.100.150 dev ext-vlan0 proto bird metric 32
```

List the ip rules for IPv6:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 rule
100:    from 2000::1 lookup 4096
```

List the route for a table for IPv6:
* The route correspond to the IPv6 of the Gateway (`100:100::150`)
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 route show table 4096
default via 100:100::150 dev ext-vlan0 proto bird metric 32 pref medium
```

Check the ARP table:
* 169.254.100.150: IPv4 of the gateway
* 02:42:ac:12:00:06: MAC address of the target with `169.254.100.150` as IP
* ext-vlan0: The MAC address `02:42:ac:12:00:06` can be reached via this interface
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- cat /proc/net/arp
IP address       HW type     Flags       HW address            Mask     Device
169.254.100.150  0x1         0x2         02:42:ac:12:00:06     *        ext-vlan0
```
note: it is also possible to use `arp -a` or also `ip neighbour`

Check the NDP table:
* `100:100::150`: IPv6 of the gateway
* 02:42:ac:12:00:06: MAC address of the target with `fd00:0:0:1::6` as IP
* ext-vlan0: The MAC address `02:42:ac:12:00:06` can be reached via this interface
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 neighbour
100:100::150 dev ext-vlan0 lladdr 02:42:ac:12:00:06 router REACHABLE
```
