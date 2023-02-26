# Contributing

Welcome to the Meridio project! We appreciate your interest in contributing to our open source project. This guide will help you understand the contribution process and how to get started.

## Code of Conduct

Before getting started, please review our code of conduct here. We strive to maintain a friendly and inclusive environment where everyone can feel welcome and respected.

## Development Environment

Getting a development environment where to run and test Meridio might be complex. To run a simple development environment, follow this guide [here](dev/environment.md).

## Makefile

The default `make` command will build, tag and push all components. You can configure the version with the `VERSION` variable, the registry with the `REGISTRY` variable.
Each component can be build, tag and pushed individually:
* `make base-image`
* `make stateless-lb`
* `make proxy`
* `make tapa`
* `make ipam`
* `make nsp`
* `make example-target`
* `make frontend`
* `make operator`
* `make init`

### Linter

We use `golangci-lint` for code linting. To run the linter, execute the following command from the project root directory:

```
make lint
```

### Generator 

To generate the manifests:
```
make manifests
```

To generate the API Code:
```
make generate-controller
```

To generate the code from proto files:
```
make proto
```

To generate the code from mocks:
```
make generate
```

To generate the Helm Chart:
```
make generate-helm-chart VERSION=latest REGISTRY="registry.nordix.org/cloud-native/meridio" NSM_REPOSITORY="cloud-native/nsm"
```

### Unit Tests

To run unit tests, execute the following command:

```
make test
```

### End-to-End Tests

End-to-end tests are located in the `test/e2e` directory. To run the end-to-end tests, follow this guide [here](dev/testing.md).

### Security Scan

To run the security scanners, follow this guide [here](dev/security-scanning.md).

## Documentation
The project documentation is located in the `docs` directory. Please ensure that your contributions to the documentation are accurate and up-to-date.

The project website is located in the `website` directory. Execute the following command to compile and run the website:
```sh
cd website
npm install
npm run start
```

## Contributing Guidelines

* Please ensure that your contributions are thoroughly tested.
* Follow the existing code style and conventions.
* Ensure that your commit messages are descriptive and meaningful.
* If your contribution introduces new dependencies, please update the project documentation accordingly.

Thank you for considering contributing to Meridio. We appreciate your help and look forward to reviewing your contributions!