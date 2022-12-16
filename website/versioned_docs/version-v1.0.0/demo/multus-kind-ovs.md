# Demo - Kind - OVS CNI

This demo deploys a Kubernetes with 2 workers and 1 master running Spire, Network Service Mesh and a single Meridio trench using Meridio Operator. The external connectivity is provided through Multus using a Network Attachment Definition relying on [Open vSwitch CNI](https://github.com/k8snetworkplumbingwg/ovs-cni) and [whereabouts](https://github.com/k8snetworkplumbingwg/whereabouts) IPAM CNI plugin. The traffic is attracted by a vlan connected to a gateway (also used as traffic generator).

A secondary Kind network is also created to get consumed by an OVS bridge and used for external connectivity.

## Installation

### Kubernetes cluster

Deploy a Kubernetes cluster with Kind
```
kind create cluster --config docs/demo/kind.yaml
```

### Secondary Kind network

Create new Kind network `meridio-net`
```
docker network create -d bridge meridio-net --opt com.docker.network.driver.mtu=1500
```

Connect nodes to network
```
docker network connect meridio-net kind-control-plane
docker network connect meridio-net kind-worker
docker network connect meridio-net kind-worker2
```

### NSM

Deploy Spire
```
kubectl apply -k docs/demo/deployments/spire
```

Deploy NSM with OVS Forwarder (supplying OVS to each worker)
```
helm install docs/demo/deployments/nsm-ovs --generate-name --create-namespace --namespace nsm
```

### Setup OVS bridge

Create and configure OVS bridge `br-meridio` on each worker (move `eth1` and its IP addresses to the bridge)
```
fwds=$(kubectl get pods --selector=app=forwarder-ovs -n nsm -o jsonpath={.items[*].metadata.name})
for f in $fwds; do \
    echo "setup $f" && \
    ips=$(kubectl exec $f -n nsm -- ip address show dev eth1|grep inet|grep -v fe80|awk '{print $2}'|xargs) && \
    kubectl exec $f -n nsm -- ovs-vsctl add-br br-meridio && \
    kubectl exec $f -n nsm -- ovs-vsctl add-port br-meridio eth1 && \
    kubectl exec $f -n nsm -- ip link set dev br-meridio up && \
    kubectl exec $f -n nsm -- ip addr flush dev eth1 && \
    for ip in $ips; do \
        ipv="-4"; test "${ip#*:}" != "$ip" && ipv="-6"; \
        kubectl exec $f -n nsm -- ip $ipv address add $ip dev br-meridio; \
    done; \
done
```

Note: This demo does not require the IPs to be moved, and OVS bridge does not seem to require setting link state UP,
so those can be skipped:
```
kubectl get pods --selector=app=forwarder-ovs -n nsm -o jsonpath={.items[*].metadata.name}| \
    cut -d " " -f1- --output-delimiter=$'\n '| \
    xargs -I {} -- sh -c 'echo "setup {}" && kubectl exec {} -n nsm -- ovs-vsctl add-br br-meridio && \
    kubectl exec {} -n nsm -- ovs-vsctl add-port br-meridio eth1'
```

### Multus

Install Multus
```
kubectl apply -f https://raw.githubusercontent.com/Nordix/xcluster/master/ovl/multus/multus-install.yaml
```

### OVS CNI

Install OVS CNI plugin
```
git clone git@github.com:k8snetworkplumbingwg/ovs-cni.git && cd ovs-cni
kubectl apply -f examples/ovs-cni.yml
```

### whereabouts

Install whereabouts IPAM CNI plugin (must include dual-stack support)
```
git clone https://github.com/k8snetworkplumbingwg/whereabouts && cd whereabouts
kubectl apply \
    -f doc/crds/daemonset-install.yaml \
    -f doc/crds/whereabouts.cni.cncf.io_ippools.yaml \
    -f doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml
```

### Network Attachment Definition

Add Network Attachment Definition that uses OVS CNI to create access ports (VLAN 100). IP address allocation is ensured by whereabouts IPAM CNI plugin (exluding IPs of the gateway).
```
cat <<EOF | kubectl apply -f -
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: meridio-nad
  namespace: default
spec:
  config: '{
      "cniVersion": "0.4.0",
      "name": "myovsnet",
      "plugins": [
        {
          "type":"ovs",
          "name": "myovs",
          "bridge": "br-meridio",
          "vlan": 100,
          "ipam": {
            "log_file": "/tmp/whereabouts.log",
            "type": "whereabouts",
            "ipRanges": [{
              "range": "169.254.100.0/24",
              "exclude": [
                "169.254.100.150/32"
              ]
            }, {
              "range": "100:100::/64",
              "exclude": [
                "100:100::150/128"
              ]
            }]
          }
        }
      ]
  }'
EOF
```

### Meridio

Deploy Meridio Operator
```
make deploy OPERATOR_NAMESPACE="red"
```

Install Meridio via Meridio Operator
```
kubectl apply -f docs/demo/multus-meridio.yaml -n red
```

Note: No NSE-VLAN will be deployed as part of the Attractor and no NSC container is started for stateless-lb-frontend
```
kubectl get pods -n red
NAME                                            READY   STATUS    RESTARTS   AGE
ipam-trench-a-0                                 1/1     Running   0          13m
meridio-operator-77f47dc748-mbq89               1/1     Running   0          14m
nsp-trench-a-0                                  1/1     Running   0          13m
proxy-load-balancer-a1-8dx9l                    1/1     Running   0          13m
proxy-load-balancer-a1-8rgs9                    1/1     Running   0          13m
stateless-lb-frontend-attr-1-79787dbb8f-rhm6n   2/2     Running   0          13m
stateless-lb-frontend-attr-1-79787dbb8f-tcvpk   2/2     Running   0          13m
```

### Target

Install targets
```
helm install examples/target/deployments/helm/ --generate-name --create-namespace --namespace red --set applicationName=target-a \
    --set default.trench.name=trench-a --set default.stream.name=stream-1 --set default.conduit.name=load-balancer-a1
```

### External host / External connectivity

Deploy an external host (Gateway-Router) connected to the secondary Kind network
```
sed -e "s|\(\s*\)docker run -t -d --network=\"[0-9a-zA-Z-]\+|\1docker run -t -d --network=\"meridio-net|g" ./docs/demo/scripts/kind/external-host.sh | source /dev/stdin
```

## Traffic

Connect to a external host (trench-a)
```
docker exec -it trench-a bash
```

Generate traffic
```
# ipv4
ctraffic -address 20.0.0.1:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
# ipv6
ctraffic -address [2000::1]:5000 -nconn 400 -rate 100 -monitor -stats all > v4traffic.json
```

Verification
```
ctraffic -analyze hosts -stat_file v4traffic.json
```
