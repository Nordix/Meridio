# Stateless-lb-frontend - Target

## VPP-Forwarder: Same node stateless-lb-frontend - proxy

![Dataplane-same-node-stateless-lb-frontend-proxy](../resources/Dataplane-same-node-stateless-lb-frontend-proxy.svg)

List the interfaces in the stateless-lb-frontend:
* conduit-a--f75b: peer of VPP `tap3` interface
```sh
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
* Get the MAC address of an interface: `cat /sys/class/net/conduit-a--f75b/address`

List the interfaces in the proxy:
* conduit-a--1b2a: peer of VPP `tap4` interface
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip a show dev conduit-a--1b2a
5: conduit-a--1b2a: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq master bridge0 state UNKNOWN group default qlen 1000
    link/ether 02:fe:7d:e7:f6:2a brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.5/24 brd 172.16.1.255 scope global conduit-a--1b2a
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::5/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:7dff:fee7:f62a/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/conduit-a--1b2a/address`

List the VPP interfaces:
* tap3: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `stateless-lb-frontend` (conduit-a--f75b). It is cross connected (l2 xconnect) with `tap4`
* tap4: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (conduit-a--1b2a). It is cross connected (l2 xconnect) with `tap3`
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface addr
tap3 (up):
  L2 xconnect tap4
tap4 (up):
  L2 xconnect tap3
```

To find the peer interface of a TAP interface in VPP, you can do it by listing the TAP interfaces and finding the one that has the same `host-mac-addr` property as the MAC address on the Linux Kernel interface.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show tap tap3
Interface: tap3 (ifindex 7)
  name "conduit-a--f75b"
  host-ns "/proc/1/fd/43"
  host-mac-addr: 02:fe:18:32:8d:87
  host-carrier-up: 1
  vhost-fds 30
  tap-fds 29
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:62:6c:fa:ab
  Device instance: 3
  flags 0x1
    admin-up (0)
  features 0x110008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_F_VERSION_1 (32)
  remote-features 0x33d008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_F_NOTIFY_ON_EMPTY (24)
    VHOST_F_LOG_ALL (26)
    VIRTIO_F_ANY_LAYOUT (27)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_RING_F_EVENT_IDX (29)
    VIRTIO_F_VERSION_1 (32)
    VIRTIO_F_IOMMU_PLATFORM (33)
  Number of RX Virtqueue  1
  Number of TX Virtqueue  1
  Virtqueue (RX) 0
    qsz 1024, last_used_idx 6, desc_next 960, desc_in_use 954
    avail.flags 0x0 avail.idx 960 used.flags 0x1 used.idx 6
    kickfd 32, callfd 31
  Virtqueue (TX) 1
    qsz 1024, last_used_idx 36, desc_next 37, desc_in_use 1
    avail.flags 0x1 avail.idx 37 used.flags 0x0 used.idx 37
    kickfd 33, callfd -1
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show tap tap4
Interface: tap4 (ifindex 8)
  name "conduit-a--1b2a"
  host-ns "/proc/1/fd/37"
  host-mac-addr: 02:fe:7d:e7:f6:2a
  host-carrier-up: 1
  vhost-fds 35
  tap-fds 34
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:c9:28:47:65
  Device instance: 4
  flags 0x1
    admin-up (0)
  features 0x110008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_F_VERSION_1 (32)
  remote-features 0x33d008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_F_NOTIFY_ON_EMPTY (24)
    VHOST_F_LOG_ALL (26)
    VIRTIO_F_ANY_LAYOUT (27)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_RING_F_EVENT_IDX (29)
    VIRTIO_F_VERSION_1 (32)
    VIRTIO_F_IOMMU_PLATFORM (33)
  Number of RX Virtqueue  1
  Number of TX Virtqueue  1
  Virtqueue (RX) 0
    qsz 1024, last_used_idx 37, desc_next 960, desc_in_use 923
    avail.flags 0x0 avail.idx 960 used.flags 0x1 used.idx 37
    kickfd 37, callfd 36
  Virtqueue (TX) 1
    qsz 1024, last_used_idx 3, desc_next 4, desc_in_use 1
    avail.flags 0x1 avail.idx 4 used.flags 0x0 used.idx 4
    kickfd 38, callfd -1
```

Access the network namespace of the `tap5` peer and `tap4` peer:
* `/proc/1/fd/43`: network namespace file (`host-ns`) of the `tap3` peer
* `/proc/1/fd/37`: network namespace file (`host-ns`) of the `tap4` peer
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- nsenter --net=/proc/1/fd/43 bash
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- nsenter --net=/proc/1/fd/37 bash
```

List the VPP interfaces with metrics and index:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface
              Name               Idx    State  MTU (L3/IP4/IP6/MPLS)     Counter          Count     
tap3                              8      up     1500/1500/1500/1500 rx packets                     6
                                                                    rx bytes                     796
                                                                    tx packets                    37
                                                                    tx bytes                    3578
                                                                    drops                          2
                                                                    ip6                            2
tap4                              9      up     1500/1500/1500/1500 rx packets                    37
                                                                    rx bytes                    3578
                                                                    tx packets                     4
                                                                    tx bytes                     536
```

To capture traffic inside the vpp forwarder:
* `vppctl pcap trace rx tx max COUNT intfc INTERFACE`: Start capturing traffic
* `vppctl pcap trace off`: Stop trace. You can use `tcpdump -nn -e -r /tmp/rxtx.pcap` to read it or use Wireshark.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc tap3
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc tap4
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
```

## VPP-Forwarder: Different node stateless-lb-frontend - proxy

![Dataplane-different-node-stateless-lb-frontend-proxy](../resources/Dataplane-different-node-stateless-lb-frontend-proxy.svg)

### Proxy Node

List the interfaces in the proxy:
* conduit-a--90c8: peer of VPP `tap1` interface
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip a show dev conduit-a--90c8
4: conduit-a--90c8: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc mq master bridge0 state UNKNOWN group default qlen 1000
    link/ether 02:fe:eb:4a:02:dc brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.3/24 brd 172.16.1.255 scope global conduit-a--90c8
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::3/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:ebff:fe4a:2dc/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/conduit-a--90c8/address`

List the VPP interfaces:
* vxlan_tunnel0: VPP VxLAN. It is cross connected (l2 xconnect) with `tap1`. The VxLAN ID is 9832580, it uses the host IP addresses and port 4789
* tap1: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (conduit-a--90c8). It is cross connected (l2 xconnect) with `vxlan_tunnel0`
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface addr
tap1 (up):
  L2 xconnect vxlan_tunnel0
vxlan_tunnel0 (up):
  L2 xconnect tap1
```

To find the peer interface of a TAP interface in VPP, you can do it by listing the TAP interfaces and finding the one that has the same `host-mac-addr` property as the MAC address on the Linux Kernel interface.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show tap tap1
Interface: tap1 (ifindex 4)
  name "conduit-a--90c8"
  host-ns "/proc/1/fd/31"
  host-mac-addr: 02:fe:eb:4a:02:dc
  host-carrier-up: 1
  vhost-fds 20
  tap-fds 19
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:49:44:7f:80
  Device instance: 1
  flags 0x1
    admin-up (0)
  features 0x110008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_F_VERSION_1 (32)
  remote-features 0x33d008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_F_NOTIFY_ON_EMPTY (24)
    VHOST_F_LOG_ALL (26)
    VIRTIO_F_ANY_LAYOUT (27)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_RING_F_EVENT_IDX (29)
    VIRTIO_F_VERSION_1 (32)
    VIRTIO_F_IOMMU_PLATFORM (33)
  Number of RX Virtqueue  1
  Number of TX Virtqueue  1
  Virtqueue (RX) 0
    qsz 1024, last_used_idx 42, desc_next 960, desc_in_use 918
    avail.flags 0x0 avail.idx 960 used.flags 0x1 used.idx 42
    kickfd 22, callfd 21
  Virtqueue (TX) 1
    qsz 1024, last_used_idx 3, desc_next 4, desc_in_use 1
    avail.flags 0x1 avail.idx 4 used.flags 0x0 used.idx 4
    kickfd 23, callfd -1
```

Access the network namespace of the `tap1` peer:
* `/proc/1/fd/31`: network namespace file (`host-ns`) of the `tap1` peer
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- nsenter --net=/proc/1/fd/31 bash
```

Get more details (source/destination IP/Port, VxLAN ID...) about the VxLAN tunnels:
* 172.18.0.2: Source IP the VxLAN will use
* 172.18.0.4: Destination IP the VxLAN will use (Check with `ip route get 172.18.0.4` to find through which interface the traffic will go)
* 4789: Source and destination port used for vxlan
* 9832580: VNI / VxLAN ID
* 4: Index of the VPP interface (can be found with `vppctl show interface`)
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show vxlan tunnel raw
[0] instance 0 src 172.18.0.2 dst 172.18.0.4 src_port 4789 dst_port 4789 vni 9832580 fib-idx 0 sw-if-idx 4 encap-dpo-idx 2 decap-next-index 3 
```
Note: You can get the `sw-if-idx` with `vppctl show interface`

List the VPP interfaces with metrics and index:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface
              Name               Idx    State  MTU (L3/IP4/IP6/MPLS)     Counter          Count     
tap1                              5      up     1400/1400/1400/1400 rx packets                    42
                                                                    rx bytes                    4344
                                                                    tx packets                     4
                                                                    tx bytes                     536
vxlan_tunnel0                     4      up     1400/1400/1400/1400 rx packets                     4
                                                                    rx bytes                     536
                                                                    tx packets                    42
                                                                    tx bytes                    5856
```

To capture traffic inside the vpp forwarder:
* `vppctl pcap trace rx tx max COUNT intfc INTERFACE`: Start capturing traffic
* `vppctl pcap trace off`: Stop trace. You can use `tcpdump -nn -e -r /tmp/rxtx.pcap` to read it or use Wireshark.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc vxlan_tunnel0
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc tap1
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
```

List the interfaces in worker node:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- ip a show dev eth0
899: eth0@if900: <BROADCAST,MULTICAST,PROMISC,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default 
    link/ether 02:42:ac:12:00:02 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.18.0.2/16 brd 172.18.255.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fc00:f853:ccd:e793::2/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:2/64 scope link 
       valid_lft forever preferred_lft forever
```

To capture the VxLAN traffic with 9832580 as VNI, 4789 as port and eth0 as base interface:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- tcpdump -nn -i eth0 'port 4789 and udp[8:2] = 0x0800 & 0x0800 and udp[11:4] = 9832580 & 0x00FFFFFF'
```

### Stateless-lb-frontend Node

List the interfaces in the stateless-lb-frontend:
* conduit-a--0a17: peer of VPP `tap1` interface
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-jkl -n red -- ip a show dev conduit-a--0a17
4: conduit-a--0a17: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 02:fe:66:8b:f7:da brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.2/24 brd 172.16.1.255 scope global conduit-a--0a17
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::2/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:66ff:fe8b:f7da/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/conduit-a--0a17/address`

List the VPP interfaces:
* vxlan_tunnel0: VPP VxLAN. It is cross connected (l2 xconnect) with `tap1`. The VxLAN ID is 9832580, it uses the host IP addresses and port 4789
* tap1: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (conduit-a--0a17). It is cross connected (l2 xconnect) with `vxlan_tunnel0`
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl show interface addr
tap1 (up):
  L2 xconnect vxlan_tunnel0
vxlan_tunnel0 (up):
  L2 xconnect tap1
```

To find the peer interface of a TAP interface in VPP, you can do it by listing the TAP interfaces and finding the one that has the same `host-mac-addr` property as the MAC address on the Linux Kernel interface.
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl show tap tap1
Interface: tap1 (ifindex 3)
  name "conduit-a--0a17"
  host-ns "/proc/1/fd/32"
  host-mac-addr: 02:fe:66:8b:f7:da
  host-carrier-up: 1
  vhost-fds 20
  tap-fds 19
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:cb:c6:ff:65
  Device instance: 1
  flags 0x1
    admin-up (0)
  features 0x110008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_F_VERSION_1 (32)
  remote-features 0x33d008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_F_NOTIFY_ON_EMPTY (24)
    VHOST_F_LOG_ALL (26)
    VIRTIO_F_ANY_LAYOUT (27)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_RING_F_EVENT_IDX (29)
    VIRTIO_F_VERSION_1 (32)
    VIRTIO_F_IOMMU_PLATFORM (33)
  Number of RX Virtqueue  1
  Number of TX Virtqueue  1
  Virtqueue (RX) 0
    qsz 1024, last_used_idx 5, desc_next 960, desc_in_use 955
    avail.flags 0x0 avail.idx 960 used.flags 0x1 used.idx 5
    kickfd 22, callfd 21
  Virtqueue (TX) 1
    qsz 1024, last_used_idx 41, desc_next 42, desc_in_use 1
    avail.flags 0x1 avail.idx 42 used.flags 0x0 used.idx 42
    kickfd 23, callfd -1
```

Access the network namespace of the `tap5` peer:
* `/proc/1/fd/32`: network namespace file (`host-ns`) of the `tap1` peer
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- nsenter --net=/proc/1/fd/32 bash
```

Get more details (source/destination IP/Port, VxLAN ID...) about the VxLAN tunnels:
* 172.18.0.4: Source IP the VxLAN will use
* 172.18.0.2: Destination IP the VxLAN will use (Check with `ip route get 172.18.0.2` to find through which interface the traffic will go)
* 4789: Source and destination port used for vxlan
* 9832580: VNI / VxLAN ID
* 5: Index of the VPP interface (can be found with `vppctl show interface`)
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl show vxlan tunnel raw
[0] instance 0 src 172.18.0.4 dst 172.18.0.2 src_port 4789 dst_port 4789 vni 9832580 fib-idx 0 sw-if-idx 5 encap-dpo-idx 1 decap-next-index 3 
```

List the VPP interfaces with metrics and index:
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl show interface
              Name               Idx    State  MTU (L3/IP4/IP6/MPLS)     Counter          Count    
tap1                              4      up     1400/1400/1400/1400 rx packets                     5
                                                                    rx bytes                     686
                                                                    tx packets                    42
                                                                    tx bytes                    4344
                                                                    drops                          1
                                                                    ip6                            1
vxlan_tunnel0                     5      up     1400/1400/1400/1400 rx packets                    42
                                                                    rx bytes                    4344
                                                                    tx packets                     4
                                                                    tx bytes                     680
```

To capture traffic inside the vpp forwarder:
* `vppctl pcap trace rx tx max COUNT intfc INTERFACE`: Start capturing traffic
* `vppctl pcap trace off`: Stop trace. You can use `tcpdump -nn -e -r /tmp/rxtx.pcap` to read it or use Wireshark.
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl pcap trace rx tx max 100 intfc vxlan_tunnel0
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl pcap trace off
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl pcap trace rx tx max 100 intfc tap1
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- vppctl pcap trace off
```

List the interfaces in worker node:
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- ip a show dev eth0
903: eth0@if904: <BROADCAST,MULTICAST,PROMISC,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default 
    link/ether 02:42:ac:12:00:04 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.18.0.4/16 brd 172.18.255.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fc00:f853:ccd:e793::4/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::42:acff:fe12:4/64 scope link 
       valid_lft forever preferred_lft forever
```

To capture the VxLAN traffic with 9832580 as VNI, 4789 as port and eth0 as base interface:
```sh
$ kubectl exec -it forwarder-vpp-worker-b -n nsm -- tcpdump -nn -i eth0 'port 4789 and udp[8:2] = 0x0800 & 0x0800 and udp[11:4] = 9832580 & 0x00FFFFFF'
```

## Proxy: Bridging (Ingress) and Routing (Egress)

![Dataplane-proxy](../resources/Dataplane-proxy.svg)

List the interfaces in the proxy:
* bridge0: Linux kernel bridge interface bridging `conduit-a--90c8`, `conduit-a--1b2a` and `proxy.cond-97e3`
* conduit-a--90c8: Linux kernel interface towards a stateless-lb-frontend attached to `bridge0`
* conduit-a--1b2a: Linux kernel interface towards a stateless-lb-frontend attached to `bridge0`
* proxy.cond-97e3: Linux kernel interface towards a target attached to `bridge0`
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip a
3: bridge0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc noqueue state UP group default 
    link/ether 02:fe:d2:cc:a2:95 brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.1/24 brd 172.16.1.255 scope global bridge0
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::1/64 scope global 
       valid_lft forever preferred_lft forever
    inet6 fe80::e443:1fff:fe88:c669/64 scope link 
       valid_lft forever preferred_lft forever
4: conduit-a--90c8: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1400 qdisc mq master bridge0 state UNKNOWN group default qlen 1000
    link/ether 02:fe:eb:4a:02:dc brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.3/24 brd 172.16.1.255 scope global conduit-a--90c8
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::3/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:ebff:fe4a:2dc/64 scope link 
       valid_lft forever preferred_lft forever
5: conduit-a--1b2a: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq master bridge0 state UNKNOWN group default qlen 1000
    link/ether 02:fe:7d:e7:f6:2a brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.5/24 brd 172.16.1.255 scope global conduit-a--1b2a
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::5/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:7dff:fee7:f62a/64 scope link 
       valid_lft forever preferred_lft forever
6: proxy.cond-97e3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq master bridge0 state UNKNOWN group default qlen 1000
    link/ether 02:fe:d4:d5:c4:53 brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.7/24 brd 172.16.1.255 scope global proxy.cond-97e3
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::7/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:d4ff:fed5:c453/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/bridge0/address`

Check the Forwarding Database entry:
* 02:fe:c2:14:ec:dd: (MAC address of a target) is accessible via `proxy.cond-97e3`
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- bridge fdb
02:fe:c2:14:ec:dd dev proxy.cond-97e3 master bridge0
```

List the ip rules for IPv4:
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip rule
32765:  from 20.0.0.1 lookup 1
```
There is a rule matching the VIP as source IP address, the rule has a corresponding table.

List the route for a table for IPv4:
* 172.16.1.2: IP of the stateless-lb-frontend on first node (See section: Same node)
* 172.16.1.4: IP of the stateless-lb-frontend on second node (See section: Different node)
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip route show table 1
default
        nexthop via 172.16.1.2 dev bridge0 weight 1
        nexthop via 172.16.1.4 dev bridge0 weight 1
```

List the ip rules for IPv6:
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip -6 rule
32765:  from 2000::1 lookup 2
```

List the route for a table for IPv6:
* fd00:0:0:1::2: IP of the stateless-lb-frontend on first node (See section: Same node)
* fd00:0:0:1::4: IP of the stateless-lb-frontend on second node (See section: Different node)
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip -6 route show table 2
default metric 1024 pref medium
        nexthop via fd00:0:0:1::2 dev bridge0 weight 1
        nexthop via fd00:0:0:1::4 dev bridge0 weight 1
```

Check the ARP table:
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- cat /proc/net/arp
IP address       HW type     Flags       HW address            Mask     Device
172.16.1.4       0x1         0x2         02:fe:18:32:8d:87     *        bridge0
172.16.1.2       0x1         0x2         02:fe:66:8b:f7:da     *        bridge0
```
note: it is also possible to use `arp -a` or also `ip neighbour`

Check the NDP table:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip -6 neighbour
fd00:0:0:1::4 dev bridge0 lladdr 02:fe:18:32:8d:87 router REACHABLE
fd00:0:0:1::2 dev bridge0 lladdr 02:fe:66:8b:f7:da router REACHABLE
```

## VPP-Forwarder: Proxy - Target

![Dataplane-proxy-target](../resources/Dataplane-proxy-target.svg)

List the interfaces in the proxy:
* proxy.cond-97e3: peer of VPP `tap5` interface
```sh
$ kubectl exec -it proxy-conduit-a-1-abc -n red -- ip a show dev proxy.cond-97e3
6: proxy.cond-97e3: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq master bridge0 state UNKNOWN group default qlen 1000
    link/ether 02:fe:d4:d5:c4:53 brd ff:ff:ff:ff:ff:ff
    inet 172.16.1.7/24 brd 172.16.1.255 scope global proxy.cond-97e3
       valid_lft forever preferred_lft forever
    inet6 fd00:0:0:1::7/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:d4ff:fed5:c453/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/proxy.cond-97e3/address`

List the interfaces in the target:
* nsm-0: peer of VPP `tap6` interface
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
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/nsm-0/address`

List the VPP interfaces:
* tap5: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `proxy` (proxy.cond-97e3). It is cross connected (l2 xconnect) with `tap6`
* tap6: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `target` (nsm-0). It is cross connected (l2 xconnect) with `tap5`
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface addr
tap5 (up):
  L2 xconnect tap6
tap6 (up):
  L2 xconnect tap5
```

To find the peer interface of a TAP interface in VPP, you can do it by listing the TAP interfaces and finding the one that has the same `host-mac-addr` property as the MAC address on the Linux Kernel interface.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show tap tap5
Interface: tap5 (ifindex 9)
  name "proxy.cond-97e3"
  host-ns "/proc/1/fd/50"
  host-mac-addr: 02:fe:d4:d5:c4:53
  host-carrier-up: 1
  vhost-fds 40
  tap-fds 39
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:3a:5d:1c:f9
  Device instance: 5
  flags 0x1
    admin-up (0)
  features 0x110008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_F_VERSION_1 (32)
  remote-features 0x33d008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_F_NOTIFY_ON_EMPTY (24)
    VHOST_F_LOG_ALL (26)
    VIRTIO_F_ANY_LAYOUT (27)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_RING_F_EVENT_IDX (29)
    VIRTIO_F_VERSION_1 (32)
    VIRTIO_F_IOMMU_PLATFORM (33)
  Number of RX Virtqueue  1
  Number of TX Virtqueue  1
  Virtqueue (RX) 0
    qsz 1024, last_used_idx 21, desc_next 960, desc_in_use 939
    avail.flags 0x0 avail.idx 960 used.flags 0x1 used.idx 21
    kickfd 42, callfd 41
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show tap tap6
Interface: tap6 (ifindex 10)
  name "nsm-0"
  host-ns "/proc/1/fd/46"
  host-mac-addr: 02:fe:c2:14:ec:dd
  host-carrier-up: 1
  vhost-fds 45
  tap-fds 44
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:2d:7a:0c:17
  Device instance: 6
  flags 0x1
    admin-up (0)
  features 0x110008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_F_VERSION_1 (32)
  remote-features 0x33d008000
    VIRTIO_NET_F_MRG_RXBUF (15)
    VIRTIO_F_NOTIFY_ON_EMPTY (24)
    VHOST_F_LOG_ALL (26)
    VIRTIO_F_ANY_LAYOUT (27)
    VIRTIO_RING_F_INDIRECT_DESC (28)
    VIRTIO_RING_F_EVENT_IDX (29)
    VIRTIO_F_VERSION_1 (32)
    VIRTIO_F_IOMMU_PLATFORM (33)
  Number of RX Virtqueue  1
  Number of TX Virtqueue  1
  Virtqueue (RX) 0
    qsz 1024, last_used_idx 15, desc_next 960, desc_in_use 945
    avail.flags 0x0 avail.idx 960 used.flags 0x1 used.idx 15
    kickfd 47, callfd 46
  Virtqueue (TX) 1
    qsz 1024, last_used_idx 19, desc_next 20, desc_in_use 1
    avail.flags 0x1 avail.idx 20 used.flags 0x0 used.idx 20
    kickfd 48, callfd -1
```

Access the network namespace of the `tap5` peer and `tap6` peer:
* `/proc/1/fd/50`: network namespace file (`host-ns`) of the `tap5` peer
* `/proc/1/fd/46`: network namespace file (`host-ns`) of the `tap6` peer
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- nsenter --net=/proc/1/fd/50 bash
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- nsenter --net=/proc/1/fd/46 bash
```

List the VPP interfaces with metrics and index:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface
              Name               Idx    State  MTU (L3/IP4/IP6/MPLS)     Counter          Count    
tap5                              10     up     1500/1500/1500/1500 rx packets                    19
                                                                    rx bytes                    1922
                                                                    tx packets                    16
                                                                    tx bytes                    1456
                                                                    drops                          1
                                                                    ip6                            1
tap6                              11     up     1500/1500/1500/1500 rx packets                    16
                                                                    rx bytes                    1456
                                                                    tx packets                    18
                                                                    tx bytes                    1772
```

To capture traffic inside the vpp forwarder:
* `vppctl pcap trace rx tx max COUNT intfc INTERFACE`: Start capturing traffic
* `vppctl pcap trace off`: Stop trace. You can use `tcpdump -nn -e -r /tmp/rxtx.pcap` to read it or use Wireshark.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc tap5
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc tap6
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
```
