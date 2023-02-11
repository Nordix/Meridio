# Summary

This is a simple playground environment with Meridio

The prerequisites take about 10 minutes to install. 

KinD with Kubernetes v1.26, Gateways/Traffic generator made for KinD, Spire, NSM v1.7.1 and Meridio v1.0.0 are being deployed.

# Examples

Deploy trench-a used in the Meridio e2e tests:
```
kubectl apply -f test/e2e/environment/kind-operator/dualstack/configuration/init-trench-a.yaml
```{{copy}}

Deploy targets attacth to the trench/conduit/stream:
```
helm install target-a examples/target/deployments/helm/ --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a --set default.conduit.name=conduit-a-1 --set default.stream.name=stream-a-i
```{{copy}}

Send traffic
```
docker exec -it trench-a mconnect -address 20.0.0.1:4000 -nconn 400 -timeout 2s
```{{copy}}
```
docker exec -it trench-a mconnect -address  [2000::1]:4000 -nconn 400 -timeout 2s
```{{copy}}

# Feedback

Do you see any bug, typo in the tutorial or you have some feedback?
Let us know on https://github.com/Nordix/Meridio or [Slack](https://cloud-native.slack.com/archives/C03ETG3J04S).

If you like Meridio, give it a star on [Github](https://github.com/Nordix/Meridio) ⭐️.

For more information about the project, please do check [Github.com/Nordix/Meridio](https://github.com/Nordix/Meridio) or [meridio.nordix.org](https://meridio.nordix.org/)

<img src="https://raw.githubusercontent.com/Nordix/Meridio/master/docs/resources/Logo.svg" width="118" height="100">