# Focus

We are currently focusing on this part of the traffic:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Traffic-Path/assets/step-8.svg)

The names of the interfaces, IDs.. in the running cluster are probably different from this picture.

You can find all pictures of the traffic path [here](https://viewer.diagrams.net/?tags=%7B%7D&highlight=0000ff&edit=_blank&layers=1&nav=1&page-id=rjszOReYDxTjH4DNYqVc&title=Diagrams%20-%20Traffic%20Path#Uhttps%3A%2F%2Fdrive.google.com%2Fuc%3Fid%3D1QRx1kS7n7Rnhc_FoJKpxiXhpXqHPYLKR%26export%3Ddownload)

# stateless-lb-frontend (Second worker node)

Get stateless-lb-frontend on second node:
```
STATELESS_LB_FRONTEND_2=$(kubectl get pods -l app=stateless-lb-frontend-attractor-a-1 -n red --field-selector spec.nodeName=kind-worker2 --no-headers=true | awk '{print $1}')
```{{exec}}

List the ip rules for IPv4:
```
kubectl exec -it $STATELESS_LB_FRONTEND_2 -n red -- ip rule
```{{exec}}
There is a rule matching the VIP as source IP address, the rule has a corresponding table.

List the route for a table for IPv4:
* The route correspond to the IP of the Gateway
```
kubectl exec -it $STATELESS_LB_FRONTEND_2 -n red -- ip route show table 4096
```{{exec}}
The IP of the nexthop correspond to the IP of the Gateway.

List the ip rules for IPv6:
```
kubectl exec -it $STATELESS_LB_FRONTEND_2 -n red -- ip -6 rule
```{{exec}}

List the route for a table for IPv6:
```
kubectl exec -it $STATELESS_LB_FRONTEND_2 -n red -- ip -6 route show table 4096
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
