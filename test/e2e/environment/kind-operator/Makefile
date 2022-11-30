
.PHONY: default
default:
	@for step in $(STEPS); do \
		$(MAKE) -s -C $(shell pwd) $${step} ; \
	done

.PHONY: all
all: default

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

############################################################################
# Variables
############################################################################

STEPS ?= environment operator wait wait-operator wait trenches-targets wait wait-trenches-targets

MERIDIO_VERSION ?= "latest"
TAPA_VERSION ?= "latest"
NSM_VERSION ?= "v1.6.0"
KUBERNETES_VERSION ?= "v1.25"
IP_FAMILY ?= "dualstack"
NAMESPACE ?= "red"

KUBERNETES_WORKERS ?= 2

WAIT_TIME ?= 10
WAIT_TIMEOUT ?= 4m # Kubectl wait timeout

BASE_PATH ?= $(shell pwd)/../../../..

#############################################################################
##@ Environment
#############################################################################

.PHONY: environment
environment: ## Deploy the enviroment (Kind + Gateways + NSM + Spire)
	$(MAKE) -s -C $(BASE_PATH)/docs/demo/scripts/kind/ KUBERNETES_VERSION=$(KUBERNETES_VERSION) NSM_VERSION=$(NSM_VERSION) KUBERNETES_IP_FAMILY=$(IP_FAMILY) KUBERNETES_WORKERS=$(KUBERNETES_WORKERS)

.PHONY: operator
operator: ## Deploy operator
	$(MAKE) -s -C $(BASE_PATH) deploy OPERATOR_NAMESPACE=$(NAMESPACE)

.PHONY: trenches-targets
trenches-targets: ## Deploy trench-a + trench-b + target-a + target-b
	kubectl apply -f $(IP_FAMILY)/configuration/init-trench-a.yaml ; \
	kubectl apply -f $(IP_FAMILY)/configuration/init-trench-b.yaml ; \
	helm install target-a $(BASE_PATH)/examples/target/deployments/helm/ --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a --set default.conduit.name=conduit-a-1 --set default.stream.name=stream-a-i ; \
	helm install target-b $(BASE_PATH)/examples/target/deployments/helm/ --create-namespace --namespace red --set applicationName=target-b --set default.trench.name=trench-b --set default.conduit.name=conduit-b-1 --set default.stream.name=stream-b-i

.PHONY: wait-operator
wait-operator:
	kubectl wait --for=condition=Ready pods --all -n red --timeout=$(WAIT_TIMEOUT) || (OUTPUT_PATH="$(BASE_PATH)/_output" $(BASE_PATH)/hack/log_collector.sh && exit 1)

.PHONY: wait-trenches-targets
wait-trenches-targets:
	kubectl wait --for=condition=Ready pods --all -n red --timeout=$(WAIT_TIMEOUT) || (OUTPUT_PATH="$(BASE_PATH)/_output" $(BASE_PATH)/hack/log_collector.sh && exit 1)

#############################################################################
# Tools
#############################################################################

.PHONY: wait
wait:
	sleep $(WAIT_TIME)
