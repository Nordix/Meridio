# Stateless-lb

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/stateless-lb)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/stateless-lb)

## Description

The load balancer is using a user space [Maglev](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf) implementation to load balance traffic among multiple targets.

At Start, the load balancer is subscribing to events from the NSP Service to get notifications about target registration / unregistration in order to update the identifiers in the [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer), the IP rules and the IP routes.

Since the [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer) is running in user space, iptables together with nfqueue are used to bring traffic from kernel space to user space. The [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer) will then add a forwarding mark on the traffic based on [Maglev](https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf) and according to the registered target identifiers, and will return the traffic to the kernel space. Using the forwarding mark, IP rules and IP routes, the traffic will be forwarded to the selected target.

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/load-balancer/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
NSM_NAME | string | Name of the pod | load-balancer
NSM_SERVICE_NAME | string | Name of providing service | load-balancer
NSM_CONNECT_TO | url.URL | url to connect to NSM | unix:///var/lib/networkservicemesh/nsm.io.sock
NSM_DIAL_TIMEOUT | time.Duration | timeout to dial NSMgr | 5s
NSM_REQUEST_TIMEOUT | time.Duration | timeout to request NSE | 15s
NSM_MAX_TOKEN_LIFETIME | time.Duration | maximum lifetime of tokens | 24h
NSM_NSP_SERVICE | string | IP (or domain) and port of the NSP Service | nsp-service:7778
NSM_CONDUIT_NAME | string | Name of the conduit | load-balancer
NSM_TRENCH_NAME | string | Trench the pod is running on | default
NSM_LOG_LEVEL | string | Log level | DEBUG
NSM_NFQUEUE | string | netfilter queue(s) to be used by nfqlb | 0:3
NSM_NFQUEUE_FANOUT | bool | enable fanout nfqueue option | false

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |

## Communication 

Here are all components the stateless-lb is communicating with:

Component | Secured | Method | Description
--- | --- | --- | ---
Spire | TBD | Unix Socket | Obtain and validate SVIDs
NSM | yes (mTLS) | Unix Socket | Register NSE
NSP Service | yes (mTLS) | TCP | Watch configuration. Watch target registry.

An overview of the communications between all components is available [here](resources.md).

## Health check

The health check is provided by the [GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md). The status returned can be `UNKNOWN`, `SERVING`, `NOT_SERVING` or `SERVICE_UNKNOWN`.

Service | Description
--- | ---
NSPCli | Monitor status of the connection to the NSP service
NSMEndpoint | Monitor status of the NSE
Egress | Monitor the frontend availability
Stream | Check if at least 1 stream is serving
Flow | Check if at least 1 flow is serving

## Privileges

To work properly, here are the privileges required by the stateless-lb:

Name | Description
--- | ---
Sysctl: net.ipv4.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv6.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv4.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv4
Sysctl: net.ipv6.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv6
Sysctl: net.ipv4.conf.all.rp_filter=0 | Allow packets to have a source IPv4 address which does not correspond to any routing destination address.
Sysctl: net.ipv4.conf.default.rp_filter=0 | Allow packets to have a source IPv6 address which does not correspond to any routing destination address.
NET_ADMIN | The load balancer configures IP rules and IP routes to steer packets (processed by [nfqueue-loadbalancer program](https://github.com/Nordix/nfqueue-loadbalancer)) to targets. The user space load balancer program relies on [libnetfilter_queue](https://netfilter.org/projects/libnetfilter_queue).
IPC_LOCK | The user space load balancer program uses shared memory.
IPC_OWNER | The user space load balancer program uses shared memory.