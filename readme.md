# Meridio-Operator

[Meridio](https://github.com/Nordix/Meridio)

Run tests, generate code and objects

```bash
make test
```

Build image

```bash
make docker-build

# And push the image to a registry
make docker-build docker-push IMG="localhost:5000/meridio/meridio-operator:v0.0.1"
```

Deploy cert manager

```bash
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

Deploy

```bash
make deploy

# Use a specific image
make deploy IMG="localhost:5000/meridio/meridio-operator:v0.0.1"
```

## Example

### Trench

A trench resource does not have any parameters in its specification.
Deploying a trench spawns the IPAM, NSP, Proxy, and needed role, role-binding and service accounts. The resouces created by a trench will be suffixed with the trench's name.

```bash
kubectl apply -f ./config/samples/meridio_v1alpha1_trench.yaml
```

After applying, you should able to see the following instances in the cluster

```bash
kubectl get trench
NAME       AGE
trench-a   1m

kubectl get all
NAME                                  READY   STATUS        RESTARTS   AGE
pod/ipam-trench-a-67474f4bf8-nt964    1/1     Running       0          1m
pod/nsp-trench-a-799984cfb5-r4g8f     1/1     Running       0          1m
pod/proxy-trench-a-dlc5c              1/1     Running       0          1m

NAME                            TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/ipam-service-trench-a   ClusterIP   10.96.6.85     <none>        7777/TCP   1m
service/nsp-service-trench-a    ClusterIP   10.96.71.187   <none>        7778/TCP   1m

NAME                              DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
daemonset.apps/proxy-trench-a     1         1         1       1            1           <none>          1m

NAME                            READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/ipam-trench-a   1/1     1            1           1m
deployment.apps/nsp-trench-a    1/1     1            1           1m
```

### Attractor

An attractor should have owner reference to be a trench, and that is realized by specifying `trench` label in metadata, and it's mandatory and immutable. If the labeled trench has not been created in the cluster, the status `lb-fe` will be marked as `disengaged`, thereafter, no instance will be created in the cluster. To be noted, there is a limitation of the order of creating resouces, meaning if the trench is created after the attractor, the status will not be updated.

`vlan-id` and `vlan-interface` parameters are mandatory and immutable parameters.
If you want to change any of the immutable paramters, the attractor should be deleted and created with new values.
Attactor specifies a list of gateway and vip items that it expects to use. If any of the items is in the cluster, they will be fetched by the attractor to use. The gateway and vip items that attractors are using are expressed by `status.gateways-in-use` and `vips-in-use`.

An attractor resource that passes all the validation will initiate the creation of LB-FE, NSE-VLAN. The resources created will be suffixed with either trench.

To be noted, meridio-operator have a limitation to have one attractor each trench.

```bash
kubectl apply -f ./config/samples/meridio_v1alpha1_attractor.yaml
```

After applying, you should able to see the following instances added in the cluster

```bash
kubectl get all
NAME                                  READY   STATUS    RESTARTS   AGE
pod/lb-fe-trench-a-786c5979b8-8vdjh   3/3     Running   0          5s
pod/nse-vlan-attr1-7844574dc-dlgkr    1/1     Running   0          5s

NAME                             READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/lb-fe-trench-a   1/1     1            1           5s
deployment.apps/nse-vlan-attr1   1/1     1            1           5s

kubectl get attractor
NAME                                 VLANID   VLANITF   GATEWAYS                  GW-IN-USE   VIPS              VIPS-IN-USE   TRENCH     LB-FE
attractor.meridio.nordix.org/attr1   100      eth0      ["gateway1","gateway3"]               ["vip1","vip2"]                 trench-a   engaged

```

### Gateway

A gateway should specify its owner reference attractor and trench by `attractor` label. Likewise, this label is immutable. If the attractor is not found in the cluster, the status of gateway resource will be verdicted as `disengaged`, and no attractor will not use it. Similarly, the status will not be updated if the attractor is created afterwards. There is a limitation of order also stands for attractors and gateways. Attractors should be created in advance than the gateways.

In gateway custom resource `address` is a mandatory parameter.
`bfd` is set to false by default, and that is the only supported value too.
`protocol` currently supports `bgp` only, and that is also set by default.

In the example below, two gateways will be created

```bash
kubectl apply -f ./config/samples/meridio_v1alpha1_gateway.yaml
```

The following resources should be found

```bash
kubectl get gateways
NAME                                  ADDRESS   PROTOCOL   BFD     ATTRACTOR   STATUS    MESSAGE
gateway.meridio.nordix.org/gateway1   2.3.4.5   bgp        false   attr1       engaged
gateway.meridio.nordix.org/gateway2   1000::1   bgp        false   attr1       engaged
```

And the *GW-IN-USE* coloumn of the attractor should be updated with the existing expected gateways. Shown as below

```bash
NAME                                 VLANID   VLANITF   GATEWAYS                  GW-IN-USE                 VIPS              VIPS-IN-USE   TRENCH     LB-FE
attractor.meridio.nordix.org/attr1   100      eth0      ["gateway1","gateway2"]   ["gateway1","gateway2"]   ["vip1","vip2"]                 trench-a   engaged
```

### Vip

A vip should belong to one `attractor` and `trench`, which specified with labels, same as gateway. Also they need to follow an order that trench and attractor are created before the vips. Otherwise the status will be verdicted as `disengaged`, and cannot be revised if labeled trench or attractor are created afterwards.

In vip resource, `address` is a mandatory paramter.

In the example below, two gateways will be created

```bash
kubectl apply -f ./config/samples/meridio_v1alpha1_vip.yaml
```

The following resources should be found

```bash
kubectl get vip
NAME                          ADDRESS       STATUS
vip.meridio.nordix.org/vip1   20.0.0.1/32   engaged
vip.meridio.nordix.org/vip2   10.0.0.1/32   engaged
```

The status *VIPS-IN-USE* of gateway will also be updated with the existing expected vips.

```bash
NAME                                 VLANID   VLANITF   GATEWAYS                  GW-IN-USE                 VIPS              VIPS-IN-USE       TRENCH     LB-FE
attractor.meridio.nordix.org/attr1   100      eth0      ["gateway1","gateway2"]   ["gateway1","gateway2"]   ["vip1","vip2"]   ["vip1","vip2"]   trench-a   engaged
```
