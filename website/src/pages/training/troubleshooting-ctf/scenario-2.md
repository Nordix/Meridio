# Scenario 2 - Stateless-lb-frontend pods are not ready

### Problem Statement

The user tries to send traffic to the application pod, target-a, which has IP address `20.0.0.1` and port `4000`. However, traffic doesn't reach the intended destination. Troubleshooting is requested.

### Known Inputs

#### Cluster resource

* Spire is deployed in the namespace `spire`
* NSM is deployed in the namespace `nsm`

#### Meridio configuration

* Meridio version: `v1.0.0`
* TAPA version: `v1.0.0`
* Meridio is deployed in the namespace `red`
* Meridio components:
  * 1 Trench `trench-a`
  * 1 Attractor `attractor-a-1`
  * 2 Gateways `gateway-v4-a`/`gateway-v6-a`
  * 2 Vips `vip-a-1-v4`/`vip-a-1-v6`
  * 1 Conduit `conduit-a-1`
  * 1 Stream `stream-a-i`
  * 1 Flow `flow-a-z-tcp`

#### Gateway configuration

* Interface
  * VLAN: VLAN ID `100`
  * The VLAN network is based on the network the Kubernetes worker nodes are attached to via `eth0`
  * IPv4 `169.254.100.150/24` and IPv6 `100:100::150::/64`
* Routing protocol
  * BGP + BFD
  * local-asn: `8103`
  * remote-asn: `4248829953`
  * local-port: `10179`
  * remote-port: `10179`

#### Target configuration

* Deployment: `target-a` in the namespace `red`
* Stream `stream-a-i` in `conduit-a-1` in `trench-a` is open in all the target pods

### Error

* Running traffic gives the following output

```bash
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
Failed connects; 400
Failed reads; 0
```

```bash
docker exec -it trench-a mconnect -address  [2000::1]:4000 -nconn 400 -timeout 2s
Failed connects; 400
Failed reads; 0
```

### Solution

1. Check the status of all the Spire and NSM pods running in the namespaces `spire` and `nsm`, correspondingly.
   * All pods should be in the *running state*, which means that all the pods have been bound to a node, and all of the containers have been created. All the containers inside the pods are in the *ready state*.

```bash
kubectl get pods -n=spire
NAME                READY   STATUS    RESTARTS   AGE
spire-agent-4wj68   1/1     Running   0          2m1s
spire-agent-q77lz   1/1     Running   0          2m4s
spire-server-0      2/2     Running   0          2m6s
```

```bash
kubectl get pods -n=nsm
NAME                                    READY   STATUS    RESTARTS   AGE
admission-webhook-k8s-b9589cbcb-g9wxs   1/1     Running   0          7m33s
forwarder-vpp-d4sqt                     1/1     Running   0          7m33s
forwarder-vpp-n5l2p                     1/1     Running   0          7m33s
nsm-registry-5b5b897645-6bcs2           1/1     Running   0          7m33s
nsmgr-6msjz                             2/2     Running   0          7m33s
nsmgr-8kh7t                             2/2     Running   0          7m33s
```

2. Check the status of all the Meridio pods running in the namespace `red`.
   * All pods should be in the *running state*, which means that all the pods have been bound to a node, and all of the containers have been created. All the containers inside the pods are in the *ready state*.
   * In the case of the current scenario, all the pods in the namespace `red` are in the *running state* ; however, it can be noticed that some containers inside the few pods are not in the *ready* state. Particularly, 2 containers in `stateless-lb-frontend-attractor-a-1` pods and 1 container in `proxy-conduit-a-1` pods.

```bash
kubectl get pods -n=red
NAME                                                  READY   STATUS    RESTARTS   AGE
ipam-trench-a-0                                       1/1     Running   0          34m
meridio-operator-596d7f88b8-v5gf9                     1/1     Running   0          34m
nse-vlan-attractor-a-1-5cf67947d5-f8slj               1/1     Running   0          34m
nsp-trench-a-0                                        1/1     Running   0          34m
proxy-conduit-a-1-6vdxl                               0/1     Running   0          34m
proxy-conduit-a-1-hv4m5                               0/1     Running   0          34m
stateless-lb-frontend-attractor-a-1-d8db96c8f-6kv6l   1/3     Running   0          34m
stateless-lb-frontend-attractor-a-1-d8db96c8f-shvfd   1/3     Running   0          34m
target-a-77b5b48457-6xkcj                             2/2     Running   0          33m
target-a-77b5b48457-jxbdv                             2/2     Running   0          33m
target-a-77b5b48457-pl9fz                             2/2     Running   0          33m
target-a-77b5b48457-szzzw                             2/2     Running   0          33m
```

3. As the `proxy-conduit-a-1` start-up depends on the successful start-up of the `stateless-lb-frontend-attractor-a-1`, the first step would be to check the details of the `stateless-lb-frontend-attractor-a-1`. This can give some hints on the possible reasons for the failed containers' start.
   * Going through the `stateless-lb-frontend-attractor-a-1` detailed description presented below, a few things can be observed:
     * Which containers are not in the ***Ready*** state  `(Ready: False)` : `frontend` and `stateless-lb`.
     * The history of ***Events***: the following warning is received `Readiness probe failed: service unhealthy (responded with "NOT_SERVING")`.

```yaml
kubectl describe pods stateless-lb-frontend-attractor-a-1-d8db96c8f-q92dc -n=red

Containers:
  stateless-lb:
    Container ID:   ...
    Image:          ...
    Image ID:       ...
    Port:           ...
    Host Port:      ...
    State:          Running
      Started:      Tue, 21 Feb 2023 21:49:20 +0000
    Ready:          False
    Restart Count:  0
    Liveness:       exec [/bin/grpc_health_probe -addr=unix:///tmp/health.sock -service= -connect-timeout=250ms -rpc-timeout=350ms] delay=0s timeout=3s period=10s #success=1 #failure=5
    Readiness:      exec [/bin/grpc_health_probe -addr=unix:///tmp/health.sock -service=Readiness -connect-timeout=250ms -rpc-timeout=350ms] delay=0s timeout=3s period=10s #success=1 #failure=5
    Startup:        exec [/bin/grpc_health_probe -addr=unix:///tmp/health.sock -service= -connect-timeout=250ms -rpc-timeout=350ms] delay=0s timeout=2s period=2s #success=1 #failure=30

---
  nsc:
    Container ID:   ...
    Image:          ...
    Image ID:       ...
    Port:           ...
    Host Port:      ...
    State:          Running
      Started:      Tue, 21 Feb 2023 21:49:43 +0000
    Ready:          True
    Restart Count:  0

---
  frontend:
    Container ID:   ...
    Image:          ...
    Image ID:       ...
    Port:           ...
    Host Port:      ...
    State:          Running
      Started:      Tue, 21 Feb 2023 21:49:52 +0000
    Ready:          False
    Restart Count:  0
    Liveness:       exec [/bin/grpc_health_probe -addr=unix:///tmp/health.sock -service= -connect-timeout=250ms -rpc-timeout=350ms] delay=0s timeout=3s period=10s #success=1 #failure=5
    Readiness:      exec [/bin/grpc_health_probe -addr=unix:///tmp/health.sock -service=Readiness -connect-timeout=250ms -rpc-timeout=350ms] delay=0s timeout=3s period=10s #success=1 #failure=5
    Startup:        exec [/bin/grpc_health_probe -addr=unix:///tmp/health.sock -service= -connect-timeout=250ms -rpc-timeout=350ms] delay=0s timeout=2s period=2s #success=1 #failure=30
```

```bash
  Warning  Unhealthy  62s (x4 over 64s)  kubelet            Readiness probe failed: service unhealthy (responded with "NOT_SERVING")
  Warning  Unhealthy  53s (x5 over 64s)  kubelet            Readiness probe failed: service unhealthy (responded with "NOT_SERVING")
```

4. From the previous step, it can be concluded that the reason for the failed *readiness* of the two containers, `frontend` and `stateless-lb`, is the failed ***readiness probe***. To debug further, it would be useful to check the logs of the two containers for any sort of errors.
   * Going through the logs of the `stateless-lb` container, it can be noticed that there are no ***error*** messages.
   * Going through the logs of the `frontend` container, it can be noticed that there is one ***error*** message, which states `"error":"gateway down"`.


```bash
kubectl logs stateless-lb-frontend-attractor-a-1-d8db96c8f-q92dc -n=red -c=stateless-lb | grep "\"severity\":\"error\""

...

kubectl logs stateless-lb-frontend-attractor-a-1-d8db96c8f-q92dc -n=red -c=frontend | grep "\"severity\":\"error\""
{"severity":"error","timestamp":"2023-02-21T21:49:56.733+00:00","service_id":"Meridio-frontend","message":"connectivity","version":"1.0.0","extra_data":{"class":"FrontEndService","func":"Monitor","status":16,"out":["BIRD 2.0.8 ready.","Name       Proto      Table      State  Since         Info","NBR-gateway-v4-a BGP        ---        start  21:49:53.760  Active            Neighbor address: 169.254.100.150%ext-vlan0","NBR-gateway-v6-a BGP        ---        start  21:49:53.760  Active            Neighbor address: 100:100::150%ext-vlan0",""],"error":"gateway down"}}
```

5. Furthermore, it would be useful to check the state of the BGP connection and Bird logs.
    * In a normal case, the  BGP state should be **ESTABLISHED**, and IPv4 and IPv6 channels' states should be **UP**. However, as can be seen, the BGP state is **ACTIVE** (refer to 'Helpful Resources: BGP' to get more info about the states), and IPv4 and IPv6 channels are **DOWN**. The error received is `connection refused`.

```bash
kubectl exec -it stateless-lb-frontend-attractor-a-1-d8db96c8f-q92dc -n=red -c=frontend -- birdc -s /var/run/bird/bird.ctl show protocol all
BIRD 2.0.8 ready.
Name       Proto      Table      State  Since         Info
device1    Device     ---        up     10:06:22.017

kernel1    Kernel     master4    up     10:06:22.017
  Channel ipv4
    State:          UP
    Table:          master4
    Preference:     10
    Input filter:   REJECT
    Output filter:  default_rt
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

kernel2    Kernel     master6    up     10:06:22.017
  Channel ipv6
    State:          UP
    Table:          master6
    Preference:     10
    Input filter:   REJECT
    Output filter:  default_rt
    Routes:         0 imported, 0 exported, 0 preferred
    Route change stats:     received   rejected   filtered    ignored   accepted
      Import updates:              0          0          0          0          0
      Import withdraws:            0          0        ---          0          0
      Export updates:              0          0          0        ---          0
      Export withdraws:            0        ---        ---        ---          0

NBR-gateway-v4-a BGP        ---        start  10:06:22.326  Active        Socket: Connection refused
  BGP state:          Active
    Neighbor address: 169.254.100.150%ext-vlan0
    Neighbor AS:      4248829953
    Local AS:         8103
    Connect delay:    2.089/5
    Last error:       Socket: Connection refused
  Channel ipv4
    State:          DOWN
    Table:          master4
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static
  Channel ipv6
    State:          DOWN
    Table:          master6
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT

NBR-gateway-v6-a BGP        ---        start  10:06:22.326  Active        Socket: Connection refused
  BGP state:          Active
    Neighbor address: 100:100::150%ext-vlan0
    Neighbor AS:      4248829953
    Local AS:         8103
    Connect delay:    0.463/5
    Last error:       Socket: Connection refused
  Channel ipv4
    State:          DOWN
    Table:          master4
    Preference:     100
    Input filter:   REJECT
    Output filter:  REJECT
  Channel ipv6
    State:          DOWN
    Table:          master6
    Preference:     100
    Input filter:   default_rt
    Output filter:  cluster_e_static

NBR-BFD    BFD        ---        up     10:06:22.017
```

```bash
kubectl exec -it stateless-lb-frontend-attractor-a-1-d8db96c8f-q92dc -n=red -c=frontend -- cat /var/log/bird.log

2023-02-22 10:24:45.650 <TRACE> NBR-gateway-v4-a: Connecting to 169.254.100.150 from local address 169.254.100.2
2023-02-22 10:24:45.652 <TRACE> NBR-gateway-v4-a: Connection lost (Connection refused)
2023-02-22 10:24:45.652 <TRACE> NBR-gateway-v4-a: Connect delayed by 5 seconds
2023-02-22 10:24:46.141 <TRACE> NBR-gateway-v6-a: Connecting to 100:100::150 from local address 100:100::2
2023-02-22 10:24:46.141 <TRACE> NBR-gateway-v6-a: Connection lost (Connection refused)
2023-02-22 10:24:46.141 <TRACE> NBR-gateway-v6-a: Connect delayed by 5 second
```

6. One of the reasons for the **ACTIVE** BGP state is the BGP configuration error. Therefore, a suggestion would be to check the BGP configuration for any sort of error.
    * The BGP configuration contains information about the gateways. Commonly, the error occurs in the `gateway` custom resource configuration.

```bash
kubectl exec -it stateless-lb-frontend-attractor-a-1-d8db96c8f-q92dc -n=red -c=frontend -- cat /etc/bird/bird-fe-meridio.conf

log "/var/log/bird.log" 20000 "/var/log/bird.log.backup" { debug, trace, info, remote, warning, error, auth, fatal, bug };
log stderr all;

protocol device {
}

filter default_rt {
        if ( net ~ [ 0.0.0.0/0 ] ) then accept;
        if ( net ~ [ 0::/0 ] ) then accept;
        else reject;
}

filter cluster_e_static {
        if ( net ~ [ 0.0.0.0/0 ] ) then reject;
        if ( net ~ [ 0::/0 ] ) then reject;
        if source = RTS_STATIC && dest != RTD_BLACKHOLE then accept;
        else reject;
}

template bgp LINK {
        debug {events, states, interfaces};
        direct;
        hold time 3;
        bfd off;
        graceful restart off;
        setkey off;
        ipv4 {
                import none;
                export none;
                next hop self;
        };
        ipv6 {
                import none;
                export none;
                next hop self;
        };
}

protocol kernel {
        ipv4 {
                import none;
                export filter default_rt;
        };
        kernel table 4096;
        merge paths on;
}

protocol kernel {
        ipv6 {
                import none;
                export filter default_rt;
        };
        kernel table 4096;
        merge paths on;
}

protocol bgp 'NBR-gateway-v4-a' from LINK {
        interface "ext-vlan0";
        local port 10180 as 8103;
        neighbor 169.254.100.150 port 10180 as 4248829953;
        bfd {
                min rx interval 300ms;
                min tx interval 300ms;
                multiplier 5;
        };
        hold time 24;
        ipv4 {
                import filter default_rt;
                export filter cluster_e_static;
        };
}

protocol bgp 'NBR-gateway-v6-a' from LINK {
        interface "ext-vlan0";
        local port 10180 as 8103;
        neighbor 100:100::150 port 10180 as 4248829953;
        bfd {
                min rx interval 300ms;
                min tx interval 300ms;
                multiplier 5;
        };
        hold time 24;
        ipv6 {
                import filter default_rt;
                export filter cluster_e_static;
        };
}

protocol bfd 'NBR-BFD' {
        interface "ext-vlan0" {
        };
}
```

7. As mentioned in the previous step, there might be an issue with the `gateway` custom resource configuration. Therefore, it would be helpful to check the details of the `gateway-v4-a` and `gateway-v6-a` for any sort of misconfiguration.
    * There are several properties of the `gateway` custom resource, which probably should be checked thoroughly since they are a common source of errors if configured incorrectly. Particularly, check if **namespace**, **trench**, **address**, **bgp.local-asn**, **bgp.remote-asn**, **bgp.local-port**, **bgp.remote-port** are set to the correct values following the specified configuration (*reference: known inputs*).
    * Going through the  `gateway-v4-a` and `gateway-v6-a` detailed description presented below and verifying the mentioned properties, it can be noticed that there is a misconfiguration in the **bgp.local-port** and **bgp.remote-port**   fields. These properties should be set to `10179`, however, in the current deployment they are set to `10180`. This misconfiguration leads to the ingress traffic not reaching the intended destination.

```yaml
kubectl get gateway gateway-v4-a -n=red -o=yaml

apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"meridio.nordix.org/v1","kind":"Gateway","metadata":{"annotations":{},"labels":{"trench":"trench-a"},"name":"gateway-v4-a","namespace":"red"},"spec":{"address":"169.254.100.150","bgp":{"bfd":{"min-rx":"300ms","min-tx":"300ms","multiplier":5,"switch":true},"hold-time":"24s","local-asn":8103,"local-port":10180,"remote-asn":4248829953,"remote-port":10180}}}
  creationTimestamp: "2023-02-21T21:49:04Z"
  generation: 2
  labels:
    trench: trench-a
  name: gateway-v4-a
  namespace: red
  ownerReferences:
  - apiVersion: meridio.nordix.org/v1
    kind: Trench
    name: trench-a
    uid: 713fd582-0f86-4c64-94e7-bcdcaa581a92
  resourceVersion: "1185"
  uid: 37cf0655-62f4-44eb-83dc-ec420f452f04
spec:
  address: 169.254.100.150
  bgp:
    bfd:
      min-rx: 300ms
      min-tx: 300ms
      multiplier: 5
      switch: true
    hold-time: 24s
    local-asn: 8103
    local-port: 10180
    remote-asn: 4248829953
    remote-port: 10180
  protocol: bgp
  static:
    bfd: {}
```

```yaml
kubectl get gateway gateway-v6-a -n=red -o=yaml

apiVersion: meridio.nordix.org/v1
kind: Gateway
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"meridio.nordix.org/v1","kind":"Gateway","metadata":{"annotations":{},"labels":{"trench":"trench-a"},"name":"gateway-v6-a","namespace":"red"},"spec":{"address":"100:100::150","bgp":{"bfd":{"min-rx":"300ms","min-tx":"300ms","multiplier":5,"switch":true},"hold-time":"24s","local-asn":8103,"local-port":10180,"remote-asn":4248829953,"remote-port":10180}}}
  creationTimestamp: "2023-02-21T21:49:04Z"
  generation: 2
  labels:
    trench: trench-a
  name: gateway-v6-a
  namespace: red
  ownerReferences:
  - apiVersion: meridio.nordix.org/v1
    kind: Trench
    name: trench-a
    uid: 713fd582-0f86-4c64-94e7-bcdcaa581a92
  resourceVersion: "1219"
  uid: b259507f-b40c-4d5e-848b-bb1b2b020fdc
spec:
  address: 100:100::150
  bgp:
    bfd:
      min-rx: 300ms
      min-tx: 300ms
      multiplier: 5
      switch: true
    hold-time: 24s
    local-asn: 8103
    local-port: 10180
    remote-asn: 4248829953
    remote-port: 10180
  protocol: bgp
  static:
    bfd: {}
```

8. Changing **bgp.local-port** and **bgp.remote-port** for both `gateway-v4-a` and `gateway-v6-a` to `10179` fixes the traffic issue which was reported by the user.

```bash
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
Failed connects; 0
Failed reads; 0
target-a-77b5b48457-n46p4 91
target-a-77b5b48457-qsrwg 89
target-a-77b5b48457-pdtq5 107
target-a-77b5b48457-72vlp 113

docker exec -it trench-a mconnect -address  [2000::1]:4000 -nconn 400 -timeout 2s
Failed connects; 0
Failed reads; 0
target-a-77b5b48457-pdtq5 90
target-a-77b5b48457-72vlp 109
target-a-77b5b48457-qsrwg 110
target-a-77b5b48457-n46p4 91
```

### Helpful Resources

* [Meridio Gateway](https://meridio.nordix.org/docs/v1.0.0/concepts#gateway)
* [Meridio Troubleshooting Guide](https://meridio.nordix.org/docs/v1.0.0/trouble-shooting/)
* [Kubectl Reference Docs](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands)
* [Bird2](https://bird.network.cz/?get_doc&f=bird.html&v=20)
* [BGP Wikipedia](https://en.wikipedia.org/wiki/Border_Gateway_Protocol)
* [mconnect](https://github.com/Nordix/mconnect)