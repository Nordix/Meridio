# Load-balancer

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/load-balancer)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/load-balancer)

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

Component | Secured | Method
--- | --- | ---
Spire | TBD | Unix Socket
NSM | yes (mTLS) | Unix Socket
NSP Service | yes (mTLS) | TCP

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
