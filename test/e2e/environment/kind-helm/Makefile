
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

STEPS ?= environment meridio wait wait-meridio

MERIDIO_VERSION ?= "latest"
TAPA_VERSION ?= "latest"
NSM_VERSION ?= "v1.6.0"
KUBERNETES_VERSION ?= "v1.25"
IP_FAMILY ?= "dualstack"

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

.PHONY: meridio
meridio: ## Deploy Meridio (trench-a + trench-b + target-a + target-b)
	helm install trench-a $(BASE_PATH)/deployments/helm/ --create-namespace --namespace red --set trench.name=trench-a --set ipFamily=$(IP_FAMILY) ; \
	helm install trench-b $(BASE_PATH)/deployments/helm/ --create-namespace --namespace red --set trench.name=trench-b --set vlan.id=200 --set ipFamily=$(IP_FAMILY) ; \
	helm install target-a $(BASE_PATH)/examples/target/deployments/helm/ --create-namespace --namespace red --set applicationName=target-a --set default.trench.name=trench-a ; \
	helm install target-b $(BASE_PATH)/examples/target/deployments/helm/ --create-namespace --namespace red --set applicationName=target-b --set default.trench.name=trench-b

.PHONY: wait-meridio
wait-meridio:
	kubectl wait --for=condition=Ready pods --all -n red --timeout=$(WAIT_TIMEOUT) || (OUTPUT_PATH="$(BASE_PATH)/_output" $(BASE_PATH)/hack/log_collector.sh && exit 1)

#############################################################################
# Tools
#############################################################################

.PHONY: wait
wait:
	sleep $(WAIT_TIME)
