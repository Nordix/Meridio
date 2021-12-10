# E2E Test Instruction

## Tools

- ginkgo - https://onsi.github.io/ginkgo/#getting-ginkgo

## Steps

1. Deploy Spire

   ```bash
   helm install  --generate-name docs/demo/deployments/spire/
   ```

2. Configure Spire

   ```bash
   ./docs/demo/scripts/spire-config.sh
   ```

3. Deploy NSM

   ```bash
   helm install docs/demo/deployments/nsm-vlan/ --generate-name
   ```

4. Configure Spire for the testing namespace(meridio to be exist), and trench

   ```bash
   ns=<testing-namespace>
   ./docs/demo/scripts/spire.sh default $ns
   # this step(step 4) needs to be done for multiple times depending on how many trenches are deployed
   # the <meridio-service-account-name>, for example, for trench-a, is "meridio-trench-a" if helm chart is used, "meridio-sa-trench-a" if operator is used
   ./docs/demo/scripts/spire.sh <meridio-service-account-name> $ns
   ```

5. Deploy common resources for the targets

   ```bash
   helm install examples/target/common/ --generate-name --create-namespace --namespace $ns
   ```

6. Configure Spire for the targets

   ```bash
   ./docs/demo/scripts/spire.sh meridio $ns
   ```

7. Install targets connected to trench-a

   ```bash
   # <conduit-name> is load-balancer if testing with helm, and lb-fe if testing with the operator
   helm install examples/target/helm/ --generate-name --create-namespace --namespace $ns --set applicationName=target-a --set default.trench.name=trench-a --set default.conduit.name=<conduit-name>
   ```

8. Deploy External host / External connectivity

   ```bash
   ./docs/demo/scripts/kind/external-host.sh
   ```

The following steps depends whether the tests are running with or without the meridio operator.

### Testing with Meridio Operator

Meridio should test with the operator before each release to ensure the compatibility.

9. Install the operator

- Install certificate manager

    ```bash
    kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
    ```

- Build operator image and make it available for cluster

    ```bash
    # update the Meridio Operator submodule
    git pull
    git submodule sync --recursive
    git submodule update --init --recursive
    # run at the git root directory
    cd Meridio-Operator
    make generate manifests docker-build kind-load
    ```

- Deploy operator and all the instances for e2e test

    ```bash
    make deploy NAMESPACE=$ns
    # go back to git root directory
    cd ..
    kubectl apply -f ./test/e2e/manifest/meridio-trench-a.yaml
    kubectl apply -f ./test/e2e/manifest/meridio-trench-b.yaml
    ```

10. To run the e2e tests with Meridio Operator

    ```bash
    # at meridio git base directory
    make e2e NAMESPACE=$ns TEST_WITH_OPERATOR=true
    ```

### Testing without Meridio Operator

9. Install Merido by helm chart

    ```bash
    # trench-a
    helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-a --set ipFamily=dualstack --set vlan.fe.gateway[0]="169.254.100.150/24" --set vlan.fe.gateway[1]="100:100::150/64"
    # trench-b
    helm install deployments/helm/ --generate-name --create-namespace --namespace red --set trench.name=trench-b --set vlan.id=101 --set ipFamily=dualstack --set vlan.fe.gateway[0]="169.254.100.150/24" --set vlan.fe.gateway[1]="100:100::150/64"
    ```

10. Run e2e tests

    ```bash
    make e2e
    ```
