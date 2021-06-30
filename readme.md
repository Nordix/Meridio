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
make docker-build docker-push IMG="localhost:5000/meridio/meridio-operator:v0.0.1"
```

Deploy cert manager
```
kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
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