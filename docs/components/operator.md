# Operator

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/operator)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/operator)

## Description

TODO

### Resource Template

TODO

### Resource Management

The resource requirements of containers making up the PODs to be spawned by the Operator can be controlled by annotating the respective custom resource.
As of now, annotation of __Trench__, __Attractor__ and __Conduit__ resources are supported, because these are responsible for creating POD resources.

A Trench can be annotated to set resource requirements by following the example below.
```
apiVersion: meridio.nordix.org/v1alpha1
kind: Trench
metadata:
  name: trench-a
  annotations:
    resource-template: "small"
spec:
  ip-family: dualstack
```

For each container making up a specific custom resource (e.g. Trench) the annotation value for key _resource-template_ is interpreted as the name of a resource requirements template. Such templates are defined per container, and are to be specified before building the Operator.

As an example some [templates](https://github.com/Nordix/Meridio/tree/master/config/manager/resource_requirements/) are included for each container out-of-the-box. But they are not verified to fit any production use cases, and can be overridden at will. (A template is basically a kubernetes [core v1 ResourceRequirements](https://pkg.go.dev/k8s.io/api@v0.22.2/core/v1#ResourceRequirements) block with name.)

The Operator looks up the templates based on the annotation value for each container contributing to the particular custom resource. If a template is missing for a container, then deployment proceeds without setting resource requirements for the container at issue. Otherwise the related resources will be deployed by importing the respective resource requirements from the matching templates.

Updating the annotation of a custom resource is possible. Changes will be applied by kubernetes according to the
Update Strategy of the related resources. Service disturbances and outages are to be expected.


## Configuration 

Environment variable | Type | Description | Default
--- | --- | --- | ---
 | |

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |
--debug | Prints meridio-version, unix-time, network-interfaces, rules, route, neighbors, system information, and environment-variables in a json format |

## Communication 

Here are all components the operator is communicating with:

Component | Secured | Method | Description
--- | --- | --- | ---
Spire | TBD | Unix Socket | Obtain and validate SVIDs
Kubernetes API | TBD | TCP | Apply/Update/Delete/Watch resources

An overview of the communications between all components is available [here](resources.md).

## Health check

The health check is provided by the [GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md). The status returned can be `UNKNOWN`, `SERVING`, `NOT_SERVING` or `SERVICE_UNKNOWN`.

TODO

## Privileges

To work properly, here are the privileges required by the operator:

Name | Description
--- | ---
Kubernetes API | meridio-operator-manager-role - daemonsets - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - deployments - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - statefulsets - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - configmaps - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - serviceaccounts - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - services - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - rolebindings - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - roles - create, delete, get, list, patch, update, watch
Kubernetes API | meridio-operator-manager-role - trenches - get, list, update, watch
Kubernetes API | meridio-operator-manager-role - conduits - get, list, update, watch
Kubernetes API | meridio-operator-manager-role - streams - get, list, update, watch
Kubernetes API | meridio-operator-manager-role - flows - get, list, update, watch
Kubernetes API | meridio-operator-manager-role - vips - get, list, update, watch
Kubernetes API | meridio-operator-manager-role - attractors - get, list, update, watch
Kubernetes API | meridio-operator-leader-election-role - gateways - get, list, update, watch
Kubernetes API | meridio-operator-leader-election-role - configmaps - get, list, watch, create, update, patch, delete 
Kubernetes API | meridio-operator-leader-election-role - leases - get, list, watch, create, update, patch, delete
Kubernetes API | meridio-operator-leader-election-role - event - create, patch
Kubernetes API | Validating Webhook - trenches - create, update
Kubernetes API | Validating Webhook - conduits - create, update
Kubernetes API | Validating Webhook - streams - create, update
Kubernetes API | Validating Webhook - flows - create, update
Kubernetes API | Validating Webhook - vips - create, update
Kubernetes API | Validating Webhook - attractors - create, update
Kubernetes API | Validating Webhook - gateways - create, update
Kubernetes API | Mutating Webhook - gateways - create, update