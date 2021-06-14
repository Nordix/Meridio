# Overview

## features

- [X] IPv4
- [X] IPv6
- [X] Dual-Stack
- [X] Target Scalability
- [X] Load balancer Scalability
- [X] Runtime configuration (Only VIP support for the moment)
- [X] External connectivity (VLAN)
- [ ] Front-end / BGP Support
- [ ] Operator

## Components

### Runtime configuration

VIPs are one of the properties which can be modified without restarting any resource of Meridio. To achieve this, each component which uses the VIP addresses is watching the configmap (meridio-configuration) using the Kubernetes API, so they will get an event triggered when the configmap is added, updated or deleted. For each event the configmap data are parsed to detect the VIP addresses to add or to remove. On the load-balancer, iptables rules will be added/removed. On the proxy, source based routes will be added/removed. And, On the target, source based routes and IPs on the loopback interface will be added/removed.

Properties:
- [X] VIPs

### Meridio Operator

Repository: [Nordix/Meridio-Operator](https://github.com/Nordix/Meridio-Operator)

Developed using the [Operator SDK](https://sdk.operatorframework.io/), the Meridio Operator is managing, for now, only the trenches, a new custom resource definition (CRD) which contains the VIP addresses and the name of the configmap. The controller of the trenches verifies if the configmap in the namespace equal to the name of the trench exists. If the configmap does not exist, the controller will create a new oen based on the data of the trench. On each update, the data contained in the configmap are verified and updated if they are not corresponding to the one registered in the Trench. 

In addition, a webhook is running to check the validity of the resource applied. So, when a trench is added or updated the list of VIP addresses is verified. An error is returned to the user and the resource is not added/updated if at least one VIP is invalid.

CRDs
- Trench

### Load balancer

The load balancer is using a user space [Maglev](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf) implementation to load balance traffic among multiple targets.

At Start, the load balancer is subscribing to events from the NSP Service to get notifications about target registration / unregistration in order to update the identifiers in the [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer), the IP rules and the IP routes.

Since the [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer) is running in user space, iptables together with nfqueue are used to bring traffic from kernel space to user space. The [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer) will then add a forwarding mark on the traffic based on [Maglev](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf) and according to the registered target identifiers, and will return the traffic to the kernel space. Using the forwarding mark, IP rules and IP routes, the traffic will be forwarded to the selected target.

### Proxy

The proxy allows targets (e.g. TCP application) to be connected to multiple network service instances (e.g. load-balancer) via Network Service Mesh.

For the different traffic flows, this component is used as a bridge for the traffic coming from the services (load-balancer) and as a gateway for the traffic coming from the targets (application). To allow this, all NSM network interfaces are connected to a bridge. In addition, source based routes (with the VIP as source IP) are created to distribute the egress traffic among the network service instances (load balancer instances).

When started, the proxy requests a subnet from the IPAM Service, so each proxy instance will own a subnet. In addition, the proxy subscribes to network service instances events (Creation / Deletion of instances) using the NSM API in order to always be connected to all network service instances (load balancer instances).

Used as a NSE for the target and NSC for the network services (load balancer), the proxy is utilizing a local IPAM to generate IPs in the associated subnet for all requests: incoming requests as NSE from targets and sent requests as NSC to connect to the network service instances and create a full mesh.

Since the proxy receives the target identifiers and IPs included in requests and closes (not supported yet) calls from the targets, the proxy can register or unregister the targets at the NSP service.

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

## Privileges

A Service account has been added on the Load-balancer, proxy and target with the access to watch configmap in their namespace using the Kubernetes API.

Some kernel parameters (sysctl) are required:
- `net.ipv4.conf.all.forwarding` and `net.ipv6.conf.all.forwarding` are required on the load balancer pod.
- `net.ipv4.conf.all.forwarding`, `net.ipv6.conf.all.forwarding`, `net.ipv4.fib_multipath_hash_policy`, `net.ipv6.fib_multipath_hash_policy` and `net.ipv6.conf.all.accept_dad` are required on the proxy pod.
- `net.ipv4.conf.fib_multipath_hash_policy` and `net.ipv6.fib_multipath_hash_policy` are required on the target pod.

The load-balancer, proxy and target containers need `NET_ADMIN` capability added in their security context.

The Meridio Operator needs the create, delete, get, list, patch, update and watch access the configmap and trench resources.

## Demo

The demo instructions are available on [this page](https://github.com/Nordix/Meridio/tree/master/docs/demo).
