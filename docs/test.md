# Test

## E2E Tests

![meridio-e2e-kind-meridio](https://img.shields.io/endpoint?url=https%3A%2F%2Fjenkins.nordix.org%2Fjob%2Fmeridio-e2e-test-kind%2FlastCompletedBuild%2Fartifact%2F_output%2Fmeridio-e2e-kind-meridio.json)

![meridio-e2e-kind-tapa](https://img.shields.io/endpoint?url=https%3A%2F%2Fjenkins.nordix.org%2Fjob%2Fmeridio-e2e-test-kind%2FlastCompletedBuild%2Fartifact%2F_output%2Fmeridio-e2e-kind-tapa.json)

![meridio-e2e-kind-nsm](https://img.shields.io/endpoint?url=https%3A%2F%2Fjenkins.nordix.org%2Fjob%2Fmeridio-e2e-test-kind%2FlastCompletedBuild%2Fartifact%2F_output%2Fmeridio-e2e-kind-nsm.json)

![meridio-e2e-kind-ip-family](https://img.shields.io/endpoint?url=https%3A%2F%2Fjenkins.nordix.org%2Fjob%2Fmeridio-e2e-test-kind%2FlastCompletedBuild%2Fartifact%2F_output%2Fmeridio-e2e-kind-ip-family.json)

![meridio-e2e-kind-kubernetes](https://img.shields.io/endpoint?url=https%3A%2F%2Fjenkins.nordix.org%2Fjob%2Fmeridio-e2e-test-kind%2FlastCompletedBuild%2Fartifact%2F_output%2Fmeridio-e2e-kind-kubernetes.json)

(These reports are for the last 1000 test runs only)

### Environment / Framework

#### Initial Deployment

The picture below shows the initial deployment that should be installed in a kubernetes cluster in order to execute the complete e2e test suite in dualstack. With only IPv4, elements containing `v6` are not used, and with only IPv6, elements containing `v4` are not used. Elements between `[]` are configurable via parameters, see the `Configuration` section. 

![Initial-Deployment-E2E](resources/Initial-Deployment-E2E.svg)

#### Configuration

| Name | Type | Description |
|---|---|---|
| traffic-generator-cmd | string | Command to use to connect to the traffic generator. All occurences of '{trench}' will be replaced with the trench name. |
| script | string | Path + script used by the e2e tests |
| skip | string | Skip specific tests |
| focus | string | Focus on specific tests |
| log-collector-enabled | bool | Is log collector enabled |
|  |  |  |
| k8s-namespace | string | Name of the namespace |
| target-a-deployment-name | string | Name of the target deployment |
| trench-a | string | Name of the trench |
| attractor-a-1 | string | Name of the attractor |
| conduit-a-1 | string | Name of the conduit |
| stream-a-I | string | Name of the stream |
| stream-a-II | string | Name of the stream |
| flow-a-z-tcp | string | Name of the flow |
| flow-a-z-tcp-destination-port-0 | int | Destination port 0 |
| flow-a-z-udp | string | Name of the flow |
| flow-a-z-udp-destination-port-0 | int | Destination port 0 |
| flow-a-x-tcp | string | Name of the flow |
| flow-a-x-tcp-destination-port-0 | int | Destination port 0 |
| vip-1-v4 | string | Address of the vip v4 number 1 |
| vip-1-v6 | string | Address of the vip v6 number 1 |
| target-b-deployment-name | string | Name of the target deployment |
| trench-b | string | Name of the trench |
| conduit-b-1 | string | Name of the conduit |
| stream-b-I | string | Name of the stream |
| vip-2-v4 | string | Address of the vip v4 number 2 |
| vip-2-v6 | string | Address of the vip v6 number 2 |
|  |  |  |
| stateless-lb-fe-deployment-name-attractor-a-1 | string | Name of stateless-lb-fe deployment in `attractor-a-1` |
| stateless-lb-fe-deployment-name-attractor-b-1 | string | Name of stateless-lb-fe deployment in `attractor-b-1` |
| ip-family | string | IP Family |

For more details about each parameter, check the picture above in the `Initial Deployment` section.

#### Script

A bash script file must be passed as parameter of the e2e tests. The script is required to allowed the e2e tests to be run in every environment (Helm/Operator deployement...). The following functions has to be implemented in the script:

| Name | Description |
|---|---|
| init () error | Executed once before running the tests |
| end () error | Executed once after running the tests |
| on_failure () error | Executed on failure |
| configuration_new_vip () error | Executed just before running the `new-vip` test |
| configuration_new_vip_revert () error | Executed just after running the `new-vip` test |

### List of tests

| Name | Type | Description |
|---|---|---|
| TCP-IPv4 | IngressTraffic | Send TCP traffic in `trench-a` with `vip-1-v4` as destination IP and `flow-a-z-tcp-destination-port-0` as destination port |
| TCP-IPv6 | IngressTraffic | Send TCP traffic in `trench-a` with `vip-1-v6` as destination IP and `flow-a-z-tcp-destination-port-0` as destination port |
| UDP-IPv4 | IngressTraffic | Send UDP traffic in `trench-a` with `vip-1-v4` as destination IP and `flow-a-z-udp-destination-port-0` as destination port |
| UDP-IPv6 | IngressTraffic | Send UDP traffic in `trench-a` with `vip-1-v6` as destination IP and `flow-a-z-udp-destination-port-0` as destination port |
| MT-Switch | MultiTrenches | Disconnect a target from `target-a-deployment-name` from `trench-a` and connect it to `trench-b` |
| MT-Parallel | MultiTrenches | Send traffic in `trench-a` and `trench-b` at the same time |
| Scale-Down | Scaling | Scale down `target-a-deployment-name` |
| Scale-Up | Scaling | Scale up `target-a-deployment-name` |
| close-open | TAPA | Close `stream-a-I` in one of the target from `target-a-deployment-name` and re-open it |
| new-vip | Configuration | Configure `vip-2-v4` and `vip-2-v6` in `flow-a-z-tcp` and `attractor-a-1` |
<!-- TODO: | open | TAPA | Open `stream-a-II` in one of the target from `target-a-deployment-name` and close it | -->

### Steps (Kind + Helm)

1. Deploy environment (Kind + Gateways + NSM + Spire) and Meridio (trench-a + trench-b + target-a + target-b)

```bash
make -s -C test/e2e/environment/kind-helm/ KUBERNETES_VERSION=v1.25 NSM_VERSION=v1.6.1 KUBERNETES_IP_FAMILY=dualstack KUBERNETES_WORKERS=2
```

2. Run e2e tests

```bash
make e2e
```

3. Uninstall environment
```bash
make -s -C docs/demo/scripts/kind/ clean
```
