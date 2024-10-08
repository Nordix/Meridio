
.PHONY: default
default:
	@set -e; \
	for step in $(STEPS); do \
		if ! $(MAKE) -s -C $(shell pwd) $${step} ; then \
			echo "$${step} failed" >&2; \
			exit 1; \
		fi; \
	done

.PHONY: all
all: default

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

############################################################################
# Variables
############################################################################

STEPS ?= clean kind-create kind-network multus cni-plugins whereabouts kind-gateways install-metrics-server install-spire wait wait-spire install-nsm wait wait-nsm

KUBERNETES_VERSION ?= "v1.25"
KUBERNETES_IP_FAMILY ?= "dualstack"
KUBERNETES_WORKERS ?= 2

NSM_VERSION ?= "v1.6.0"

# Tools
export PATH := $(shell pwd)/bin:$(PATH)
KIND = $(shell pwd)/bin/kind
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

TEMP_DIR ?= temp

WAIT_TIME ?= 10
WAIT_TIMEOUT ?= 4m # Kubectl wait timeout

BUILD_STEPS ?= build tag push
REGISTRY ?= registry.nordix.org/cloud-native/meridio
VERSION ?= latest
KIND_EXTERNAL_HOST_VERSION ?= $(VERSION)
KIND_EXTERNAL_HOST_DEFAULT_ROUTE ?= yes
BREAKOUT_NETWORK ?= kind-breakout

OBSERVABILITY_ENABLED ?= false

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
version="kindest/node:v1.31.0@sha256:53df588e04085fd41ae12de0c3fe4c72f7013bba32a20e7325357a1ac94ba865" ; \
if [ $(KUBERNETES_VERSION) = "v1.30" ]; then \
    version="kindest/node:v1.30.4@sha256:976ea815844d5fa93be213437e3ff5754cd599b040946b5cca43ca45c2047114" ; \
elif [ $(KUBERNETES_VERSION) = "v1.29" ]; then \
    version="kindest/node:v1.29.8@sha256:d46b7aa29567e93b27f7531d258c372e829d7224b25e3fc6ffdefed12476d3aa" ; \
elif [ $(KUBERNETES_VERSION) = "v1.28" ]; then \
    version="kindest/node:v1.28.13@sha256:45d319897776e11167e4698f6b14938eb4d52eb381d9e3d7a9086c16c69a8110" ; \
elif [ $(KUBERNETES_VERSION) = "v1.27" ]; then \
    version="kindest/node:v1.27.3@sha256:3966ac761ae0136263ffdb6cfd4db23ef8a83cba8a463690e98317add2c9ba72" ; \
elif [ $(KUBERNETES_VERSION) = "v1.26" ]; then \
    version="kindest/node:v1.26.6@sha256:6e2d8b28a5b601defe327b98bd1c2d1930b49e5d8c512e1895099e4504007adb" ; \
elif [ $(KUBERNETES_VERSION) = "v1.25" ]; then \
    version="kindest/node:v1.25.11@sha256:227fa11ce74ea76a0474eeefb84cb75d8dad1b08638371ecf0e86259b35be0c8" ; \
elif [ $(KUBERNETES_VERSION) = "v1.24" ]; then \
    version="kindest/node:v1.24.15@sha256:7db4f8bea3e14b82d12e044e25e34bd53754b7f2b0e9d56df21774e6f66a70ab" ; \
elif [ $(KUBERNETES_VERSION) = "v1.23" ]; then \
    version="kindest/node:v1.23.17@sha256:e5fd1d9cd7a9a50939f9c005684df5a6d145e8d695e78463637b79464292e66c" ; \
elif [ $(KUBERNETES_VERSION) = "v1.22" ]; then \
    version="kindest/node:v1.22.17@sha256:c8a828709a53c25cbdc0790c8afe12f25538617c7be879083248981945c38693" ; \
elif [ $(KUBERNETES_VERSION) = "v1.21" ]; then \
    version="kindest/node:v1.21.14@sha256:27ef72ea623ee879a25fe6f9982690a3e370c68286f4356bf643467c552a3888" ; \
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

.PHONY: whereabouts
whereabouts: ## Install whereabouts IPAM CNI plugin
	kubectl apply \
	-f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/daemonset-install.yaml \
	-f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_ippools.yaml \
	-f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml && \
	kubectl rollout status daemonset whereabouts -n kube-system --timeout 300s || \
	{ echo >&2 "Error: Failed to install Whereabouts"; exit 1; }

.PHONY: multus
multus: ## Install multus
	kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml && \
	kubectl rollout status daemonset kube-multus-ds -n kube-system --timeout 300s || { echo >&2 "Error: Failed to install Multus"; exit 1; }

.PHONY: cni-plugins
cni-plugins: ## Install CNI plugins (add recovery option by failing cni install container if wget fails)
	curl -sL https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/e2e/templates/cni-install.yml.j2 | \
	sed '/^\s*wget / s/$$/ || { echo "wget failed"; exit 1; }/' | \
	kubectl apply -f - && \
	kubectl rollout status daemonset install-cni-plugins -n kube-system --timeout 300s || \
	{ echo >&2 "Error: Failed to install CNI plugins"; exit 1; }

.PHONY: kind-network
kind-network: kind-network-delete ## Setup the Kind network
	./bridge-network-setup.sh --network-name=$(BREAKOUT_NETWORK) --kubernetes-workers=$(KUBERNETES_WORKERS) --ipfamily=$(KUBERNETES_IP_FAMILY) || \
	{ echo >&2 "Error: Failed to setup network"; exit 1; }

.PHONY: kind-network-delete
kind-network-delete:  # Clean up the Kind network
	docker network rm $(BREAKOUT_NETWORK) || true ;

.PHONY: kind-gateways-network-disconnect
kind-gateways-network-disconnect:
	docker network disconnect -f $(BREAKOUT_NETWORK) trench-a ; \
	docker network disconnect -f $(BREAKOUT_NETWORK) trench-b ; \
	docker network disconnect -f $(BREAKOUT_NETWORK) trench-c ;

.PHONY: kind-gateways
kind-gateways: ## Create the Kind cluster gateways
	./external-host.sh --default-route=$(KIND_EXTERNAL_HOST_DEFAULT_ROUTE) --network-name=$(BREAKOUT_NETWORK)

.PHONY: kind-gateways
kind-delete-gateways: ## Delete the Kind gateways
	docker kill trench-a || true ; \
	docker kill trench-b || true ; \
	docker kill trench-c || true ; \
	docker rm trench-a || true ; \
	docker rm trench-b || true ; \
	docker rm trench-c || true

.PHONY: kind-delete
kind-delete-cluster: kind ## Delete the Kind cluster
	$(KIND) delete cluster

.PHONY: clean
clean: kind-delete-cluster kind-delete-gateways kind-network-delete ## Delete the Kind cluster and the Kind gateways

#############################################################################
##@ Spire
#############################################################################

.PHONY: install-spire
install-spire: ## Install spire
	kubectl create namespace spire ; \
	kubectl apply -f ../../deployments/spire/spire-crds.yaml -n spire ; \
	kubectl apply -f ../../deployments/spire/spire.yaml -n spire

.PHONY: wait-spire
wait-spire: ## Wait for spire to be ready
	kubectl wait -n spire --timeout=$(WAIT_TIMEOUT) --for=condition=ready pod -l app.kubernetes.io/instance=my-spire

#############################################################################
##@ NSM
#############################################################################

.PHONY: install-nsm
install-nsm: ## Install nsm
	helm install nsm ../../deployments/nsm --create-namespace --namespace nsm --set tag=$(NSM_VERSION)

.PHONY: wait-nsm
wait-nsm: ## Wait for NSM to be ready
	kubectl wait -n nsm --timeout=$(WAIT_TIMEOUT) --for=condition=ready pod -l app=nsmgr ; \
	kubectl wait -n nsm --timeout=$(WAIT_TIMEOUT) --for=condition=ready pod -l app=forwarder-vpp ; \
	kubectl wait -n nsm --timeout=$(WAIT_TIMEOUT) --for=condition=ready pod -l app=nsm-registry

#############################################################################
##@ Observability
#############################################################################

.PHONY: install-metrics-server
install-metrics-server: ## Install metrics-server
	kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/download/v0.6.4/components.yaml ; \
	kubectl patch deployment metrics-server -n kube-system -p '{"spec":{"template":{"spec":{"containers":[{"name":"metrics-server","args":["--cert-dir=/tmp", "--secure-port=4443", "--kubelet-insecure-tls", "--metric-resolution=15s", "--kubelet-preferred-address-types=InternalIP"]}]}}}}'

.PHONY: install-optl-prometheus-grafana
install-optl-prometheus-grafana: ## Install Open-Telemetry + Prometheus + Grafana
	if [ $(OBSERVABILITY_ENABLED) = "true" ]; then \
		./../../deployments/optl-prometheus-grafana/deploy.sh ; \
	fi

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
	$(call go-get-tool,$(KIND),sigs.k8s.io/kind@v0.20.0)

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
