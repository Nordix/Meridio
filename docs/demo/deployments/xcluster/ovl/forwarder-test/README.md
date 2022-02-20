# Xcluster/ovl - forwarder-test

Tests Meridio in `xcluster`. Originally this ovl was testing different NSM
forwarders only, but it has evolved to generic e2e tests with `xcluster`.

The [ovl/nsm-ovs](https://github.com/Nordix/nsm-test/tree/master/ovl/nsm-ovs)
is used for NSM setup and the same network setup is used;

<img src="https://raw.githubusercontent.com/Nordix/nsm-test/master/ovl/nsm-vlan-dpdk/multilan.svg" alt="NSM network-topology" width="70%" />

Three trenches are defined;
```
Trench:  vm iface:     vm-202 iface:  Net:                 VIP:
red      eth2.100      eth3.100       169.254.101.0/24     10.0.0.1
blue     eth2.200      eth3.200       169.254.102.0/24     10.0.0.2
green    eth3.100      eth4.100       169.254.103.0/24     10.0.0.3

Add "1000::1" for ipv6, e.g. 169.254.101.0/24 -> 1000::1:169.254.101.0/120
```


## Usage

Pre-load the local registry if necessary;
```
images lreg_preload k8s-pv
images lreg_preload spire
images lreg_preload nsm-ovs
images lreg_preload forwarder-test
```

The default test ("trench") starts three trenches and test external
connectivity from `vm-202` using [mconnect](https://github.com/Nordix/mconnect).

```
#images lreg_preload .               # Load local registry if necessary
#export xcluster_NSM_FORWARDER=ovs   # default "vpp"
./forwarder-test.sh  # Help printout
./forwarder-test.sh test > $log
# Or
xcadmin k8s_test --cni=calico forwarder-test > $log
```

#### Scaling test

The scaling is tested by changing the `replica` count for the targets
and by disconnect/reconnect targets from the stream. An optional `--cnt`
parameter can be set to repeat the disconnect/reconnect test.

```
./forwarder-test.sh test --cnt=5 scale > $log
```


## Meridio setup

Meridio is started with K8s manifests. Helm charts or the
`Meridio-Operator` are *not* used. The helm-charts are used as base
for the manifests and base manifests can be generated with;

```
./forwarder-test.sh generate_manifests
```

The process of updating the local manifests when the helm charts
changes is not automatic, it has to be done manually.

The trenches all have different configurations and to make things
easier individual configurations are used. This it the one for trench "red";

```bash
export NAME=red
export NS=red
export CONDUIT1=load-balancer
export STREAM1=stream1
export VIP1=10.0.0.1/32
export VIP2=1000::1:10.0.0.1/128
export NSM_SERVICES="trench-red { vlan: 100; via: service.domain.2}"
export NSM_CIDR_PREFIX="169.254.101.0/24"
export NSM_IPV6_PREFIX="1000::1:169.254.101.0/120"
export NSC_NETWORK_SERVICES="kernel://trench-red/nsm-1"
export GATEWAY4=169.254.101.254
export GATEWAY6=1000::1:169.254.101.254
export POD_CIDR=172.16.0.0
```

The configuration is then used with a template to produce the final
manifest to be loaded with `kubectl`. Test manually with;

```
(. default/etc/kubernetes/forwarder-test/red.conf; \
 envsubst < default/etc/kubernetes/forwarder-test/nse-template.yaml) | less
# (the parentheses uses a sub-shall and prevents polluting your env)
```




## Local images

For development local built images can be used. The images are *not*
built in the same way as with `make` in the Meridio top
directory. They contain the same binaries, but start-up is simplified
to make debug easier.

```
./forwarder-test.sh build_image
#./forwarder-test.sh build_image ipam    # To build just one
```

This will compile the meridio programs, build the images and upload
them to the local registry.

To use the local images start the test with `--local`, example;

```
./forwarder-test.sh test --local --trenches=red trench > $log
```


The local images works as the real ones but lacks the probes. The
reason for omitting the probes is to simplify trouble shooting when
the probes fail. To simplify trouble shooting more you can prevent the
meridio program from starting by setting;

```yaml
           env:
           - name: NO_START
              value: "yes"
```

This let you login to the POD and start the program manually.


## Multus

The external interface in the `load-balancer` POD (the `fe` POD actually)
can be injected using Multus instead of NSM with vlan. This may be
more mainstream in some deployments and gives the opportunity so use
any inteface type supported by Multus (except `ipvlan` which can't
support VIPs).

In this example the same vlan interfaces are used, e.g. `eth3.100` but
a difference is that the device must be create in main netns on the
nodes and then Multus "host-device" is used to move the interface to
the `load-balancer` POD and rename it "nsm-1".

```
./forwarder-test.sh test --use-multus > $log
```

We must assign addresses to the external interface in the
`load-balancer` POD. In this example `node-local` ipam is used which
is a tiny script wrapper around `host-local`. IRL this may be DHCP or
[whereabouts](https://github.com/k8snetworkplumbingwg/whereabouts) or
something else.

#### Multus stand-alone test

To configure Multus can be tricky so you can test the Multus setup
without NSM and Meridio;

```
./forwarder-test.sh test multus > $log
```


## Antrea CNI-plugin and forwarder-ovs

The [Antrea](https://github.com/antrea-io/antrea) CNI-plugin uses
`ovs`. It conflicts with `forwarder-ovs`. Easiest and fastest to test
without Meridio;

```
export xcluster_NSM_FORWARDER=vpp
xcadmin k8s_test --cni=antrea forwarder-test nsm > $log  # Works
export xcluster_NSM_FORWARDER=ovs
xcadmin k8s_test --cni=antrea forwarder-test nsm > $log  # FAILS!
xcadmin k8s_test --cni=calico forwarder-test nsm > $log  # Works
```

There are other CNI-plugins that uses `ovs` but only `Antrea` is
currently available in `xcluster`.
