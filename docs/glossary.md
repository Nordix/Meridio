# Glossary

## Meridio

- `TAPA`: Target Access Point Ambassador
- `NSP`: Network Service Platform
- `Target`: Endpoint / Application / Pod using the TAPA to utilize Meridio.
- `Open <stream-name>/<conduit-name>/<trench-name>`: The TAPA connects a trench if not already connected, connects a conduit if not already connected, and registers to the stream. The stream is considered `Opened` in the target.
- `Close <stream-name>/<conduit-name>/<trench-name>`: The TAPA unregisters from the stream, disconnects from the conduit if it was the only stream opened in that conduit, and diconnects from the trench if it was the onle conduit connected. The stream is considered `Closed` in the target.
- `NFQLB`: NFQueue Load Balancer

## Kubernetes

- `CR`/`CO`: Custom Resource / Custom Object

## Networking 

- `IPAM`: IP Address Management
- `ARP`: Address Resolution Protocol
- `NDP`: Neighbor Discovery Protocol
- `LB`: Load Balancer
- `OVS`: Open vSwitch
- `VPP`: Vector Packet Processing
- `VNI`: VLAN/VxLAN Network Identifier
- `VLAN`: Virtual Local Area Network
- `VxLAN`: Virtual eXtensible Local Area Network
- `VIP`: Virtual IP
- `NAT`: Network Address Translation
- `MTU`: Maximum Transmission Unit
- `DNS`: Domain Name System
- `ICMP`: Internet Control Message Protocol
- `ECMP`: Equal-Cost Multi-Path routing
- `AS`: Autonomous System
- `ASN`: Autonomous System Number
- `BGP`: Border Gateway Protocol
- `BFD`: Bidirectional Forwarding Detection
- `SCTP`: Stream Control Transmission Protocol
- `TCP`: Transmission Control Protocol
- `UDP`: User Datagram Protocol

## Network Service Mesh

- `NSM`: Network Service Mesh
- `NS`: Network Service
- `NSC`: Network Service Client
- `NSE`: Network Service Endpoint
- `NSM Request`: An NSM connection will be created
- `NSM Close`: An NSM connection will be deleted
