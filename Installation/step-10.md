# Send traffic

Traffic can now be sent from the traffic generator.

For IPv4:
* `20.0.0.1` is the VIP `vip-a-1-v4` which has been configured in `flow-a-z-tcp` and `attractor-a-1`
* `4000` is the port opened in flow `flow-a-z-tcp`
```
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
```{{exec}}

For IPv6:
* `2000::1` is the VIP `vip-a-1-v6` which has been configured in `flow-a-z-tcp` and `attractor-a-1`
* `4000` is the port opened in flow `flow-a-z-tcp`
```
docker exec -it trench-a mconnect -address  [2000::1]:4000 -nconn 400 -timeout 2s
```{{exec}}

# Verify

Get the IPs of the Gateway traffic generator:
* `vlan0` is the interface on VLAN 100
* `169.254.100.150` is the IP configure in gateway `gateway-a-1-v4`
* `100:100::150` is the IP configure in gateway `gateway-a-1-v6`
```
docker exec -it trench-a ip a
```{{exec}}

Get the Routes of the Gateway traffic generator:
* New routes have been added for the 2 VIPs: `vip-a-1-v4` and `vip-a-1-v6`
```
docker exec -it trench-a ip route
docker exec -it trench-a ip -6 route
```{{exec}}

Get all custom resources previously deployed:
```
kubectl get trench -n red
kubectl get vips -n red
kubectl get conduits -n red
kubectl get attractors -n red
kubectl get gateways -n red
kubectl get streams -n red
kubectl get flows -n red
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-9.svg)