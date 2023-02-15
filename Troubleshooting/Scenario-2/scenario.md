# User

Message: `I want to send traffic with 20.0.0.1 as destination IP and 4000 as port, but the traffic doesn't reach the targets (target-a).`

Cluster resource:
* Spire is deployed in namespace `spire`
* NSM is deployed in namespace `nsm`

Meridio configuration:
* Meridio version: `v1.0.0`
* TAPA version: `v1.0.0`
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

[Documentation](https://meridio.nordix.org/docs/v1.0.0/)
[Troubleshooting Guide](https://meridio.nordix.org/docs/v1.0.0/trouble-shooting/)

List the Meridio resources: trenches, conduits, streams, flows, vips, attractors, gateways

```
# Kubectl
kubectl api-resources # List resources in the cluster
kubectl get pods --all-namespaces -o wide # Get all pods in the cluster
kubectl exec -it <pod-name> -n <namespace> -- <command> # Execute a command in a pod
kubectl logs <pod-name> -c <container-name> # Get the logs of a container in a pod
kubectl get <resource> <resource-name> -o yaml # Get resource (e.g. pod) in yaml format 
kubectl describe pods <pod-name> # Describe a pod

# NFQLB (in stateless-lb-frontend)
nfqlb flow-list # Get the flow configured in NFQLB
nfqlb show --shm=tshm-stream-a-i # Get the stream-a-i configuration in NFQLB

# VPP (in forwarder)
vppctl show interface addr # List VPP interfaces 
vppctl show bridge-domain # List bridges
vppctl show tap # List tap interfaces

# Networking
ip a # List interfaces
ip rule # List rules (ip -6 rule for IPv6)
ip route # List routes (ip -6 route for IPv6)
ip route show table 1 # List routes in table 1
nft list ruleset # List nftables

# tcpdump
tcpdump -nn -i any port 4000 # Capture traffic on any interface with 4000 as destination port
tcpdump -nn -i eth0 -e 'dst port 4000 and (vlan 100)' # Capture traffic on eth0 with vlan id 100 and 4000 as destination port
tcpdump -nn -i eth0 'port 4789 and udp[8:2] = 0x0800 & 0x0800 and udp[11:4] = 9832580 & 0x00FFFFFF' # Capture vxlan traffic with 9832580 as vxlan ID and 4789 as port
```

# Solution

Click on "NEXT" to find the solution.