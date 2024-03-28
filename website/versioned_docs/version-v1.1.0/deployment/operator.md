# Operator

## Helm

v1.1.0:
* https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-CRDs-v1.1.0.tgz
* https://artifactory.nordix.org/artifactory/cloud-native/meridio/Meridio-v1.1.0.tgz

## Deployment

TODO

7 CRDs:
```sh
$ kubectl api-resources
NAME           SHORTNAMES   APIVERSION                NAMESPACED   KIND
attractors                  meridio.nordix.org/v1     true         Attractor
conduits                    meridio.nordix.org/v1     true         Conduit
flows                       meridio.nordix.org/v1     true         Flow
gateways                    meridio.nordix.org/v1     true         Gateway
streams                     meridio.nordix.org/v1     true         Stream
trenches                    meridio.nordix.org/v1     true         Trench
vips                        meridio.nordix.org/v1     true         Vip

$ kubectl get crds
NAME                            CREATED AT
attractors.meridio.nordix.org   2023-02-27T09:18:12Z
conduits.meridio.nordix.org     2023-02-27T09:18:12Z
flows.meridio.nordix.org        2023-02-27T09:18:12Z
gateways.meridio.nordix.org     2023-02-27T09:18:12Z
streams.meridio.nordix.org      2023-02-27T09:18:12Z
trenches.meridio.nordix.org     2023-02-27T09:18:12Z
vips.meridio.nordix.org         2023-02-27T09:18:12Z
```

2 configmaps:
* meridio-deployment-templates: Contains the templates used to deploy the child resources of the custom resources (stateless-lb-frontend, proxy, nsp...)
* meridio-resource-templates: Contains the templates of the resource limits and requests
```sh
$ kubectl get configmap
NAME                           DATA   AGE
meridio-deployment-templates   8      8s
meridio-resource-templates     7      8s
```

1 deployment with 1 replica:
* meridio-operator: The Meridio Operator itself
```sh
$ kubectl get deployment
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
meridio-operator   1/1     1            1           50s

$ kubectl get pods
NAME                                READY   STATUS    RESTARTS   AGE
meridio-operator-596d7f88b8-bffk5   1/1     Running   0          50s
```

4 Roles:
* meridio-fes-role: Will be used by the frontend container in the stateless-lb-frontend pods to read the secrets for BGP authentication
* meridio-leader-election-role: (optional) Used by the operator to run multiple replicas
* meridio-nsp-role: Will be used by the NSP pods to watch the configmaps for configuration
* meridio-operator-role: Used by the operator to manage resources
```sh
$ kubectl get roles
NAME                           CREATED AT
meridio-fes-role               2023-02-07T13:23:25Z
meridio-leader-election-role   2023-02-07T13:23:25Z
meridio-nsp-role               2023-02-07T13:23:25Z
meridio-operator-role          2023-02-07T13:23:25Z
```

4 Role Bindings:
* meridio-fes-rolebinding: Binds `meridio-fes-role` Role to `meridio-fes` service account
* meridio-leader-election-rolebinding: Binds `meridio-leader-election-role` Role to `meridio-operator` service account
* meridio-nsp-rolebinding: Binds `meridio-nsp-role` Role to `meridio-fes` service account
* meridio-operator-rolebinding: Binds `meridio-operator-role` Role to `meridio-operator` service account
```sh
$ kubectl get rolebindings
NAME                                  ROLE                                AGE
meridio-fes-rolebinding               Role/meridio-fes-role               21s
meridio-leader-election-rolebinding   Role/meridio-leader-election-role   21s
meridio-nsp-rolebinding               Role/meridio-nsp-role               21s
meridio-operator-rolebinding          Role/meridio-operator-role          21s
```

1 Service:
* meridio-operator-webhook-service: Used to serve the `meridio-operator-validating-webhook-configuration-default` validating webhook
```sh
$ kubectl get service
NAME                               TYPE        CLUSTER-IP    EXTERNAL-IP   PORT(S)   AGE
meridio-operator-webhook-service   ClusterIP   10.96.60.15   <none>        443/TCP   25s
```

3 Service account:
* meridio-fes: Will be applied to stateless-lb-frontend pods
* meridio-nsp: Will be applied to nsp pods
* meridio-operator: Applied to the operator pod
```sh
$ kubectl get sa
NAME               SECRETS   AGE
meridio-fes        0         29s
meridio-nsp        0         29s
meridio-operator   0         29s
```

1 validating webhook:
* meridio-operator-validating-webhook-configuration-default: validating webhook for all new custom resources (trench, conduit, stream, flow, vip, attractor and gateway)
```sh
$ kubectl get validatingwebhookconfigurations
NAME                                                        WEBHOOKS   AGE
meridio-operator-validating-webhook-configuration-default   7          33s
```

The picture below represents a Kubernetes cluster with operator applied and highlighted in red:
![Installation-Operator](../resources/Installation-Operator.svg)
