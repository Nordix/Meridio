# Target example application

## Deploy

Deploy the common helm chart
```
helm install examples/target/common/ --generate-name --create-namespace --namespace my-app
```

Deploy application target-a with trench-a as default trench
```
helm install examples/target/deployments/helm/ --generate-name --create-namespace --namespace my-app --set ipFamily=dualstack --set applicationName=target-a --set defaultTrench=trench-a
```

Deploy application target-b with trench-b as default trench
```
helm install examples/target/deployments/helm/ --generate-name --create-namespace --namespace my-app --set ipFamily=dualstack --set applicationName=target-b --set defaultTrench=trench-b
```

## Target Client

Open a stream
```
./target-client open -t trench-a -c load-balancer -s stream-a
```

Close a stream
```
./target-client close -t trench-a -c load-balancer -s stream-a
```

Watch stream events (on each event the full list is sent with the status of each stream)
```
./target-client watch
```