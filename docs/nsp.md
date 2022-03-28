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

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |

## Communication 

Component | Secured | Method
--- | --- | ---
Spire | TBD | Unix Socket
Kubernetes API | TBD | TCP

## Health check

TODO

## Privileges

Name | Description
--- | ---
Kubernetes API | The NSP watches a configmap to get updated about the configuration of the trench
