# Deploy

Apply the Meridio CRDs v1.0.0 Helm Chart
```
helm install meridio-crds https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-CRDs-v1.0.0.tgz --create-namespace --namespace red
```{{exec}}

Apply the Meridio Operator v1.0.0 Helm Chart
```
helm install meridio https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-v1.0.0.tgz --create-namespace --namespace red --set registry="registry.gitlab.com" --set repository="lionelj/meridio" --set nsm.repository="lionelj/meridio"
```{{exec}}

# Verify

Now the Meridio CRDs and the operator have been applied, here is what has been deployed:

7 new custom resource definition (trench, conduit, stream, flow, vip, attractor and gateway)
```
kubectl api-resources | grep meridio
```{{exec}}

2 configmaps:
* meridio-deployment-templates: Contains the templates used to deploy the child resources of the custom resources (stateless-lb-frontend, proxy, nsp...)
* meridio-resource-templates: Contains the templates of the resource limits and requests
```
kubectl get -n red configmap
```{{exec}}

1 deployment with 1 replica:
* meridio-operator: The Meridio Operator itself
```
kubectl get -n red deployment
kubectl get -n red pods
```{{exec}}

4 Roles:
* meridio-fes-role: Will be used by the frontend container in the stateless-lb-frontend pods to read the secrets for BGP authentication
* meridio-leader-election-role: (optional) Used by the operator to run multiple replicas
* meridio-nsp-role: Will be used by the NSP pods to watch the configmaps for configuration
* meridio-operator-role: Used by the operator to manage resources
```
kubectl get -n red roles
```{{exec}}

4 Role Bindings:
* meridio-fes-rolebinding: Binds `meridio-fes-role` Role to `meridio-fes` service account
* meridio-leader-election-rolebinding: Binds `meridio-leader-election-role` Role to `meridio-operator` service account
* meridio-nsp-rolebinding: Binds `meridio-nsp-role` Role to `meridio-fes` service account
* meridio-operator-rolebinding: Binds `meridio-operator-role` Role to `meridio-operator` service account
```
kubectl get -n red rolebindings
```{{exec}}

1 Service:
* meridio-operator-webhook-service: Used to serve the `meridio-operator-validating-webhook-configuration-red` validating webhook
```
kubectl get -n red service
```{{exec}}

3 Service account:
* meridio-fes: Will be applied to stateless-lb-frontend pods
* meridio-nsp: Will be applied to nsp pods
* meridio-operator: Applied to the operator pod
```
kubectl get -n red sa
```{{exec}}

1 validating webhook:
* meridio-operator-validating-webhook-configuration-red: validating webhook for all new custom resources (trench, conduit, stream, flow, vip, attractor and gateway)
```
kubectl get validatingwebhookconfigurations
```{{exec}}

# Cluster State

Here is a picture of the Kubernetes cluster with the resources currently deployed:

![step](https://raw.githubusercontent.com/LionelJouin/Meridio-Killercoda/main/Installation/assets/step-1.svg)
