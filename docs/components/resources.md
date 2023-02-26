# Components

* Deployment: `operator`
   * [operator](operator.md)
* Statefulset: `ipam-<trench-name>`
   * [ipam](ipam.md)
* Statefulset: `nsp-<trench-name>`
   * [nsp](nsp.md)
* Deployment: `stateless-lb-frontend-<attractor-name>`
   * sysctl-init
   * [stateless-lb](frontend.md)
   * [frontend](frontend.md)
   * nsc
* Deployment: `nse-vlan-<attractor-name>`
   * nse-vlan
* Daemonset: `proxy-<conduit-name>`
   * sysctl-init
   * [proxy](proxy.md)

## Communication

The picture below provides an overview of the communication within Meridio.

![Overview-Communication](../resources/Overview-Communication.svg)

## From an NSM Perspective

This is how Meridio looks like from an NSM perspective:
![NSM-Perspective](../resources/NSM-Perspective.svg)
* NS: Network Service
* NSE: Network Service Endpoint
* NSC: Network Service Client
* Type: Type of data plane that carries the traffic
* Mechanism: Type of interface injected in the pod
* Payload: Ethernet: The frame sent from an interface to another interface will be intact.

Number of NSEs/NSCs:
* Calculate the number of NSE: `Number of Worker * Number of Conduits + Number of attractor + Number of attractor * Number of replicas (attractor spec)`
* Calculate the number of NSC (not counting the targets): `Number of Worker * Number of Conduits * Number of attractor * Number of replicas (attractor spec) + Number of attractor * Number of replicas (attractor spec)`