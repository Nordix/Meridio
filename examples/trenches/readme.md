# Trenches example

## Deployment instructions

Configure Spire
```
./examples/trenches/scripts/spire-config.sh
```

Deploy Meridio in namespace (trench): trench-a
```
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-a
```

Deploy Meridio in namespace (trench): trench-b
```
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-b
```
