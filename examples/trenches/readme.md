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

### Parameters

Set vlan ID and subnet
```
# IPv4
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-a --set vlanID=100 --set vlanIPv4Prefix=169.254.100.0/24
# IPv6
helm install deployments/helm/ --generate-name --create-namespace --namespace trench-a --set vlanID=100 --set vlanIPv6Prefix=100:100::/64
```
