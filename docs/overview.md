# Overview

## How it works

### Concepts

Meridio introduces the concept of a "trench". A trench can be seen as an extension of an actual external network. It is the trench that upholds the traffic separation for each external network inside the realm of the cluster.

Inside each trench the traffic can be split to take different logical and physical paths through the cluster. Such a path is termed as a "conduit". In each conduit the traffic can be subjected to a chain of different network services.

A network service is instantiated as set of scaling elements in a function block. By interlinking the elements in one function block with elements in the next function block a network service chain is build. One common example would be a Front-End function block chained with a Load-Balancer function block.

Some network service chains do inherently open for collocation of more function blocks to minimize the interlinking overhead.

The "cluster breakout" and connectivity with the gateway providing access to an external network is handled by the Front-End function block. The Front-End function block is common for all conduits in the trench. To align with possible existing sub-divisions of a legacy VPN network the Front-End function block can be split into more "attractors". The Front-End instances in each attractor can be connected with a separate gateway which provides access to a distinct part of the external network.

When more conduits are instantiated within the trench, the Front-End function block must classify the incoming traffic for selecting the conduit to forward the traffic via.

Traffic classification is done using pre-configured 5-tuple based flow-policies which group the traffic into "streams". A stream is a logical abstraction for a collection of traffic flows, which are forwarded through one conduit.

After passing the chain of network services inside the conduit the stream is to be consumed by the target application. In each target application instance the
interaction with Meridio is handled by one "Target Access Point" (TAP) for each conduit the application wants to connect to. By utilizing the TAP the target application instance can sign-up to one or more "Stream Consumer Pools" (SCP) inside each conduit in order to receive traffic from the different streams.

In the target application instance the traffic will finally reach the destination VIP address for being consumed by the application.

Egress response will via the TAP be fed into a conduit.
Depending on the actual network service chain the traffic could be subjected to some traffic handling, but in most cases the traffic will just be forwarded to the gateway via the Front-End function block.

The picture below provides an architectural overview of Meridio.
<img src="resources/Overview-Concepts.svg" width="75%">

For more details, please read the documentation of each concept:
* [Trench](concepts.md#trench)
* [Conduit](concepts.md#conduit)
* [Stream](concepts.md#stream)
* [Flow](concepts.md#flow)
* [VIP](concepts.md#vip)
* [Attractor](concepts.md#attractor)
* [Gateway](concepts.md#gateway)

<!-- 
### Runtime configuration

https://github.com/Nordix/Meridio-Operator

### Network Service Mesh

https://networkservicemesh.io/
https://github.com/networkservicemesh/networkservicemesh/tree/v0.2.0
https://github.com/networkservicemesh/networkservicemesh/blob/v0.2.0/docs/what-is-nsm.md 
-->

### Communication

The picture below provides an overview of the communication within Meridio.

<img src="resources/Overview-Communication.svg" width="100%">

For more details, please read the documentation of each component:
* [table of contents](readme.md)
    * [Load-Balancer](load-balancer.md)
    * [Front-End](front-end.md)
    * [Proxy](proxy.md)
    * [Target Access Point Ambassador (TAPA)](tapa.md)
    * [Network Service Platform (NSP)](nsp.md)
    * [IPAM](ipam.md)
    * [User Application](user-application.md)
    * NSC
    * NSE-VLAN
