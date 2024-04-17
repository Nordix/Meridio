# Spire yaml generation

The yaml files have been generated with these commands:
```
helm install -n spire --create-namespace my-spire-crds spiffe/spire-crds --version 0.4.0 --dry-run
helm install -n spire --create-namespace my-spire spiffe/spire --version 0.20.0 -f docs/demo/deployments/spire/values.yaml --dry-run
```

`"webhook_label": "spiffe.io/webhook",` has been added to Notifier.k8sbundle.plugin_data in the spire-server configmap. 
This ticket would solve the problem: https://github.com/spiffe/helm-charts-hardened/issues/47

`"mutatingwebhookconfigurations"` next to the validatingwebhookconfigurations resource has been added to the spire-controller-manager ClusterRole.
