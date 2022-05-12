# Front-End

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/frontend)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/frontend)

## Description

The frontend makes it possible to attract external traffic to Meridio via a secondary network.

The external interface to be used for external connectivity must be provided to the frontend.  
Currently this is achieved by relying on NSM that through a NSC container installs a VLAN interface into the particular frontend POD. The trunk interface (i.e. secondary network), the VLAN ID and the IP subnet of the VLAN network NSM will use to assign IP address to the external interface can be set during deployment to get consumed by the VLAN NSE.

When started, the frontend installs src routing rules for each configured VIP address, then configures and spins off a [BIRD](https://bird.network.cz/) routing program instance providing for external connectivity. The bird routing suite is restricted to the external interface. The frontend uses [birdc](https://bird.network.cz/?get_doc&v=20&f=bird-4.html) for both monitoring and changing BIRD configuration.

Only BGP protocol is supported at the moment, which lacks inherent neighbor discovery mechanism. Thus the external gateway IP addresses must be configured during deployment time (or runtime once Meridio operator support is implemented).  
A next-hop route for each VIP address gets announced by BGP to its external peer advertising the frontend IP as next-hop, thus attracting external traffic to the frontend. While from the external BGP peer a default next-hop route is expected that will be utilized by the VIP src routing to steer egress traffic. Both ingress and egress traffic traverse a frontend POD (not necessarily the same).

Currently the frontend is collocated with the load balancer, hence reside in the same POD. A load balancer relies on the collocated frontend to forwarder egress traffic, and the other way around to handle ingress traffic. There's no direct communication between the two though.

For setting external connectivity related parameters for Meridio refer to the vlan options in the values file and the [install guide](https://github.com/Nordix/Meridio/tree/master/docs/demo#meridio).

#### External gateway router

The external peer a frontend is intended to connect with must be configured separately as it is outside the scope of Meridio.

Some generic pointers to setup the external router side:  
The external peer must be part of the same (secondary) network and subnet as the external interface of the connected frontend. At the moment the IPAM assigning external IPs to frontends has no means to reserve IPs (e.g. to be used by external peers). However the IPAM starts assigning IPs from the start of the range, thus it is recommended to pick IPs from the end of the range to configure external peers. To avoid the need of having to configure all the possible IPs the frontends might use to connect to an external BGP router, it's worth considering passive BGP peering on the router side.  
By default Meridio side uses BGP AS 8103 and assumes AS 4248829953 on the gateway router side, while default BGP port for both side is 10179.

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/front-end/internal/env/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
NFE_VRRPS | []string | VRRP IP addresses to be used as next-hops for static default routes | 
NFE_EXTERNAL_INTERFACE | string | External interface to start BIRD on | ext-vlan
NFE_BIRD_CONFIG_PATH | string | Path to place bird config files | /etc/bird
NFE_LOCAL_AS | string | Local BGP AS number | 8103
NFE_REMOTE_AS | string | Local BGP AS number | 4248829953
NFE_BGP_LOCAL_PORT | string | Local BGP server port | 10179
NFE_BGP_REMOTE_PORT | string | Remote BGP server port | 10179
NFE_BGP_HOLD_TIME | string | Seconds to wait for a Keepalive message from peer before considering the connection stale | 3
NFE_TABLE_ID | int | OS Kernel routing table ID BIRD syncs the routes with | 4096
NFE_ECMP | bool | Enable ECMP towards next-hops of avaialble gateways | false
NFE_DROP_IF_NO_PEER | bool | Install default blackhole route with high metric into routing table TableID | false
NFE_LOG_BIRD | bool | Add important bird log snippets to our log | false
NFE_NAMESPACE | string | Namespace the pod is running on | default
NFE_NSP_SERVICE | string | IP (or domain) and port of the NSP Service | nsp-service-trench-a:7778
NFE_TRENCH_NAME | string | Name of the Trench the frontend is associated with | default
NFE_ATTRACTOR_NAME | string | Name of the Attractor the frontend is associated with | default
NFE_LOG_LEVEL | string | Log level | DEBUG

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |

## Communication 

Component | Secured | Method
--- | --- | ---
Spire | TBD | Unix Socket
NSP Service | yes (mTLS) | TCP
Gateways | / | /

## Health check

TODO

## Privileges

Name | Description
--- | ---
Sysctl: net.ipv6.conf.all.forwarding=1  | 
Sysctl: net.ipv4.conf.all.forwarding=1 | 
Sysctl: net.ipv4.fib_multipath_hash_policy=1 | 
Sysctl: net.ipv6.fib_multipath_hash_policy=1 | 
Sysctl: net.ipv4.conf.all.rp_filter=0 | 
Sysctl: net.ipv4.conf.default.rp_filter=0 | 
NET_ADMIN | 
