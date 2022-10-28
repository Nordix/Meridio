
.PHONY: default
default:
	$(MAKE) -s $(IMAGES)

.PHONY: all
all: default

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

############################################################################
# Variables
############################################################################

IMAGES ?= base-image stateless-lb proxy tapa ipam nsp example-target frontend operator

# Versions
VERSION ?= latest
VERSION_STATELESS_LB ?= $(VERSION)
VERSION_PROXY ?= $(VERSION)
VERSION_TAPA ?= $(VERSION)
VERSION_IPAM ?= $(VERSION)
VERSION_NSP ?= $(VERSION)
VERSION_EXAMPLE_TARGET ?= $(VERSION)
VERSION_FRONTEND ?= $(VERSION)
VERSION_BASE_IMAGE ?= $(VERSION)
VERSION_OPERATOR ?= $(VERSION)
LOCAL_VERSION ?= $(VERSION)

# E2E tests
E2E_FOCUS ?= ""
E2E_PARAMETERS ?= $(shell cat ./test/e2e/environment/kind-helm/dualstack/config.txt | tr '\n' ' ')
E2E_SEED ?= $(shell shuf -i 1-2147483647 -n1)

# Contrainer Registry
REGISTRY ?= registry.nordix.org/cloud-native/meridio
BASE_IMAGE ?= $(REGISTRY)/base-image:$(VERSION_BASE_IMAGE)
DEBUG_IMAGE ?= $(REGISTRY)/debug:$(VERSION)

# Tools
export PATH := $(shell pwd)/bin:$(PATH)
GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GINKGO = $(shell pwd)/bin/ginkgo
MOCKGEN = $(shell pwd)/bin/mockgen
PROTOC_GEN_GO = $(shell pwd)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC = $(shell pwd)/bin/protoc-gen-go-grpc
NANCY = $(shell pwd)/bin/nancy
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
KUSTOMIZE = $(shell pwd)/bin/kustomize
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

BUILD_DIR ?= build
BUILD_STEPS ?= build tag push

# Security Scan
OUTPUT_DIR ?= _output
SECURITY_SCAN_VOLUME ?= --volume /var/run/docker.sock:/var/run/docker.sock --volume $(HOME)/Library/Caches:/root/.cache/

# Operator
TEMPLATES_HELM_CHART_VALUES_PATH = config/templates/charts/meridio/values.yaml
OPERATOR_NAMESPACE = meridio-operator
ENABLE_MUTATING_WEBHOOK?=true
WEBHOOK_SUPPORT ?= spire # spire or certmanager

#############################################################################
# Container: Build, tag, push
#############################################################################

.PHONY: build
build:
	docker build -t $(IMAGE):$(LOCAL_VERSION) --build-arg meridio_version=$(shell git describe --dirty --tags) --build-arg base_image=$(BASE_IMAGE) -f ./$(BUILD_DIR)/$(IMAGE)/Dockerfile .
.PHONY: tag
tag:
	docker tag $(IMAGE):$(LOCAL_VERSION) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

#############################################################################
##@ Component (Build, tag, push): use VERSION to set the version. Use BUILD_STEPS to set the build steps (build, tag, push)
#############################################################################

.PHONY: base-image
base-image: ## Build the base-image.
	VERSION=$(VERSION_BASE_IMAGE) IMAGE=base-image $(MAKE) -s $(BUILD_STEPS)

.PHONY: debug-image
debug-image: ## Build the debug-image.
	docker build -t $(DEBUG_IMAGE) -f ./build/debug/Dockerfile .

.PHONY: stateless-lb
stateless-lb: ## Build the stateless-lb.
	VERSION=$(VERSION_STATELESS_LB) IMAGE=stateless-lb $(MAKE) -s $(BUILD_STEPS)

.PHONY: proxy
proxy: ## Build the proxy.
	VERSION=$(VERSION_PROXY) IMAGE=proxy $(MAKE) -s $(BUILD_STEPS)

.PHONY: tapa
tapa: ## Build the tapa.
	VERSION=$(VERSION_TAPA) IMAGE=tapa $(MAKE) -s $(BUILD_STEPS)

.PHONY: ipam
ipam: ## Build the ipam.
	VERSION=$(VERSION_IPAM) IMAGE=ipam $(MAKE) -s $(BUILD_STEPS)

.PHONY: nsp
nsp: ## Build the nsp.
	VERSION=$(VERSION_NSP) IMAGE=nsp $(MAKE) -s $(BUILD_STEPS)

.PHONY: example-target
example-target: ## Build the example target.
	VERSION=$(VERSION_EXAMPLE_TARGET) BUILD_DIR=examples/target/build IMAGE=example-target $(MAKE) $(BUILD_STEPS)

.PHONY: frontend
frontend: ## Build the frontend.
	VERSION=$(VERSION_FRONTEND) IMAGE=frontend $(MAKE) -s $(BUILD_STEPS)

.PHONY: operator
operator: ## Build the operator.
	VERSION=$(VERSION_OPERATOR) IMAGE=operator $(MAKE) -s $(BUILD_STEPS)

#############################################################################
##@ Testing & Code check
#############################################################################

.PHONY: lint
lint: golangci-lint ## Run linter against code.
	$(GOLANGCI_LINT) run ./...

.PHONY: e2e
e2e: ginkgo ## Run the E2E tests.
	$(GINKGO) -v --focus=$(E2E_FOCUS) --seed=$(E2E_SEED) --repeat=0 --timeout=1h ./test/e2e/... -- $(E2E_PARAMETERS)

.PHONY: test
test: ## Run the Unit tests.
	go test -race -cover -short ./... 

.PHONY: cover
cover: 
	go test -race -coverprofile cover.out -short ./... 
	go tool cover -html=cover.out -o cover.html

.PHONY: check
check: lint test ## Run the linter and the Unit tests.

#############################################################################
##@ Security Scan
#############################################################################

# https://github.com/anchore/grype
.PHONY: grype
grype: ## Run grype scanner on images.
	@BUILD_STEPS=grype-scan $(MAKE) -s $(IMAGES)

.PHONY: grype-scan
grype-scan: output-dir
	docker run --rm $(SECURITY_SCAN_VOLUME) \
	--name Grype anchore/grype:v0.47.0 \
	$(REGISTRY)/$(IMAGE):$(VERSION) -o json --add-cpes-if-none > $(OUTPUT_DIR)/grype_$(IMAGE)_$(VERSION).json

# https://github.com/aquasecurity/trivy
.PHONY: trivy
trivy: ## Run trivy scanner on images.
	@BUILD_STEPS=trivy-scan $(MAKE) -s $(IMAGES)

.PHONY: trivy-scan
trivy-scan: output-dir
	docker run --rm $(SECURITY_SCAN_VOLUME) \
	aquasec/trivy:0.31.3 image \
	-f json $(REGISTRY)/$(IMAGE):$(VERSION) > $(OUTPUT_DIR)/trivy_$(IMAGE)_$(VERSION).json

# https://github.com/sonatype-nexus-community/nancy
.PHONY: nancy
nancy: nancy-tool output-dir ## Run nancy scanner on dependencies.
	go list -json -deps ./... | nancy sleuth -o json-pretty > $(OUTPUT_DIR)/nancy.json || true
	
#############################################################################
##@ Code generation
#############################################################################

.PHONY: generate
generate: mockgen ## Generate the mocks.
	go generate ./... 

.PHONY: ipam-proto
ipam-proto: proto-compiler
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/ipam/**/*.proto

.PHONY: nsp-proto
nsp-proto: proto-compiler
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/nsp/**/*.proto

.PHONY: ambassador-proto
ambassador-proto: proto-compiler
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/ambassador/**/*.proto

.PHONY: proto
proto: ipam-proto nsp-proto ambassador-proto ## Compile the proto.

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd rbac:roleName=operator-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate-controller
generate-controller: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

#############################################################################
##@ Operator
#############################################################################

.PHONY: deploy
deploy: manifests kustomize namespace configure-webhook set-templates-values ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/operator && $(KUSTOMIZE) edit set image operator=${REGISTRY}/operator:${VERSION_OPERATOR}
	$(KUSTOMIZE) build config/default --enable-helm | kubectl apply -f -

.PHONY: undeploy
undeploy: namespace configure-webhook ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default --enable-helm | kubectl delete -f - --ignore-not-found=true

.PHONY: set-templates-values
set-templates-values: # Set the values in the templates helm chart
	sed -i 's/^version: .*/version: ${VERSION}/' ${TEMPLATES_HELM_CHART_VALUES_PATH} ; \
	sed -i 's/^registry: .*/registry: $(shell echo ${REGISTRY} | cut -d "/" -f 1)/' ${TEMPLATES_HELM_CHART_VALUES_PATH} ; \
	sed -i 's#^organization: .*#organization: $(shell echo ${REGISTRY} | cut -d "/" -f 2-)#' ${TEMPLATES_HELM_CHART_VALUES_PATH}

.PHONY: namespace
namespace: # Edit the namespace of operator to be deployed
	cd config/default && $(KUSTOMIZE) edit set namespace ${OPERATOR_NAMESPACE}

.PHONY: print-manifests
print-manifests: manifests kustomize namespace configure-webhook set-templates-values # Generate manifests to be deployed in the cluster
	cd config/operator && $(KUSTOMIZE) edit set image operator=${REGISTRY}/operator:${VERSION_OPERATOR}
	$(KUSTOMIZE) build config/default --enable-helm

configure-webhook:
	ENABLE_MUTATING_WEBHOOK=$(ENABLE_MUTATING_WEBHOOK) WEBHOOK_SUPPORT=$(WEBHOOK_SUPPORT) hack/webhook-switch.sh
	
#############################################################################
# Tools
#############################################################################

.PHONY: output-dir
output-dir:
	mkdir -p $(OUTPUT_DIR)

.PHONY: golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.47.2)

.PHONY: proto-compiler
proto-compiler: protoc protoc-gen-go protoc-gen-go-grpc

.PHONY: protoc
protoc:
	@if [ ! $(shell which protoc) ]; then\
        echo "Protocol buffer compiler (protoc) must be installed: https://grpc.io/docs/protoc-installation/#install-pre-compiled-binaries-any-os";\
    fi

.PHONY: protoc-gen-go
protoc-gen-go:
	$(call go-get-tool,$(PROTOC_GEN_GO),google.golang.org/protobuf/cmd/protoc-gen-go@v1.28)

.PHONY: protoc-gen-go-grpc
protoc-gen-go-grpc:
	$(call go-get-tool,$(PROTOC_GEN_GO_GRPC),google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2)

.PHONY: mockgen
mockgen:
	$(call go-get-tool,$(MOCKGEN),github.com/golang/mock/mockgen@v1.6.0)

.PHONY: ginkgo
ginkgo:
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.1.4)

.PHONY: nancy-tool
nancy-tool:
	$(call go-get-tool,$(NANCY),github.com/sonatype-nexus-community/nancy@v1.0.37)

.PHONY: controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.10.0)

.PHONY: kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

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