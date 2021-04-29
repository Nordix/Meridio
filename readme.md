# Meridio

- [Demo instructions](https://github.com/Nordix/Meridio/tree/master/docs/demo/)
- [Documentation](https://github.com/Nordix/Meridio/tree/master/docs)
- [Multi-trenches example](https://github.com/Nordix/Meridio/tree/master/examples/trenches)

List of projects:
1. ctraffic
2. ipam
3. load-balancer
4. nsp
5. proxy
6. target

* Default version: latest
* Default container registry: localhost:5000/meridio


Build, tag and push all projects
```
make
```

Build, use specific tags and push all projects to a specific container registry
```
make VERSION=0.1 REGISTRY=localhost:5005/meridio
```

Build specific project
```
make build IMAGE=proxy
```

Tag specific project
```
make tag VERSION=0.1 IMAGE=proxy
```

Push specific project
```
make push VERSION=0.1 IMAGE=proxy
```

Generate API code from proto files
```
make proto
```

Check code
```
make check
```

Deploy
```
helm install deployments/helm/ --generate-name
```