# IPAM

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/ipam)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/ipam)

## Description

In order to avoid IP collisions in the system and ensure a proper IPs distribution, this service is offering some IPAM functionalities that can be consumed using a kubernetes clusterIP service (over the kubernetes primary network). This IPAM Service is a [GRPC](https://grpc.io/) server listening on port 7777.

The specifications of the IPAM Service are written in a proto file accessible [here](https://github.com/Nordix/Meridio/blob/master/api/ipam/v1/ipam.proto).

### IP/Prefix distribution granularity

The Meridio IPAM distributes IP/Prefixes (always within the trench subnet defined in the configuration by `IPAM_PREFIX_IPV4` and `IPAM_PREFIX_IPV6`) at a few different levels. 

The first one is at the conduit level. Represented in blue (Conduit-A) and in red (Conduit-B) in the picture below, they are allocated automatically by the IPAM by watching the conduit list via the NSP service. The conduit subnet prefix lengths are defined in the configuration by `IPAM_CONDUIT_PREFIX_LENGTH_IPV4` and `IPAM_CONDUIT_PREFIX_LENGTH_IPV6`.

The second one is at the node level. Represented in black in the picture below (1 per node per conduit), they are allocated when the `Allocate` API function is called (note: there is currently no way to unallocate except if the conduit is removed.). The node subnet prefix lengths are defined in the configuration by `IPAM_NODE_PREFIX_LENGTH_IPV4` and `IPAM_NODE_PREFIX_LENGTH_IPV6`.

The third (last one) is at the pod level. Each pod will get assigned a unique IP address with `IPAM_NODE_PREFIX_LENGTH_IPV4` or `IPAM_NODE_PREFIX_LENGTH_IPV6` as prefix length.

![ipam](resources/IPAM.svg)

Picture representing a cluster with 2 nodes (worked-A and worker-B), 2 conduits (Conduit-A and Conduit-B), 4 targets and the corresponding subnets.
* Target-1 is running on worker-A and connected to Conduit-A
* Target-2 is running on worker-A and connected to Conduit-B
* Target-3 is running on worker-B and connected to Conduit-A and Conduit-B
* Target-4 is running on worker-B and connected to Conduit-B

### Data persistence

Running as StatefulSet with a single replica, the IPAM handles restarts and pod deletions by saving the data in a local sqlite stored in a persistent volume requested via a volumeClaimTemplates.

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/ipam/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
IPAM_PORT | int | Port the pod is running the service | 7777
IPAM_DATA_SOURCE | string | Path and file name of the sqlite database | /run/ipam/data/registry.db
IPAM_TRENCH_NAME | string | Trench the pod is running on | 
IPAM_NSP_SERVICE | string | IP (or domain) and port of the NSP Service | 
IPAM_PREFIX_IPV4 | string | IPv4 prefix from which the proxy prefixes will be allocated | 169.255.0.0/16
IPAM_CONDUIT_PREFIX_LENGTH_IPV4 | int | Conduit prefix length which will be allocated | 20
IPAM_NODE_PREFIX_LENGTH_IPV4 | int | node prefix length which will be allocated | 24
IPAM_PREFIX_IPV6 | string | IPv6 prefix from which the proxy prefixes will be allocated | fd00::/48
IPAM_CONDUIT_PREFIX_LENGTH_IPV6 | int | Conduit prefix length which will be allocated | 56
IPAM_NODE_PREFIX_LENGTH_IPV6 | int | node prefix length which will be allocated | 64
IPAM_IP_FAMILY | string | IP family (ipv4, ipv6, dualstack) | dualstack
IPAM_LOG_LEVEL | string | Log level (TRACE, DEBUG, INFO, WARNING, ERROR, FATAL, PANIC) | DEBUG

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

## Health check

TODO

## Privileges

No privileges required.
