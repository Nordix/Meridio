# Use Cases

## Secondary Networking

### Resiliency, Redundancy and Fault-Tolerance

Secondary networking can provide redundancy, automatic failover and fault-tolerance. If one network interface fails, the other can take over, ensuring that the system remains connected to the network. The global system will become more resilient, secondary networking will help isolate the effects of a network failure, preventing it from affecting other parts of the system and minimizing downtime.

### Security

Traffic isolation is also tightly coupled with the security requirements of the telecommunication companies, internet service providers (ISP), and advanced enterprise networks in order to secure the users/consumers data flow[^1]. Security functions, such as firewalls or intrusion detection systems can be used onto a separate network interface to filter and block malicious traffic using, for instance, advanced tools like a Next-Generation Firewall (NGFW). Additionally, access controls and encryption may be used to ensure that data is protected from unauthorized access or theft.

### Performance

From the usage of secondary networking where multiple network interfaces are used to attract and distribute network traffic, infrastructures and applications also benefit with a performance gain thanks to the reduction of the congestion and the minimization of the local traffic. Furthermore, each network can have its own specificities and capabilities provided via different dataplane methods, some critical network traffic may be prioritized over less critical traffic, some others may require high performance with traffic acceleration on specific dedicated network/hardware.

### Simplified Network Management

Telco networks can be complex and require extensive management, making it challenging to keep them running efficiently. However, secondary networking can help simplify network configuration and management by facilitating multi-tenancy. This simplification reduces the risk of human error and ensures that network issues are identified and resolved quickly.

### Specialization

One of the legacy requirements of the telecom infrastructure is to support network separation with a dedicated network interface according to the type of the traffic and its policies. By segregating traffic based on its type, for instance, voice, data, or video, the quality of service (QoS) for each type of traffic can be guaranteed. Additionally, separating based on the task of traffic, such as packet capturing, network monitoring, or testing can help for troubleshooting, network analysis, or compliance efforts. Each network can have its own configuration, specific protocol, network function and type of connectivity.

## External Traffic Attraction

Announcing service-addresses (VIP-addresses) to the external gateways permits the traffic to be attracted towards the different services exposed by the target applications. Frontend services provide different connectivity mechanisms such as VLANs or host network interfaces to connect the network services to the gateways. In addition, Multus can offer decoupling of connectivity mechanisms while supplying a whole variety of network interfaces through CNI plugins. To announce the service and ensure the link-supervision, routing protocols such as BGP and Static BFD are used.

## Network Services

### Stateless load balancing

Stateless load balancers offer several advantages over stateful load balancers. One key advantage is horizontal scalability without relying on shared state information, which in turn reduces latency. Additionally, stateless load balancers simplify management, deployment, monitoring, and troubleshooting of the infrastructure. They are particularly well-suited to modern application architectures that rely on stateless services and require high scalability and availability.

### No NAT

Certain applications, particularly in telecommunication are sensitive to Network Address Translation (NAT). For instance, Session Initiation Protocol (SIP), a protocol used for establishing and terminating voice and video calls over IP networks, is sensitive to NAT. The IP addresses and ports in the SIP messages would need to be translated in order to reach the correct destination. This can cause issues with call quality and reliability.

However, some applications accepting NATting might require it in order to expose their service with certain ports. By default, privileged ports under 1024 require root or CAP_NET_BIND_SERVICE in order to bind to them[^2]. Therefore, applications must NAT service port to an unprivileged port open on their host.

### Policies

By providing traffic classification to steer traffic into multiple different instances of the network service applications can subscribe, users are able to customize their service as desired. The classification and steering can be done bases on the 5-tuple (the source and destination IP, source and destination port and protocol), as well as on some specific bytes of the layer 4 header. Additionally, priority classification offers even greater flexibility in configuration.

## Runtime Configuration

Finally, current cloud-native solutions allow for dynamic configuration of what to deploy, but the realization of that deployment is mostly immutable[^3]. Network allocation is taking place only during the workload initialization and deletion phases. To meet the expectations of adopters, it is crucial to leverage cloud-native paradigms effectively with the possibility to handle on demand network service adjustment with minimization of the traffic disturbance.

[^1]: https://uu.diva-portal.org/smash/get/diva2:1500742/FULLTEXT01.pdf
[^2]: https://sysctl-explorer.net/net/ipv4/ip_unprivileged_port_start/
[^3]: https://github.com/networkservicemesh/networkservicemesh/blob/release-0.2/docs/what-is-nsm.md