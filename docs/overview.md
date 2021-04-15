# Overview

## features

- [X] External connectivity (VLAN)
- [X] IPv4
- [ ] IPv6
- [ ] Dual-Stack
- [ ] Target Scalability (Only scale-out is supported for the moment)
- [X] Load balancer Scalability

## Components

### Load balancer

The load balancer is using a user space [Maglev](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf) implementation to load balance traffic among multiple targets.

On Start, the load balancer is subscribing to events to the NSP Service to get notifications on target registration / unregistration in order to update the identifiers in the [lb program](https://github.com/Nordix/Meridio/tree/master/third_party/lb), the IP rules and the IP routes.

Since the [lb program](https://github.com/Nordix/Meridio/tree/master/third_party/lb) is running on the user space, iptables together with nfqueue are used to bring traffic from kernel space to user space. The [lb program](https://github.com/Nordix/Meridio/tree/master/third_party/lb) will then add a forwarding mark on the traffic based on [Maglev](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf) and registered identifiers, and will return the traffic to the kernel space. Using the forwarding mark, IP rules and IP routes, the traffic will be forwarded to a right target.

### Proxy

The proxy allows targets (e.g. TCP application) to be connected to multiple network service instances (e.g. load-balancer) on Network Service Mesh.

For the traffic flow, this component is used as a bridge for the traffic coming from the services (load-balancer) and as a gateway for the traffic coming from the targets (application). To allow this, all NSM network interfaces are connected to a bridge. In addition, source based routes (with the VIP as source IP) are created to load balancer the egress traffic among the network service instances (load balancer instances).

When started, the proxy requests a subnet to the IPAM Service, so each proxy instance will own a subnet. In addition, the proxy subscribes to network service instances events (Creation / Deletion of instances) using the NSM API in order to always be connected to all network service instances (load balancer instances).

Used as a NSE for the target and NSC for the network services (load balancer), the proxy is consuming the IPAM Service to generate IPs in the associated subnet for all requests: incoming requests as NSE from targets and sent requests as NSC to connect to the network service instances and create a full mesh.

Since the proxy receives the target identifiers and IPs included in requests and closes (not supported yet) calls from the targets, the proxy can then register or unregister the targets using the NSP service.

### Target

The target is a simple NSC requesting a connection to the proxy network service.

On Start, the target adds the VIP to the loopback interface to handle the traffic and generate its identifier which will be included in the extra-context of the connection request to the proxy network service.

Once the connection is established, a source based route is added to ensure traffic with the VIP as source IP is always going back through the proxy.

One of the containers in the target pod is ctraffic. ctraffic is a testing application offering a TCP server listening on port 5000. This testing application can be also used as TCP client from the external host to generate traffic in the system and create a traffic report.

### Services

#### IPAM

In order to avoid IP collisions in the system and ensure a proper IPs distribution, this service is offering some IPAM functionalities that can be consumed using a kubernetes clusterIP service (over the kubernetes primary network). This IPAM Service is a [GRPC](https://grpc.io/) server listening on port 7777.

The specifications of the IPAM Service are written in a proto file accessible [here](https://github.com/Nordix/Meridio/blob/master/api/ipam/ipam.proto).

#### Network Service Platform (NSP)

The Network Service Platform (NSP) Service allows targets to notify their availability or unavailability by sending their IP and key-value pairs (e.g. identifiers). The service is also providing the possibility to receive notifications on registration / unregistration of targets.

This NSP Service is a [GRPC](https://grpc.io/) server listening on port 7778 that can be consumed using a kubernetes clusterIP service (over the kubernetes primary network) 

The specifications of the NSP Service are written in a proto file accessible [here](https://github.com/Nordix/Meridio/blob/master/api/nsp/nsp.proto).

## Diagrams

![Overview](resources/Overview.svg)

## Demo

The demo instructions are available on [this page](https://github.com/Nordix/Meridio/tree/master/docs/demo).
