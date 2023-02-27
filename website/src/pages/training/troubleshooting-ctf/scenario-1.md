# Scenario 1 - Traffic does not reach the application pods (target pods)

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
   * All pods should be in the *running state*, which means that all the pods have been bound to a node, and all of the containers have been created. The containers inside the pods are running, or are in the process of starting or restarting.

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
    * All pods should be in the *running state*, which means that all the pods have been bound to a node, and all of the containers have been created. The containers inside the pods are running, or are in the process of starting or restarting.

```bash
kubectl get pods -n=red
NAME                                                  READY   STATUS    RESTARTS   AGE
ipam-trench-a-0                                       1/1     Running   0          5m33s
meridio-operator-596d7f88b8-9f79p                     1/1     Running   0          6m37s
nse-vlan-attractor-a-1-5cf67947d5-wqhnd               1/1     Running   0          5m34s
nsp-trench-a-0                                        1/1     Running   0          5m34s
proxy-conduit-a-1-dlz75                               1/1     Running   0          5m32s
proxy-conduit-a-1-p9nlq                               1/1     Running   0          5m32s
stateless-lb-frontend-attractor-a-1-d8db96c8f-g82b6   3/3     Running   0          5m34s
stateless-lb-frontend-attractor-a-1-d8db96c8f-rmffb   3/3     Running   0          5m34s
target-a-77b5b48457-2m5j9                             2/2     Running   0          4m51s
target-a-77b5b48457-69nhb                             2/2     Running   0          4m51s
target-a-77b5b48457-9w2vk                             2/2     Running   0          4m51s
target-a-77b5b48457-tpj8r                             2/2     Running   0          4m51s
```

3. In the case of the current scenario, all the conditions from the previous steps are met. The next step is to check the deployed custom resources for any sort of misconfiguration. The suggestion would be to start checking resources in the reversed order of their deployment, i.e. at first check the configuration of the `flow`.
   * There are several properties of the `flow` custom resource, which probably should be checked thoroughly since they are a common source of errors if configured incorrectly. Particularly, check if **namespace**, **trench**, **destination-ports**, **protocols**, **source-ports**, **stream**, **vips** are set to the correct values following the specified configuration (*reference: known inputs*).
   * Going through the `flow-a-z-tcp` detailed description presented below, and verifying the mentioned properties, it can be noticed that there is a misconfiguration in the **destination-ports** field. The destination port of the target pod is `4000`, however in the current deployment it is set to `400`, therefore traffic can not reach the intended destination.

```yaml
kubectl get flow flow-a-z-tcp -n=red -o=yaml

apiVersion: meridio.nordix.org/v1
kind: Flow
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"meridio.nordix.org/v1","kind":"Flow","metadata":{"annotations":{},"labels":{"trench":"trench-a"},"name":"flow-a-z-tcp","namespace":"red"},"spec":{"destination-ports":["400"],"priority":1,"protocols":["tcp"],"source-ports":["any"],"source-subnets":["0.0.0.0/0","0:0:0:0:0:0:0:0/0"],"stream":"stream-a-i","vips":["vip-a-1-v4","vip-a-1-v6"]}}
  creationTimestamp: "2023-02-21T15:10:28Z"
  generation: 1
  labels:
    trench: trench-a
  name: flow-a-z-tcp
  namespace: red
  ownerReferences:
  - apiVersion: meridio.nordix.org/v1
    kind: Trench
    name: trench-a
    uid: 9dc15ad5-ff23-491b-aed4-d23ce11cecd8
  resourceVersion: "1318"
  uid: d44e87c8-6564-4aad-a6bb-b3c36814a61e
spec:
  destination-ports:
  - "400"
  priority: 1
  protocols:
  - tcp
  source-ports:
  - any
  source-subnets:
  - 0.0.0.0/0
  - 0:0:0:0:0:0:0:0/0
  stream: stream-a-i
  vips:
  - vip-a-1-v4
  - vip-a-1-v6
```

4. Changing the destination port to `4000` fixes the traffic issue which was reported by the user.

```bash
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
Failed connects; 0
Failed reads; 0
target-a-77b5b48457-gqgdx 101
target-a-77b5b48457-7wg8j 98
target-a-77b5b48457-s5nnq 110
target-a-77b5b48457-sz8r8 91

docker exec -it trench-a mconnect -address  [2000::1]:4000 -nconn 400 -timeout 2s
Failed connects; 0
Failed reads; 0
target-a-77b5b48457-pdtq5 90
target-a-77b5b48457-72vlp 109
target-a-77b5b48457-qsrwg 110
target-a-77b5b48457-n46p4 91
```

### Helpful Resources

* [Meridio Flow Concept](https://meridio.nordix.org/docs/v1.0.0/concepts#flow)
* [Meridio Troubleshooting Guide](https://meridio.nordix.org/docs/v1.0.0/trouble-shooting/)
* [Kubectl Reference Docs](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands)
* [mconnect](https://github.com/Nordix/mconnect)