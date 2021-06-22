# Target example application

Deploy the common helm chart
```
helm install examples/target/common/ --generate-name --create-namespace --namespace my-app
```

Deploy application target-a with trench-a as default trench
```
helm install examples/target/helm/ --generate-name --create-namespace --namespace my-app --set ipFamily=dualstack --set applicationName=target-a --set defaultTrench=trench-a
```

Deploy application target-b with trench-b as default trench
```
helm install examples/target/helm/ --generate-name --create-namespace --namespace my-app --set ipFamily=dualstack --set applicationName=target-b --set defaultTrench=trench-b
```