# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-5.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# target

Get a target on first node:
```
TARGET=$(kubectl get pods -l app=target-a -n red --field-selector spec.nodeName=kind-worker --no-headers=true | awk '{print $1}' | head -n 1)
```{{exec}}

List the interfaces in the target:
* nsm-0: contains the 2 VIPs: `20.0.0.1/32` and `2000::1/128`
```
kubectl exec -it $TARGET -n red -- ip a
```{{exec}}

List the ip rules for IPv4:
```
kubectl exec -it $TARGET -n red -- ip rule
```{{exec}}
There is a rule matching the VIP as source IP address, the rule has a corresponding table.

List the route for a table for IPv4:
```
kubectl exec -it $TARGET -n red -- ip route show table 1
```{{exec}}
The IP of the nexthop corresponds to the IP of the bridge in the proxy.

List the ip rules for IPv6:
```
kubectl exec -it $TARGET -n red -- ip -6 rule
```{{exec}}

List the route for a table for IPv6:
```
kubectl exec -it $TARGET -n red -- ip -6 route show table 2
```{{exec}}

# Traffic

Send IPv4 traffic with `20.0.0.1` as destination IP and `4000` as destination port:
```
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
```{{exec}}

Create a single TCP connection with `20.0.0.1` as destination IP, `4000` as destination port and `35000` as source port:
```
docker exec -it trench-a timeout 0.5s nc 20.0.0.1 4000 -p 35000
```{{exec}}
