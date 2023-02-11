# User

Message: `I want to send traffic with 20.0.0.1 as destination IP and 4000 as port, but the traffic doesn't reach the targets (target-a).`

Meridio configuration:
* Everything is deployed in namespace `red`
* 1 Trench `trench-a`, 1 Attractor `attractor-a-1`, 2 Gateways `gateway-v4-a`/`gateway-v6-a`, 2 Vips `vip-a-1-v4`/`vip-a-1-v6`, 1 Conduit `conduit-a-1`, 1 Stream `stream-a-i`, 1 Flow `flow-a-z-tcp`

Gateway configuration:
* Interface
   * VLAN: VLAN ID `100`
   * The VLAN network is based on the network the Kubernetes worker nodes are attached to via `eth0`
   * IP: `169.254.100.150/24` and `100:100::150/64`
* Routing protocol
   * BGP + BFD
   * local-asn: `8103`
   * remote-asn: `4248829953`
   * local-port: `10179`
   * remote-port: `10179`

Target configuration:
* Deployment: `target-a` in namespace `red`
* Stream `stream-a-i` in `conduit-a-1` in `trench-a` is opened in all targets

Run traffic: 
* `docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s`{{exec}}
* `docker exec -it trench-a mconnect -address  [2000::1]:4000 -nconn 400 -timeout 2s`{{exec}}

# Help

```
```

# Solution

Click on "NEXT" to find the solution.