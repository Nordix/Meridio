# Frontend

* [cmd](https://github.com/Nordix/Meridio/tree/master/cmd/frontend)
* [Dockerfile](https://github.com/Nordix/Meridio/tree/master/build/frontend)

## Description

The frontend enables Meridio to attract external traffic through a secondary network.

An external interface must be explicitly provided to the frontend for external connectivity. This interface can be provisioned using Multus, and when Multus is utilized, IP address allocation can be managed by a suitable IPAM CNI plugin (e.g., [whereabouts](https://github.com/k8snetworkplumbingwg/whereabouts)) configured within the Network Attachment Definition.

Upon startup, the frontend performs several crucial actions. It installs source routing rules for each configured VIP address. It configures and launches a [BIRD](https://bird.network.cz/) routing program instance to provide external connectivity. The operations of this BIRD routing suite are specifically restricted to the external interface. For both monitoring and modifying the BIRD configuration, the frontend leverages [birdc](https://bird.network.cz/?get_doc&v=20&f=bird-4.html). Since the frontend relies on `birdc` for monitoring connectivity, any BIRD version update must be executed with special care to maintain compatibility and avoid breaking functionality.

Currently, BGP protocol (with optional BFD supervision) and Static+BFD setups are supported. A key characteristic of these protocols is their lack of inherent neighbor discovery, which means the external gateway IP addresses must be manually configured.

For BGP, the frontend advertises a next-hop route for each VIP address to its external peer, using the frontend's IP as the next-hop. This mechanism directly attracts external traffic to the frontend. Conversely, the external BGP peer is expected to provide at least one next-hop route, which the VIP source routing then utilizes to steer egress traffic. The external BGP peer has the flexibility to announce either a default route or a specific set of network routes.

It's important to note that both ingress and egress traffic will traverse a frontend POD, though not necessarily the same POD for both directions.

Currently, the frontend and load balancer are collocated within the same POD. This collocation is crucial for traffic flow:
- The load balancer relies on the frontend to forward egress traffic.
- Conversely, the frontend depends on the load balancer to handle ingress traffic.

Furthermore, the frontend signals its external connectivity status to its local load balancer. In turn, the frontend receives information from the collocated load balancer about its capability to forward ingress traffic to application targets.

This information is particularly vital in a BGP setup. It allows the system to control precisely when to advertise VIP addresses, thereby preventing the attraction of external traffic if ingress forwarding is not yet fully available. This also implies that VIP addresses are not advertised via BGP unless application targets exist (i.e., TAPA users must open Streams for VIP addresses to be announced externally).

To prevent egress VIP traffic from leaking into the primary network, the frontend installs source routing rules with a lower priority. These rules are designed to match and blackhole such traffic whenever there is no external connectivity.

### External Gateway Router

The external peer that a Meridio frontend connects with must be configured separately, as this falls outside the scope of Meridio itself.

Here are some generic pointers for setting up the external router side, with a focus on BGP configuration:
- The external peer must reside within the same secondary network and subnet as the connected frontend's external interface.
- Depending on your chosen IPAM CNI plugin, you might need to exclude addresses allocated to external peers from assignment to prevent conflicts.
- To avoid configuring every possible IP address that frontends might use to connect to an external BGP router, consider enabling passive BGP peering on the router side.
- By default, Meridio uses BGP Autonomous System (AS) 8103. It expects the gateway router to use AS 4248829953. The default BGP port for both sides is 10179.

### BIRD Router ID

BIRD requires a router ID for its operation. Since the frontend does not currently provide a router ID within the BIRD configuration file it assembles, BIRD determines the ID itself. This determination is based on the interfaces and IPv4 addresses available at its startup.

Typically, this process results in BIRD selecting the primary interface's IPv4 address. However, on an IPv6-only cluster, this automated router ID selection fails, preventing BIRD from successfully starting.

#### Solution for IPv6-Only Clusters

A possible workaround to enable router ID generation on an IPv6-only cluster is to leverage a second Network Attachment Definition (NAD) within the Frontend POD. This NAD can supply an IPv4 address on a dummy interface, allowing BIRD to find a suitable IPv4 address for its Router ID. Alternatively, ensure that the Network Attachment Definition provisioning the external interface also assigns an IPv4 address.

Here's an example configuration:

```yaml
--- a/config/templates/charts/meridio/deployment/stateless-lb-frontend.yaml
+++ b/config/templates/charts/meridio/deployment/stateless-lb-frontend.yaml
@@ -22,6 +22,8 @@ spec:
     type: RollingUpdate
   template:
     metadata:
+      annotations:
+        k8s.v1.cni.cncf.io/networks: '[{"name":"bird-router-id-ipv4","namespace":"default","interface":"dummy"}]'
       labels:
         app: stateless-lb-frontend
         app-type: stateless-lb-frontend
```

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: bird-router-id-ipv4
  namespace: default
spec:
  config: |
    {
      "cniVersion": "1.0.0",
      "name": "bird-rid-net",
      "plugins": [
        {
          "type": "dummy",
          "ipam": {
            "type": "whereabouts",
            "range": "10.255.0.0/20"
          }
        }
      ]
    }
```

The following logs confirm that BIRD successfully uses the dummy interface's IPv4 address as its router ID, demonstrating the workaround's effectiveness:

```
kubectl exec -ti stateless-lb-frontend-attr-a1-6c54757d74-m6vh4 -c frontend -- cat /var/log/bird.log
2025-07-24 15:09:08.815 <INFO> Chosen router ID 10.255.0.2 according to interface dummy
...
2025-07-24 15:09:08.820 <INFO> Reconfigured
2025-07-24 15:09:09.744 <TRACE> NBR-gateway-a2: Incoming connection from 100:100::150 (port 38797) accepted
2025-07-24 15:09:09.744 <TRACE> NBR-gateway-a2: BGP session established
2025-07-24 15:09:09.744 <TRACE> NBR-gateway-a2: State changed to up


kubectl exec -ti stateless-lb-frontend-attr-a1-6c54757d74-8n67j -c frontend -- cat /var/log/bird.log
2025-07-24 15:09:09.895 <INFO> Chosen router ID 10.255.0.1 according to interface dummy
...
2025-07-24 15:09:09.900 <INFO> Reconfigured
2025-07-24 15:09:10.383 <TRACE> NBR-gateway-a2: Incoming connection from 100:100::150 (port 56819) accepted
2025-07-24 15:09:10.383 <TRACE> NBR-gateway-a2: BGP session established
2025-07-24 15:09:10.383 <TRACE> NBR-gateway-a2: State changed to up

```

## Configuration 

https://github.com/Nordix/Meridio/blob/master/cmd/front-end/internal/env/config.go

Environment variable | Type | Description | Default
--- | --- | --- | ---
NFE_VRRPS | []string | VRRP IP addresses to be used as next-hops for static default routes | 
NFE_EXTERNAL_INTERFACE | string | External interface to start BIRD on | ext-vlan
NFE_BIRD_CONFIG_PATH | string | Path to place bird config files | /etc/bird
NFE_LOCAL_AS | string | Local BGP AS number | 8103
NFE_REMOTE_AS | string | Local BGP AS number | 4248829953
NFE_BGP_LOCAL_PORT | string | Local BGP server port | 10179
NFE_BGP_REMOTE_PORT | string | Remote BGP server port | 10179
NFE_BGP_HOLD_TIME | string | Seconds to wait for a Keepalive message from peer before considering the connection stale | 3
NFE_TABLE_ID | int | Start ID of the two consecutive OS Kernel routing tables BIRD syncs the routes with | 4096
NFE_ECMP | bool | Enable ECMP towards next-hops of avaialble gateways | false
NFE_DROP_IF_NO_PEER | bool | Install default blackhole route with high metric into routing table TableID | true
NFE_LOG_BIRD | bool | Add important bird log snippets to our log | false
NFE_NAMESPACE | string | Namespace the pod is running on | default
NFE_NSP_SERVICE | string | IP (or domain) and port of the NSP Service | nsp-service-trench-a:7778
NFE_TRENCH_NAME | string | Name of the Trench the frontend is associated with | default
NFE_ATTRACTOR_NAME | string | Name of the Attractor the frontend is associated with | default
NFE_LOG_LEVEL | string | Log level | DEBUG
NFE_NSP_ENTRY_TIMEOUT | time.Duration | Timeout of entries registered in NSP | 30s
NFE_GRPC_KEEPALIVE_TIME | time.Duration | gRPC keepalive timeout | 30s
NFE_GRPC_MAX_BACKOFF | time.Duration |  Upper bound on gRPC connection backoff delay | 5s
NFE_DELAY_CONNECTIVITY | time.Duration | Delay between routing suite checks with connectivity | 1s
NFE_DELAY_NO_CONNECTIVITY | time.Duration | Delay between routing suite checks without connectivity | 3s
NFE_MAX_SESSION_ERRORS | int | Max session errors when checking routing suite until denounce | 5
NFE_METRICS_ENABLED | bool | Enable the metrics collection | false
NFE_METRICS_PORT | int | Specify the port used to expose the metrics | 2224
NFE_LB_SOCKET | url.URL | LB socket to connect to | unix:///var/lib/meridio/lb.sock

## Command Line 

Command | Action | Default
--- | --- | ---
--help | Display a help describing |
--version | Display the version |
--debug | Prints meridio-version, unix-time, network-interfaces, rules, route, neighbors, system information, and environment-variables in a json format |

## Communication 

Here are all components the frontend is communicating with:

Component | Secured | Method | Description
--- | --- | --- | ---
Spire | TBD | Unix Socket | Obtain and validate SVIDs
NSP Service | yes (mTLS) | TCP | Watch configuration. Register/Unregister target (Advertise its readiness to the NSP target registry)
Gateways | / | / | Routing protocol
Kubernetes API | TDB | TCP | Watch the secrets for BGP authentication
LB | yes (mTLS) | Unix socket | Watch internal connectivity status of collocated stateless-lb

An overview of the communications between all components is available [here](resources.md).

## Health check

The health check is provided by the [GRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md). The status returned can be `UNKNOWN`, `SERVING`, `NOT_SERVING` or `SERVICE_UNKNOWN`.

Service | Description
--- | ---
Readiness | A unique service to be used by readiness probe to return status, can aggregate other lesser services

Service | Probe | Description
--- | --- | ---
NSPCli | Readiness | Monitor status of the connection to the NSP service
Egress | Readiness | Monitor the gateways connectivity

## Privileges

To work properly, here are the privileges required by the frontend:

Name | Description
--- | ---
Sysctl: net.ipv4.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv6.conf.all.forwarding=1 | Enable IP forwarding
Sysctl: net.ipv4.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv4
Sysctl: net.ipv6.fib_multipath_hash_policy=1 | To use Layer 4 hash policy for ECMP on IPv6
Sysctl: net.ipv4.conf.all.rp_filter=0 | Allow packets to have a source IPv4 address which does not correspond to any routing destination address.
Sysctl: net.ipv4.conf.default.rp_filter=0 | Allow packets to have a source IPv6 address which does not correspond to any routing destination address.
Sysctl: net.ipv4.ip_local_port_range='49152 65535' | The source port of BFD Control packets must be in the IANA approved range 49152-65535
NET_ADMIN | The frontend creates IP rules to handle outbound traffic from VIP sources. BIRD interacts with kernel routing tables.
NET_BIND_SERVICE | Allows BIRD to bind to privileged ports depending on the config (for example to BGP port 173).
NET_RAW | Allows BIRD to use the SO_BINDTODEVICE socket option.
Kubernetes API | fes-role - secrets - watch
