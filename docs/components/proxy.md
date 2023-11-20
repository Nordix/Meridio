# Proxy

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/proxy)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/proxy)

## Description

The proxy allows targets (e.g. TCP application) to be connected to multiple network service instances (e.g. stateless-lb) via a single network interface.

To create the full mesh between the proxy and the network service instances, the proxy uses the NSM API to monitor the NSEs, and requests the connection to each of them. For the targets, the proxy acts as a network service with the same network service name + proxy as prefix: `proxy.<conduit-name>.<trench-name>.<namespace>`.

When started, the proxy requests a subnet from the IPAM Service, so each proxy instance will own a unique subnet and will allocate IPs of targets and Network service instances based on it. Since each proxy has a unique subnet, the network service instances will easily find the correct path to the target via the default routes.

From the network service instances side, the proxy acts as a bridge, so the network service instances can access each individual target via their IPs. From the target side, the proxy acts as a router/gateway, the outgoing traffic of the target. Since it acts as a router/gateway, the proxy is creation source based routes to distribute the outgoing traffic among the network service instances.

Note: Currently the proxy support only 1 conduit.

![Proxy](../resources/Proxy.svg)

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/proxy/internal/config/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
NSM_NAME | string | Name of the pod | proxy
NSM_SERVICE_NAME | string | Name of the Network Service | proxy
NSM_CONNECT_TO | url.URL | url to connect to NSM | unix:///var/lib/networkservicemesh/nsm.io.sock
NSM_DIAL_TIMEOUT | time.Duration | timeout to dial NSMgr | 5s
NSM_REQUEST_TIMEOUT | time.Duration | timeout to request NSE | 15s
NSM_MAX_TOKEN_LIFETIME | time.Duration | maximum lifetime of tokens | 24h
NSM_IPAM_SERVICE | string | IP (or domain) and port of the IPAM Service | ipam-service:7777
NSM_HOST | string | Host name the proxy is running on | 
NSM_NETWORK_SERVICE_NAME | string | Name of the network service the proxy request the connection | load-balancer
NSM_NAMESPACE | string | Namespace the pod is running on | default
NSM_TRENCH | string | Trench the pod is running on | default
NSM_CONDUIT | string | Name of the conduit | load-balancer
NSM_NSP_SERVICE_NAME | string | IP (or domain) of the NSP Service | nsp-service
NSM_NSP_SERVICE_PORT | int | port of the NSP Service | 7778
NSM_IP_FAMILY | string | ip family | dualstack
NSM_LOG_LEVEL | string | Log level | DEBUG

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |

## Communication 

Here are all components the proxy is communicating with:

Component | Secured | Method | Description
--- | --- | --- | ---
Spire | TBD | Unix Socket | Obtain and validate SVIDs
NSM | yes (mTLS) | Unix Socket | Request/Close connections. Register NSE.
NSP Service | yes (mTLS) | TCP | Watch configuration
IPAM Service | yes (mTLS) | TCP | Allocate/Release IPs

An overview of the communications between all components is available [here](resources.md).

## Health check

The health check is provided by the [GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md). The status returned can be `UNKNOWN`, `SERVING`, `NOT_SERVING` or `SERVICE_UNKNOWN`.

Service | Description
--- | ---
Liveness | A unique service to be used by liveness probe to return status, can aggregate other lesser services
Readiness | A unique service to be used by readiness probe to return status, can aggregate other lesser services

Service | Probe | Description
--- | --- | ---
IPAMCli | Readiness | Monitor status of the connection to the IPAM service
NSPCli | Readiness | Monitor status of the connection to the NSP service
NSMEndpoint | Readiness,Liveness | Monitor status of the NSE
Egress | Readiness | Check if at least 1 stateless-lb-frontend is connected

## Privileges

To work properly, here are the privileges required by the proxy:

Name | Description
--- | ---
Sysctl: net.ipv4.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv6.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv6.conf.all.accept_dad=0 | Disable DAD (Duplicate Address Detection)
Sysctl: net.ipv4.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv4
Sysctl: net.ipv6.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv6
Sysctl: net.ipv4.conf.all.rp_filter=0 | Allow packets to have a source IPv4 address which does not correspond to any routing destination address.
Sysctl: net.ipv4.conf.default.rp_filter=0 | Allow packets to have a source IPv6 address which does not correspond to any routing destination address.
NET_ADMIN | The proxy creates IP rules, IP routes, bridge interfaces and modifies NSM interfaces to link them to bridge interfaces.
