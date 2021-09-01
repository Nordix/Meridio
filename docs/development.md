# Development

## Testing

### E2E tests

#### Tools

- ginkgo - https://onsi.github.io/ginkgo/#getting-ginkgo

#### Environment

Before executing the tests, this environment should be deployed:
- 1 Host for each trench with the network interface configured and ctraffic available
- 2 Worker nodes
- NSM 1.0
- Spire configured for NSM, the trench-a and the trench-b
- 2 identical trenches ("trench-a" and "trench-b") running in namespace "red"
    - 2 LBs
    - 4 Targets
    - 2 VIPs
        - 20.0.0.1/32
        - 2000::1/128
On kind, this environment can be deployed by following the steps described in the [demo instructions](https://github.com/Nordix/Meridio/tree/master/docs/demo/).

#### Test Execution

Run all e2e tests:
```
make e2e
```

By default, the traffic is sent from a container named with the name of the trench ("docker exec -i {trench}"). This can be changed by using the TRAFFIC_GENERATOR_CMD env variable in the Makefile.
```
make e2e TRAFFIC_GENERATOR_CMD="docker exec -i {trench}"
```

To execute a single test, the E2E_FOCUS env variable in the Makefile can be used:
```
make e2e E2E_FOCUS="Attractor"
```
4 test suites are available:
- IngressTraffic
- MultiTrenches
- Scaling
- Target
