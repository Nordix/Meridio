# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= latest

# https://sdk.operatorframework.io/docs/faqs/#when-invoking-make-targets-why-do-i-see-errors-like-forkexec-usrlocalkubebuilderbinetcd-no-such-file-or-directory-occurred
SHELL := /bin/bash

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# nordix.org/meridio-operator-bundle:$VERSION and nordix.org/meridio-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= nordix.org/meridio-operator
NAMESPACE ?= meridio-operator-system

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(VERSION)

REGISTRY ?= registry.nordix.org/cloud-native/meridio

# Image URL to use all building/pushing image targets
IMG ?= $(REGISTRY)/operator:$(VERSION)
BUILDER ?= golang:1.19.0
BASE_IMG ?= ubuntu:22.04
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= crd

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

BUILD_HASH=$(shell git rev-parse --verify HEAD --short)
BUILD_BRANCH=$(shell git branch --show-current)

GO_BUILD_VARS = \
	github.com/nordix/meridio-operator/controllers/version.Branch=${BUILD_BRANCH} \
	github.com/nordix/meridio-operator/controllers/version.Hash=${BUILD_HASH}

LDFLAGS := $(patsubst %,-X %, $(GO_BUILD_VARS))

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=operator-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

TEMPLATES_HELM_CHART_VALUES_PATH = config/templates/charts/meridio/values.yaml
set-templates-values: ## Set the values in the templates helm chart
	sed -i 's/^version: .*/version: ${VERSION}/' ${TEMPLATES_HELM_CHART_VALUES_PATH} ; \
	sed -i 's/^registry: .*/registry: $(shell echo ${REGISTRY} | cut -d "/" -f 1)/' ${TEMPLATES_HELM_CHART_VALUES_PATH} ; \
	sed -i 's#^organization: .*#organization: $(shell echo ${REGISTRY} | cut -d "/" -f 2-)#' ${TEMPLATES_HELM_CHART_VALUES_PATH}

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.21
test: manifests generate fmt vet envtest ginkgo ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR)
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test -v ./... -short -coverprofile cover.out

E2E_FOCUS?=""
.PHONY: e2e
e2e: ginkgo
	$(GINKGO) -v --focus=$(E2E_FOCUS) ./testdata/e2e/... -- -namespace=${NAMESPACE} -mutating=${ENABLE_MUTATING_WEBHOOK}

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)
##@ Build

build: generate fmt vet ## Build manager binary.
	go build -ldflags="${LDFLAGS}" -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} --build-arg BUILDER=${BUILDER} --build-arg BASE_IMG=${BASE_IMG} --build-arg LDFLAGS="${LDFLAGS}" .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

kind-load: ## Load docker image to kind cluster
	kind load docker-image ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

namespace: ## Edit the namespace of operator to be deployed
	cd config/default && $(KUSTOMIZE) edit set namespace ${NAMESPACE}

ENABLE_MUTATING_WEBHOOK?=true
WEBHOOK_SUPPORT ?= spire # spire or certmanager
configure-webhook:
	ENABLE_MUTATING_WEBHOOK=$(ENABLE_MUTATING_WEBHOOK) WEBHOOK_SUPPORT=$(WEBHOOK_SUPPORT) hack/webhook-switch.sh

deploy: manifests kustomize namespace configure-webhook set-templates-values ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/operator && $(KUSTOMIZE) edit set image operator=${IMG}
	$(KUSTOMIZE) build config/default --enable-helm | kubectl apply -f -

undeploy: namespace configure-webhook ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default --enable-helm | kubectl delete -f - --ignore-not-found=true

apply-samples: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/samples/meridio_v1alpha1_trench.yaml -n ${NAMESPACE}
	kubectl apply -f config/samples/meridio_v1alpha1_attractor.yaml -n ${NAMESPACE}
	kubectl apply -f config/samples/meridio_v1alpha1_vip.yaml -n ${NAMESPACE}
	kubectl apply -f config/samples/meridio_v1alpha1_gateway.yaml -n ${NAMESPACE}
	kubectl apply -f config/samples/meridio_v1alpha1_conduit.yaml -n ${NAMESPACE}
	kubectl apply -f config/samples/meridio_v1alpha1_stream.yaml -n ${NAMESPACE}
	kubectl apply -f config/samples/meridio_v1alpha1_flow.yaml -n ${NAMESPACE}

print-manifests: manifests kustomize namespace  configure-webhook ## Generate manifests to be deployed in the cluster
	cd config/operator && $(KUSTOMIZE) edit set image operator=${IMG}
	$(KUSTOMIZE) build config/default --enable-helm

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

GINKGO = $(shell pwd)/bin/ginkgo
ginkgo: ## Download ginkgo locally if necessary.
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/ginkgo@v1.16.5)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
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

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/operator && $(KUSTOMIZE) edit set image operator=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)