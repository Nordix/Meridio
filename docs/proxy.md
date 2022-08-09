# Proxy

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/proxy)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/proxy)

## Description

The proxy allows targets (e.g. TCP application) to be connected to multiple network service instances (e.g. load-balancer) via a single network interface.

To create the full mesh between the proxy and the network service instances, the proxy uses the NSM API to monitor the NSEs, and requests the connection to each of them. For the targets, the proxy acts as a network service with the same network service name + proxy as prefix: `proxy.<conduit-name>.<trench-name>.<namespace>`.

When started, the proxy requests a subnet from the IPAM Service, so each proxy instance will own a unique subnet and will allocate IPs of targets and Network service instances based on it. Since each proxy has a unique subnet, the network service instances will easily find the correct path to the target via the default routes.

From the network service instances side, the proxy acts as a bridge, so the network service instances can access each individual target via their IPs. From the target side, the proxy acts as a router/gateway, the outgoing traffic of the target. Since it acts as a router/gateway, the proxy is creation source based routes to distribute the outgoing traffic among the network service instances.

Note: Currently the proxy support only 1 conduit.

<img src="resources/Proxy.svg" width="100%">

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

Component | Secured | Method
--- | --- | ---
Spire | TBD | Unix Socket
NSM | yes (mTLS) | Unix Socket
NSP Service | yes (mTLS) | TCP
IPAM Service | yes (mTLS) | TCP

## Health check

TODO

## Privileges

Name | Description
--- | ---
Sysctl: net.ipv6.conf.all.forwarding=1 | 
Sysctl: net.ipv4.conf.all.forwarding=1 | 
Sysctl: net.ipv6.conf.all.accept_dad=0 | 
Sysctl: net.ipv4.fib_multipath_hash_policy=1 | 
Sysctl: net.ipv6.fib_multipath_hash_policy=1 | 
Sysctl: net.ipv4.conf.all.rp_filter=0 | 
Sysctl: net.ipv4.conf.default.rp_filter=0 | 
NET_ADMIN | The proxy creates IP rules, IP routes, bridge interfaces and modifies NSM interfaces to link them to bridge interfaces.
