# Frontend

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/frontend)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/frontend)

## Description

The frontend makes it possible to attract external traffic to Meridio via a secondary network.

The external interface to be used for external connectivity must be provided to the frontend.  
One way to achieve this is to rely on NSM which through a NSC container can install a VLAN capable interface into the particular frontend POD. The master interface residing in the host network namespace, the VLAN ID and the IP network NSM shall use to allocate IP address to the external interface must be configured to get consumed by the Remote VLAN NSE.  
Alternatively, the external interface can be provided using Multus in which case no NSC or Remote VLAN NSE is required, and IP address allocation can be taken care of by a suitable IPAM CNI plugin (configured in the Network Attachment Definition).


When started, the frontend installs src routing rules for each configured VIP address, then configures and spins off a [BIRD](https://bird.network.cz/) routing program instance providing for external connectivity. The bird routing suite is restricted to the external interface. The frontend uses [birdc](https://bird.network.cz/?get_doc&v=20&f=bird-4.html) for both monitoring and changing BIRD configuration.

BGP protocol with optional BFD supervision and Static+BFD setup are supported at the moment. Since they lack inherent neighbor discovery mechanism, the external gateway IP addresses must be configured.
In case of BGP a next-hop route for each VIP address gets announced by the protocol to its external peer advertising the frontend IP as next-hop, thus attracting external traffic to the frontend. While from the external BGP peer at least one next-hop route is expected to be utilized by the VIP src routing to steer egress traffic. The external BGP peer can decide to announce a default route or a set of network routes.

Both ingress and egress traffic traverse a frontend POD (not necessarily the same).

Currently the frontend is collocated with the load balancer, hence reside in the same POD. A load balancer relies on the collocated frontend to forward egress traffic, and the other way around to handle ingress traffic. There's no direct control plane interaction between the two though.

#### External gateway router

The external peer a frontend is intended to connect with must be configured separately as it is outside the scope of Meridio.

Some generic pointers to setup the external router side (focusing on BGP):  
The external peer must be part of the same (secondary) network and subnet as the external interface of the connected frontend. NSM _exclude prefixes_ functionality can be used to prevent the IPAM in Remote VLAN NSE assigning IPs that have been allocated to external peers. (On the other hand, the IPAM starts assigning IPs from the start of the range, thus in development environments it might be sufficent to pick IPs from the end of the range to configure external peers.)  
To avoid the need of having to configure all the possible IPs the frontends might use to connect to an external BGP router, it's worth considering passive BGP peering on the router side.  
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
NFE_TABLE_ID | int | Start ID of the two consecutive OS Kernel routing tables BIRD syncs the routes with | 4096
NFE_ECMP | bool | Enable ECMP towards next-hops of avaialble gateways | false
NFE_DROP_IF_NO_PEER | bool | Install default blackhole route with high metric into routing table TableID | true
NFE_LOG_BIRD | bool | Add important bird log snippets to our log | false
NFE_NAMESPACE | string | Namespace the pod is running on | default
NFE_NSP_SERVICE | string | IP (or domain) and port of the NSP Service | nsp-service-trench-a:7778
NFE_TRENCH_NAME | string | Name of the Trench the frontend is associated with | default
NFE_ATTRACTOR_NAME | string | Name of the Attractor the frontend is associated with | default
NFE_LOG_LEVEL | string | Log level | DEBUG
NFE_NSP_ENTRY_TIMEOUT | time.Duration | Timeout of entries registered in NSP | 30s
NFE_GRPC_KEEPALIVE_TIME | time.Duration | gRPC keepalive timeout | 30s
NFE_GRPC_MAX_BACKOFF | time.Duration |  Upper bound on gRPC connection backoff delay | 5s
NFE_DELAY_CONNECTIVITY | time.Duration | Delay between routing suite checks with connectivity | 1s
NFE_DELAY_NO_CONNECTIVITY | time.Duration | Delay between routing suite checks without connectivity | 3s
NFE_MAX_SESSION_ERRORS | int | Max session errors when checking routing suite until denounce | 5
NFE_METRICS_ENABLED | bool | Enable the metrics collection | false
NFE_METRICS_PORT | int | Specify the port used to expose the metrics | 2224
NFE_LB_SOCKET | url.URL | LB socket to connect to | unix:///var/lib/meridio/lb.sock

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |

## Communication 

Here are all components the frontend is communicating with:

Component | Secured | Method | Description
--- | --- | --- | ---
Spire | TBD | Unix Socket | Obtain and validate SVIDs
NSP Service | yes (mTLS) | TCP | Watch configuration. Register/Unregister target (Advertise its readiness to the NSP target registry)
Gateways | / | / | Routing protocol
Kubernetes API | TDB | TCP | Watch the secrets for BGP authentication
LB | yes (mTLS) | Unix socket | Watch internal connectivity status of collocated stateless-lb

An overview of the communications between all components is available [here](resources.md).

## Health check

The health check is provided by the [GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md). The status returned can be `UNKNOWN`, `SERVING`, `NOT_SERVING` or `SERVICE_UNKNOWN`.

Service | Description
--- | ---
Readiness | A unique service to be used by readiness probe to return status, can aggregate other lesser services

Service | Probe | Description
--- | --- | ---
NSPCli | Readiness | Monitor status of the connection to the NSP service
Egress | Readiness | Monitor the gateways connectivity

## Privileges

To work properly, here are the privileges required by the frontend:

Name | Description
--- | ---
Sysctl: net.ipv4.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv6.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv4.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv4
Sysctl: net.ipv6.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv6
Sysctl: net.ipv4.conf.all.rp_filter=0 | Allow packets to have a source IPv4 address which does not correspond to any routing destination address.
Sysctl: net.ipv4.conf.default.rp_filter=0 | Allow packets to have a source IPv6 address which does not correspond to any routing destination address.
Sysctl: net.ipv4.ip_local_port_range='49152 65535' | The source port of BFD Control packets must be in the IANA approved range 49152-65535
NET_ADMIN | The frontend creates IP rules to handle outbound traffic from VIP sources. BIRD interacts with kernel routing tables.
NET_BIND_SERVICE | Allows BIRD to bind to privileged ports depending on the config (for example to BGP port 173).
NET_RAW | Allows BIRD to use the SO_BINDTODEVICE socket option.
Kubernetes API | fes-role - secrets - watch
