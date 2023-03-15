# Common Issues

### Stateless-lb-frontend containers are not ready

1. Check the Gateways configuration.
   1. Misconfiguration of the Attractor
   1. Misconfiguration of the Gateways

### All resources are not deployed with a Trench/Conduit/Attractor

1. Does the custom resource name match the limitations ([Trench](../concepts/trench.md) / [Conduit](../concepts/conduit.md) / [Attractor](../concepts/attractor.md)).
2. Check the logs in the Operator for errors.

### Containers are not ready

1. Check the health probes in the logs ([Operator](../components/operator.md#health-check) / [TAPA](../components/tapa.md#health-check) / [Stateless-lb](../components/stateless-lb.md#health-check) / [Frontend](../components/frontend.md#health-check) / [NSP](../components/nsp.md#health-check) / [IPAM](../components/ipam.md#health-check) / [Proxy](../components/proxy.md#health-check))

### Traffic not received by any targets

1. Is the stream status open in the targets.
2. Find what path the traffic is using with the [Data Plane documentation](../dataplane).
   1. Misconfiguration of the Flows / VIPs
   2. Faulty connectivity

### Traffic received by the wrong targets

1. Is the stream status open in the targets and closed in the wrong targets.
2. Find what path the traffic is using with the [Data Plane documentation](../dataplane).
   1. Misconfiguration of the Flows / VIPs
   2. Faulty connectivity

### Traffic received by only a few targets

1. Is the stream status open in all targets.
2. Find what path the traffic is using with the [Data Plane documentation](../dataplane).
   1. Misconfiguration of the Flows / VIPs
   2. Faulty connectivity

### Some pods are losing connectivity with primary kubernetes network

1. Check if the internal Meridio subnet is conflicting with the Kubernetes ones (Pod, Service, DNS).

### `metadata.labels.trench: Forbidden: update on label trench is forbidden`

1. A custom object cannot be switched between trenches. Check the limitations ([Conduit](../concepts/conduit.md#limitations) / [Stream](../concepts/stream.md#limitations) / [Flow](../concepts/flow.md#limitations) / [VIP](../concepts/vip.md#limitations) / [Attractor](../concepts/attractor.md#limitations) / [Gateway](../concepts/gateway.md#limitations))

### `Failed to call webhook: connect: connection refused`

1. The operator is not fully ready.

### Stream status is `unavailable` in the TAPA

1. The NSP service is unreachable.
   1. The NSP pod is not running (Upgrade, Not ready...).
   2. The namespace is misconfigured.
   3. Too many targets are registered in the stream.
   4. The nsp service name/port is misconfigured.

### Stream status is `undefined` in the TAPA

1. The stream/conduit/trench is not existing.

### Something else

Please, open an issue on [Github](https://github.com/Nordix/Meridio/issues/new?assignees=&labels=kind%2Fbug&template=bug_report.md&title=).
