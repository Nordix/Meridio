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
- Cert-manager
- Meridio Operator

#### Test Execution

Run all e2e tests:
```
make e2e
```

To execute a single test, the E2E_FOCUS env variable in the Makefile can be used:
```
make e2e E2E_FOCUS="Validation"
```
3 test suites are available:
- Validation
- Trench
- Attractor
