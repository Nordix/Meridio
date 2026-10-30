# Network Service Platform (NSP)

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/nsp)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/nsp)

## Description

The Network Service Platform (NSP) is a [GRPC](https://grpc.io/) server listening on port 7778 (configurable) that can be consumed using a kubernetes clusterIP service (over the kubernetes primary network).

The NSP has two responsabilities: 
- Watch the configuration changes and propagate the updates to every Meridio component.
- Manage the targets.

### Configuration Manager

Since Meridio can handle configuration changes during runtime, the components (Load-balancer, proxy, targets...) need to get notification about the changes.

Meridio uses a configmap to store the configuration of all resources (trench, conduits, streams...) and their properties. To get notified about the changes in the configmap, the NSP uses the Kubernetes API to watch the configmap. Once the modifications are received, the NSP will forward all updated resources to the clients.

Clients of the NSP configuration manager can use multiple different functions to watch all types of resources and filter them if needed via the parameter of each function. The proto file of the configuration manager can be found [here](https://github.com/Nordix/Meridio/blob/master/api/nsp/v1/configurationmanager.proto).

### Target Registry

The NSP Service allows targets to notify their availability or unavailability by sending their IPs, stream, status and a key-value pair (e.g. identifiers). The service is also providing the possibility to receive notifications on registration / unregistration of targets via a watch function. The proto file of the target registry can be found [here](https://github.com/Nordix/Meridio/blob/master/api/nsp/v1/targetregistry.proto).

In order to avoid "ghost" targets if a target cannot unregister itself from the NSP service (Node crash, ungraceful terminaison of a target...), the target registry removes the targets which are not refreshing their registration. To do so, a target has to update its entry by calling the Register function regularly.

### Target Registry internals

#### Layers

The target registry is built from three layers (wired in [cmd/nsp/main.go](https://github.com/Nordix/Meridio/blob/master/cmd/nsp/main.go)). Each layer only calls the one below it:

Layer | Package | Role | Lock
--- | --- | --- | ---
gRPC server | [pkg/nsp](https://github.com/Nordix/Meridio/tree/master/pkg/nsp) | gRPC handlers for Register, Unregister and Watch | none
Keepalive | [pkg/nsp/registry/keepalive](https://github.com/Nordix/Meridio/tree/master/pkg/nsp/registry/keepalive) | Removes targets that are not refreshed within `NSP_ENTRY_TIMEOUT` | keepalive lock
SQLite registry | [pkg/nsp/registry/sqlite](https://github.com/Nordix/Meridio/tree/master/pkg/nsp/registry/sqlite) | Stores the targets and keeps the list of watchers | registry lock

A watcher ([pkg/nsp/registry/common](https://github.com/Nordix/Meridio/tree/master/pkg/nsp/registry/common)) belongs to one Watch call. It holds the client's target selector and a mailbox with a single slot, protected by its own watcher lock.

gRPC runs every call in its own goroutine, so Register, Unregister and Watch calls run in parallel. The locks serialize them.

#### Clients

* Targets (TAPA) register their target and refresh it periodically. After each refresh, they open a short Watch to verify that the registry holds the target as enabled, read one target list, and close the Watch.
* Frontends register themselves as frontend targets.
* Load balancers keep long-lived Watches: one per stream for the enabled targets of that stream, and one for the frontend targets.

#### Register and Unregister

Register and Unregister are synchronous: the reply is sent after the whole call has completed.

```
Register
 └ keepalive: holds the keepalive lock for the whole call
    ├ a target that was not registered before is stored as disabled
    └ sqlite: holds the registry lock
       ├ new target:             store it, remove conflicting targets, broadcast
       ├ unchanged refresh:      return (no write, no broadcast)
       └ changed target:         store it, remove conflicting targets, broadcast
 └ keepalive: (re)arm the target's expiry timer
```

A broadcast reads all targets from the database and puts the current target list into the mailbox of every watcher. It runs inside the write, under the registry lock. Putting a list into a mailbox never blocks: a newer list replaces an older one that was not yet sent, and each watcher filters the list with its own selector.

An unchanged refresh skips the broadcast, since the watchers already have the same target list. If a write succeeded but no broadcast has succeeded since (for example because of a database error), the next refresh broadcasts even if unchanged.

When an expiry timer fires, the keepalive layer takes the keepalive lock and removes the target, which broadcasts like Unregister.

#### Watch

Watch is a long-running server-streaming call:

```
Watch
 └ keepalive: passes the call through, no lock
    └ sqlite: holds the registry lock briefly:
         create a watcher, add it to the watcher list,
         put the current target list into its mailbox
 loop, holding no registry lock:
    mailbox has a target list → send it to the client
 until the client goes away or a send fails
 stop the watcher: close its mailbox, remove it from the watcher list
```

Sending to the client is asynchronous to the broadcast: each Watch sends from its own goroutine. A slow client only delays its own stream. Since the mailbox keeps only the latest list, a slow client skips intermediate lists but always gets the newest one.

#### Lock order

Lock | Taken by | Order
--- | --- | ---
Keepalive lock | Register, Unregister, expiry | 1
Registry lock | Database access, watcher list | 2
Watcher lock | Putting a list into a mailbox, stopping the watcher | 3

The locks are always taken in this order, never in reverse. Watch and stopping a watcher do not take the keepalive lock. Stopping a watcher takes the watcher lock and releases it before taking the registry lock to remove the watcher from the list.

### Data persistence

Running as StatefulSet with a single replica, the NSP handles restarts and pod deletions by saving the data in a local sqlite stored in a persistent volume requested via a volumeClaimTemplates.

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/nsp/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
NSM_NAMESPACE | string | Namespace the pod is running on | default
NSM_PORT | string | Trench the pod is running on | 7778
NSM_CONFIG_MAP_NAME | string | Name of the ConfigMap containing the configuration | meridio-configuration
NSM_DATASOURCE | string | Path and file name of the sqlite database | /run/nsp/data/registry.db
NSM_LOG_LEVEL | string | Log level | DEBUG
NSM_ENTRY_TIMEOUT | time.Duration | Timeout of the entries | 60s
NSM_GRPC_PROBE_RPC_TIMEOUT | time.Duration | RPC timeout of internal gRPC health probe | 1s

## Command Line 

Command | Action | Default 
--- | --- | ---
--help | Display a help describing |
--version | Display the version |
--debug | Prints meridio-version, unix-time, network-interfaces, rules, route, neighbors, system information, and environment-variables in a json format |

## Communication 

Here are all components the nsp is communicating with:

Component | Secured | Method | Description
--- | --- | --- | ---
Spire | TBD | Unix Socket | Obtain and validate SVIDs
Kubernetes API | TBD | TCP | Watch the trench configmap

An overview of the communications between all components is available [here](resources.md).

## Health check

The health check is provided by the [GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md). The status returned can be `UNKNOWN`, `SERVING`, `NOT_SERVING` or `SERVICE_UNKNOWN`.

Service | Description
--- | ---
Liveness | A unique service to be used by liveness probe to return status, can aggregate other lesser services
Readiness | A unique service to be used by readiness probe to return status, can aggregate other lesser services

Service | Probe | Description
--- | --- | ---
NSP | Liveness | Monitor status of the server

## Privileges

To work properly, here are the privileges required by the nsp:

Name | Description
--- | ---
Kubernetes API | nsp-role - configmaps - watch
