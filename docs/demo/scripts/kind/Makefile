
.PHONY: default
default:
	$(MAKE) -s -C $(shell pwd) $(STEPS)

.PHONY: all
all: default

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

############################################################################
# Variables
############################################################################

STEPS ?= clean kind-create kind-gateways install-spire wait wait-spire install-nsm wait wait-nsm

KUBERNETES_VERSION ?= "v1.25"
KUBERNETES_IP_FAMILY ?= "dualstack"
KUBERNETES_WORKERS ?= 2

NSM_VERSION ?= "v1.6.0"

# Tools
export PATH := $(shell pwd)/bin:$(PATH)
KIND = $(shell pwd)/bin/kind
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

TEMP_DIR ?= temp

WAIT_TIME ?= 2

BUILD_STEPS ?= build tag push
REGISTRY ?= registry.nordix.org/cloud-native/meridio
VERSION ?= latest
KIND_EXTERNAL_HOST_VERSION ?= $(VERSION)

#############################################################################
# Kind Config
#############################################################################

define KIND_CONFIG_NO_NODE
---
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  kubeProxyMode: ipvs
  ipFamily: $(shell echo $(KUBERNETES_IP_FAMILY) | sed -e "s/dualstack/dual/g")
nodes:
endef
export KIND_CONFIG_NO_NODE

define KIND_CONFIG
version="kindest/node:v1.25.2@sha256:9be91e9e9cdf116809841fc77ebdb8845443c4c72fe5218f3ae9eb57fdb4bace" ; \
if [ $(KUBERNETES_VERSION) = "v1.24" ]; then \
    version="kindest/node:v1.24.6@sha256:97e8d00bc37a7598a0b32d1fabd155a96355c49fa0d4d4790aab0f161bf31be1" ; \
elif [ $(KUBERNETES_VERSION) = "v1.23" ]; then \
    version="kindest/node:v1.23.12@sha256:9402cf1330bbd3a0d097d2033fa489b2abe40d479cc5ef47d0b6a6960613148a" ; \
elif [ $(KUBERNETES_VERSION) = "v1.22" ]; then \
    version="kindest/node:v1.22.15@sha256:bfd5eaae36849bfb3c1e3b9442f3da17d730718248939d9d547e86bbac5da586" ; \
elif [ $(KUBERNETES_VERSION) = "v1.21" ]; then \
    version="kindest/node:v1.21.14@sha256:ad5b7446dd8332439f22a1efdac73670f0da158c00f0a70b45716e7ef3fae20b" ; \
elif [ $(KUBERNETES_VERSION) = "v1.20" ]; then \
    version="kindest/node:v1.20.15@sha256:45d0194a8069c46483a0e509088ab9249302af561ebee76a1281a1f08ecb4ed3" ; \
elif [ $(KUBERNETES_VERSION) = "v1.19" ]; then \
    version="kindest/node:v1.19.16@sha256:a146f9819fece706b337d34125bbd5cb8ae4d25558427bf2fa3ee8ad231236f2" ; \
fi ; \
echo "$$KIND_CONFIG_NO_NODE" ; \
echo "- role: control-plane" ; \
echo "  image: $$version" ; \
for number in `seq 1 $(KUBERNETES_WORKERS)` ; do \
	echo "- role: worker" ; \
	echo "  image: $$version" ; \
done
endef

#############################################################################
# Container: Build, tag, push
#############################################################################

.PHONY: build
build:
	docker build -t $(IMAGE):$(KIND_EXTERNAL_HOST_VERSION) --build-arg BIRD_CONFIG_PATH=bird -f ./Dockerfile .
.PHONY: tag
tag:
	docker tag $(IMAGE):$(KIND_EXTERNAL_HOST_VERSION) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

#############################################################################
##@ Component (Build, tag, push): use VERSION to set the version. Use BUILD_STEPS to set the build steps (build, tag, push)
#############################################################################

.PHONY: kind-host
kind-host: ## Build the external kind host (used as a gateway / traffic generator)
	VERSION=$(KIND_EXTERNAL_HOST_VERSION) IMAGE=kind-host $(MAKE) -s $(BUILD_STEPS)

#############################################################################
##@ Kind
#############################################################################

.PHONY: kind-config
kind-config: ## Print the Kind cluster config
	$(call KIND_CONFIG)

.PHONY: kind-create
kind-create: temp-dir kind kind-delete ## Create the Kind cluster
	$(MAKE) -s kind-config > $(TEMP_DIR)/kind-config.yaml ; \
	$(KIND) create cluster --config $(TEMP_DIR)/kind-config.yaml ; \
	rm $(TEMP_DIR)/kind-config.yaml

.PHONY: kind-gateways
kind-gateways: ## Create the Kind cluster gateways
	./external-host.sh

.PHONY: kind-gateways
kind-delete-gateways: ## Delete the Kind gateways
	docker kill trench-a ; \
	docker kill trench-b ; \
	docker kill trench-c ; \
	docker rm trench-a ; \
	docker rm trench-b ; \
	docker rm trench-c

.PHONY: kind-delete
kind-delete-cluster: kind ## Delete the Kind cluster
	$(KIND) delete cluster

.PHONY: clean
clean: kind-delete-cluster kind-delete-gateways ## Delete the Kind cluster and the Kind gateways

#############################################################################
##@ Spire
#############################################################################

.PHONY: install-spire
install-spire: ## Install spire
	kubectl apply -k ../../deployments/spire

.PHONY: wait-spire
wait-spire: ## Wait for spire to be ready
	kubectl wait -n spire --timeout=2m --for=condition=ready pod -l app=spire-agent ; \
	kubectl wait -n spire --timeout=2m --for=condition=ready pod -l app=spire-server

#############################################################################
##@ NSM
#############################################################################

.PHONY: install-nsm
install-nsm: ## Install nsm
	helm install nsm ../../deployments/nsm --create-namespace --namespace nsm

.PHONY: wait-nsm
wait-nsm: ## Wait for NSM to be ready
	kubectl wait -n nsm --timeout=2m --for=condition=ready pod -l app=nsmgr ; \
	kubectl wait -n nsm --timeout=2m --for=condition=ready pod -l app=forwarder-vpp ; \
	kubectl wait -n nsm --timeout=2m --for=condition=ready pod -l app=nsm-registry

#############################################################################
# Tools
#############################################################################

.PHONY: wait
wait:
	sleep $(WAIT_TIME)

.PHONY: temp-dir
temp-dir:
	mkdir -p $(TEMP_DIR)

.PHONY: kind
kind:
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind@v0.16.0)

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
