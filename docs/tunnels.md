# Tunnels to emulate secondary networks

To test applications with `Meridio` you need secondary
networks. However, clusters with multiple networks may not be
available. In such clusters tunnels can be used to emulate multiple
networks. The "external" interface in the FE PODs are tunnel endpoints
setup [through an UDP
service](https://github.com/Nordix/k8s-service-tunnel).

## Setup

If the external machine is behind a NAT box you must find you "real"
IP address.

```
myip=$(curl -sq https://api.myip.com | jq -r .ip)
```

This address is given as `tunnel.peer` in the Meridio deployment.
```
MERIDIOD=<path to your Meridio clone>
helm install meridio-trench-a $MERIDIOD/deployments/helm-tunnel \
  --set ipFamily=dualstack --create-namespace --namespace red \
  --set tunnel.peer=$myip
```

The "load-balancer" POD will be stuck in `Init` waiting for packets
from the remote tunnel endpoint. The loadBalancerIP of the UDP
service must be used as address and the egress device must be
specified (otherwise the MTU will be incorrect).

```
lbip=$(kubectl get svc -n red tunnel-trench-a -o json | jq -r .status.loadBalancer.ingress[0].ip)
dev=$(ip -j ro get $lbip | jq -r .[0].dev)
```

Now we can setup the tunnel endpoint on the external machine.
```
sudo ip link add trench-a type vxlan id 100 dev $dev remote $lbip \
  dstport 5533 srcport 5533 5534
sudo ip link set up dev trench-a
sudo ip addr add 169.254.100.254/24 dev trench-a
sudo ip -6 addr add fd00:100::169.254.100.254/120 dev trench-a
ping 169.254.100.1    # (may be needed)
```

When the tunnel endpoint is setup and some traffic is sent to the UDP
service, the load-balancer should continue it's startup and become
"ready".

```
watch kubectl get pods -n red
```

Now we deploy some targets (test applications).
```
helm install meridio-targets-trench-a --namespace red \
  $MERIDIOD/examples/target/deployments/helm
```

Wait until the target pods become ready and test.

```
# Setup route to the VIP address through the tunnel
sudo ip ro add 20.0.0.1 via 169.254.100.1
mconnect -address 20.0.0.1:4000 -nconn 100
```

## Cleanup

```
sudo ip ro del 20.0.0.1
sudo ip link del trench-a
helm uninstall -n red meridio-targets-trench-a
helm uninstall -n red meridio-trench-a
```
