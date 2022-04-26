# Overview

The Operator requires only Kubernetes API privileges.

### Roles

Resource | Verbs | Description
--- | --- | ---
daemonsets | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
deployments | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
statefulsets | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
configmaps | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
serviceaccounts | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
services | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
rolebindings | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
roles | create, delete, get, list, patch, update, watch | meridio-operator-manager-role
trenches | get, list, update, watch | meridio-operator-manager-role
conduits | get, list, update, watch | meridio-operator-manager-role
streams | get, list, update, watch | meridio-operator-manager-role
flows | get, list, update, watch | meridio-operator-manager-role
vips | get, list, update, watch | meridio-operator-manager-role
attractors | get, list, update, watch | meridio-operator-manager-role
gateways | get, list, update, watch | meridio-operator-leader-election-role
configmaps | get, list, watch, create, update, patch, delete | meridio-operator-leader-election-role
leases | get, list, watch, create, update, patch, delete | meridio-operator-leader-election-role
event | create, patch | meridio-operator-leader-election-role

### Validating Webhook

Resource | Operations
--- | ---
trenches | CREATE, UPDATE
conduits | CREATE, UPDATE
streams | CREATE, UPDATE
flows | CREATE, UPDATE
vips | CREATE, UPDATE
attractors | CREATE, UPDATE
gateways | CREATE, UPDATE

### Mutating Webhook

Resource | Operations
--- | ---
gateways | CREATE, UPDATE