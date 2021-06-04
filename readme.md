# Meridio-Operator

[Meridio](https://github.com/Nordix/Meridio)

Run tests, generate code and objects
```
make test
```

Build image
```
make docker-build

# And push the image to a registry
make docker-builddocker-push IMG="localhost:5000/meridio/meridio-operator:v0.0.1"
```

Deploy
```
make deploy

# Use a specific image
make deploy IMG="localhost:5000/meridio/meridio-operator:v0.0.1"
```

### Example

Deploy Trench
```
kubectl apply -f ./config/samples/meridio_v1alpha1_trench.yaml
```