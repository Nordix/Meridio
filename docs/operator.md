# Operator

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/operator)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/operator)

## Description

TODO

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/ipam/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
 | |

## Command Line 

Command | Action | Default
--- | --- | ---
 | |

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
Kubernetes API | meridio-operator-leader-election-role- event - create, patch
Kubernetes API | Validating Webhook - trenches - create, update
Kubernetes API | Validating Webhook - conduits - create, update
Kubernetes API | Validating Webhook - streams - create, update
Kubernetes API | Validating Webhook - flows - create, update
Kubernetes API | Validating Webhook - vips - create, update
Kubernetes API | Validating Webhook - attractors - create, update
Kubernetes API | Validating Webhook - gateways - create, update
Kubernetes API | Mutating Webhook - gateways - create, update