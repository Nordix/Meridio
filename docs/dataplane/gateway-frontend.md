# Gateway - Frontend

## nsm-vlan 

### vlan-id 1 to 4094 - NSM VPP-Forwarder v1.7.0 to latest 

Here is the implementation of the network using the nsm-vlan attractor type with vlan-id set to a value from 1 to 4094 and with NSM the VPP-Forwarder from v1.7.0 to latest. This example use eth0 as base interface and 100 as vlan-id, we will also consider traffic used is `20.0.0.1/32` and `2000::1/128` as VIPs and `4000` as destination port.

![Dataplane-nsm-vlan-100-v1.7.0](../resources/Dataplane-nsm-vlan-100-v1.7.0.svg)

List the kernel interfaces of the host
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

You can capture VLAN traffic with VLAN 100 and 4000 as destination port with:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- tcpdump -nn -i eth0 -e 'dst port 4000 and (vlan 100)'
```

List the VPP interfaces:
* host-eth0: VPP interface representing the Linux kernel base interface `eth0` of the host
* host-eth0.100: VPP VLAN (ID: 100) interface based on `eth0`
* 7271263: VPP bridge-domain interface bridging `host-eth0.100` and `tap0`
* tap0: VPP vETH (might also be tapV2). Its peer interface is the Linux kernel interface in the `stateless-lb-frontend` (ext-vlan0)
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface addr
host-eth0 (up):
  L3 172.18.0.2/16
host-eth0.100 (up):
  L2 bridge bd-id 7271263 idx 1 shg 0
tap0 (up):
  L2 bridge bd-id 7271263 idx 1 shg 1
```
Note: `vppctl show mode` can also be used.

List the VPP bridges:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show bridge-domain
  BD-ID   Index   BSN  Age(min)  Learning  U-Forwrd   UU-Flood   Flooding  ARP-Term  arp-ufwd Learn-co Learn-li   BVI-Intf 
 7271263    1      0     off        on        on       flood        on       off       off        2    16777216     N/A 
```

To find the peer interface of a TAP interface in VPP, you can do it by listing the TAP interfaces and finding the one that has the same `host-mac-addr` property as the MAC address on the Linux Kernel interface.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show tap tap0
Interface: tap0 (ifindex 2)
  name "ext-vlan0"
  host-ns "/proc/1/fd/19"
  host-mac-addr: 02:fe:1c:50:da:6c
  host-carrier-up: 1
  vhost-fds 15
  tap-fds 14
  gso-enabled 0
  csum-enabled 0
  packet-coalesce 0
  packet-buffering 0
  Mac Address: 02:fe:7b:f9:4e:98
  Device instance: 0
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
    qsz 1024, last_used_idx 2793, desc_next 640, desc_in_use 919
    avail.flags 0x0 avail.idx 3712 used.flags 0x1 used.idx 2793
    kickfd 17, callfd 16
  Virtqueue (TX) 1
    qsz 1024, last_used_idx 2765, desc_next 718, desc_in_use 1
    avail.flags 0x1 avail.idx 2766 used.flags 0x0 used.idx 2766
    kickfd 18, callfd -1
```

Access the network namespace of the `tap0` peer:
* `/proc/1/fd/19`: network namespace file (`host-ns`) of the `tap0` peer
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- nsenter --net=/proc/1/fd/19 bash
```

List the VPP interfaces with metrics and index:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show interface
              Name               Idx    State  MTU (L3/IP4/IP6/MPLS)     Counter          Count     
host-eth0                         1      up     1454/1454/1454/1454 rx packets                222890
                                                                    rx bytes               272031148
                                                                    tx packets                  3356
                                                                    tx bytes                  283573
                                                                    drops                     219563
                                                                    ip4                        72607
                                                                    ip6                           12
host-eth0.100                     2      up           0/0/0/0       rx packets                  3284
                                                                    rx bytes                  274240
                                                                    tx packets                  3312
                                                                    tx bytes                  276867
tap0                              3      up     1454/1454/1454/1454 rx packets                  3312
                                                                    rx bytes                  263619
                                                                    tx packets                  3284
                                                                    tx bytes                  261104
```

To capture traffic inside the vpp forwarder:
* `vppctl trace add TRACE COUNT`: capture any traffic from TRACE. List the supported trace: https://fd.io/docs/vpp/v2101/gettingstarted/progressivevpp/traces.html#add-trace
* `vppctl clear trace`: Clear trace buffer
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl trace add virtio-input 100
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl show trace
```

Alternative to capture traffic inside the vpp forwarder:
* `vppctl pcap trace rx tx max COUNT intfc INTERFACE`: Start capturing traffic
* `vppctl pcap trace off`: Stop trace. You can use `tcpdump -nn -e -r /tmp/rxtx.pcap` to read it or use Wireshark.
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace rx tx max 100 intfc tap0
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- vppctl pcap trace off
```

To capture the VLAN traffic using vlan 100 with 4000 as port:
```sh
$ kubectl exec -it forwarder-vpp-worker-a -n nsm -- tcpdump -nn -i eth0 -e 'port 4000 and (vlan 100)'
```

List the interfaces in the stateless-lb-frontend:
* ext-vlan0: peer of VPP tap0 interface
``` sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- ip a show dev ext-vlan0
3: ext-vlan0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1454 qdisc mq state UNKNOWN group default qlen 1000
    link/ether 02:fe:1c:50:da:6c brd ff:ff:ff:ff:ff:ff
    inet 169.254.100.2/24 brd 169.254.100.255 scope global ext-vlan0
       valid_lft forever preferred_lft forever
    inet6 100:100::2/64 scope global nodad 
       valid_lft forever preferred_lft forever
    inet6 fe80::fe:1cff:fe50:da6c/64 scope link 
       valid_lft forever preferred_lft forever
```
note: if ip command is not available, it is also possible to use these commands:
* List the interfaces: `cat /proc/net/dev`
* Get the MAC address of an interface: `cat /sys/class/net/ext-vlan0/address`
* Get the ARP table: `cat /proc/net/arp`

To capture traffic inside the stateless-lb-frontend:
```sh
$ kubectl exec -it stateless-lb-frontend-attractor-a-1-ghi -n red -- tcpdump -nn -i any port 4000
```

### vlan-id 0 - NSM VPP-Forwarder v1.7.0 to latest

Here is the implementation of the network using the nsm-vlan attractor type with vlan-id set to 0 and with the NSM VPP-Forwarder from v1.7.0 to latest.

TODO

### vlan-id 1 to 4094 - NSM VPP-Forwarder v1.1.0 to v1.6.1

Here is the implementation of the network using the nsm-vlan attractor type with vlan-id set to a value from 1 to 4094 and with NSM the VPP-Forwarder from v1.1.0 to v1.6.1.

TODO

### vlan-id 0 - NSM VPP-Forwarder v1.4.0 to v1.6.1

Here is the implementation of the network using the nsm-vlan attractor type with vlan-id set to 0 and with the NSM VPP-Forwarder from v1.4.0 to v1.6.1.

TODO

## network-attachment

TODO
