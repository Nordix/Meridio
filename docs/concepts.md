# Concepts

## Trench

A Trench defines the extension of an external external network into the Kubernetes cluster. Inside each trench the traffic can be split to take different logical and physical paths through the cluster.

### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
Name | string | Name of the Trench | yes | 

## Conduit

The Conduit defines a physical path of chained network services the traffic will pass before it can be consumed by the target application. The Traffic is assigned to the Conduit by Stream. Currently, the only supported network service function is stateless load balancing.

The picture below describes 2 conduits. the left one is a stateless load-balancer and the right one is a SCTP network service. The left user pod has subscribed to one or more stream the stateless load-balancer is handling, so it is connected to it via a new network interface. The right user pod has subscribed to streams the stateless load-balancer and the SCTP network services are handling, so it is connected to them via two new network interfaces, one connected to the stateless load-balancer network service, and the other one connected to the SCTP network service. 

<img src="resources/Overview-Conduit.svg" width="50%">
<!-- ![Overview](resources/Overview-Conduit.svg) -->

### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Conduit | yes |
trench | string | Name of the Trench the Conduit belongs to | yes | 

## Stream/Flow

<img src="resources/Overview-Stream-Flow.svg" width="75%">
<!-- ![Overview](resources/Overview-Stream-Flow.svg) -->

### Stream

The Stream reflects a logical grouping of traffic flows. The stream points out the conduit the traffic will pass before it can be consumed by the target application.

Stream is a logical configuration entity, which cannot directly be found in the payload traffic.

The stream is the "network service entity" to be known and referred in the target application. The different target application pods can sign up for consumption of traffic from different streams. When more target pods are signed up for the same stream the traffic will be load-balanced between the pods.

Notice that a target pod concurrently only can sign up for streams belonging to the same trench.

#### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Stream | yes |
conduit | string | Name of the Conduit the Stream belongs to | yes | 

### Flow

The Flow classifies incoming traffic based on the 5-tuple in the IP-header containing: Source and destination IP-addresses, source and destination ports and protocol type.

The traffic that match the classification are mapped to a logically stream.

It is allowed to configure flows having overlapping IP-addresses and port ranges. In this case the priority setting is used to decide which flow will be tried matched first. 

Traffic that do not match the classification for any flow will be dropped.

Notice that destination IP-address is provided by referencing one or more Vip.

#### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
Name | string | Name of the Flow | yes |
source-subnets | []string |  | yes | 
destination-port-ranges | []string |  | yes | 
source-port-ranges | []string |  | yes | 
protocols | []string |  | yes | 
vips | []string |  | yes | 
priority | int |  | yes | 
stream | string | Name of the Stream the Flow belongs to | yes | 

## VIP

The Vip defines a virtual IP address towards which external traffic can be attracted and load-balanced over a scaling set of target application pods. 

As soon as a Vip has been applied to a working trench and attractor the VIP address will be announced by all FEs using BGP. 

### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the VIP | yes | 
address | string | The virtual IPaddress. Both ipv4 and ipv6 addresses are supported. The VIP address must be a valid network prefix. | yes | 
trench | string | Name of the Trench the VIP belongs to | yes | 


## Attractor

The Attractor defines a group of Front-Ends (FEs) sharing commonalities, for example; external interface to utilize, IP-address assignment method, gateway(s) to establish connection with, VIP-addresses to announce etc. 

### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Attractor | yes | 
vips | []string |  | yes | 
gateways | []string |  | yes | 
trench | string | Name of the Trench the Attractor belongs to | yes | 

## Gateway

The Gateway defines how to establish connectivity with an external gateway such as; which IP-address, routing- and supervision-protocol(s) to use. The Gateway can also define specific protocol settings that differ from the default etc.

Notice that it normally is good practice and often required have "mirrored" settings shared between the external gateway and the Meridio FrontEnds to get the BGP and BFD sessions established. The used "retry" and "time-out" settings will dictate the time it takes traffic to fail-over in case of a link failure.

Notice that when static is chosen as routing protocol BFD link-supervision is by default turned on with default settings.

Note: In the Alpha release BGP with BFD is not a supported option.

### Configuration

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Gateway | yes | 
address | string |  | yes | 
remote-asn | int |  | yes | 
local-asn | int |  | yes | 
remote-port | int |  | yes | 
local-port | int |  | yes | 
ip-family | string |  | yes | 
bfd | bool |  | yes | 
protocol | string |  | yes | 
hold-time | int |  | yes | 
min-tx | int |  | yes | 
min-rx | int |  | yes | 
multiplier | int |  | yes | 
trench | string | Name of the Trench the Gateway belongs to | yes | 