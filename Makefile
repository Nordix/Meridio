
.PHONY: default
default: base-image load-balancer proxy tapa ipam nsp ctraffic frontend

############################################################################
# Variables
############################################################################

# Versions
VERSION ?= latest
VERSION_LOAD_BALANCER ?= $(VERSION)
VERSION_PROXY ?= $(VERSION)
VERSION_TAPA ?= $(VERSION)
VERSION_IPAM ?= $(VERSION)
VERSION_NSP ?= $(VERSION)
VERSION_CTRAFFIC ?= $(VERSION)
VERSION_FRONTEND ?= $(VERSION)

# E2E tests
E2E_FOCUS ?= ""
TRAFFIC_GENERATOR_CMD ?= "docker exec -i {trench}"
NAMESPACE ?= red

# Contrainer Registry
REGISTRY ?= localhost:5000/meridio
BASE_IMAGE ?= $(REGISTRY)/base:latest
DEBUG_IMAGE ?= $(REGISTRY)/debug:$(VERSION)

# Tools
export PATH := $(shell pwd)/bin:$(PATH)
GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GINKGO = $(shell pwd)/bin/ginkgo
MOCKGEN = $(shell pwd)/bin/mockgen
PROTOC_GEN_GO = $(shell pwd)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC = $(shell pwd)/bin/protoc-gen-go-grpc
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

BUILD_DIR ?= build
BUILD_STEPS ?= build tag push

#############################################################################
# Container: Build, tag, push
#############################################################################

.PHONY: all
all: default

.PHONY: build
build:
	docker build -t $(IMAGE) --build-arg meridio_version=$(shell git describe --dirty --tags) --build-arg base_image=$(BASE_IMAGE) -f ./$(BUILD_DIR)/$(IMAGE)/Dockerfile .
.PHONY: tag
tag:
	docker tag $(IMAGE) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

#############################################################################
# Component (Build, tag, push): base, debug, load-balancer, proxy, tapa, ipam, nsp, ctraffic, frontend
#############################################################################

.PHONY: base-image
base-image:
	docker build -t $(BASE_IMAGE) -f ./build/base/Dockerfile .

.PHONY: debug-image
debug-image:
	docker build -t $(DEBUG_IMAGE) -f ./build/debug/Dockerfile .

.PHONY: load-balancer
load-balancer:
	VERSION=$(VERSION_LOAD_BALANCER) IMAGE=load-balancer $(MAKE) $(BUILD_STEPS)

.PHONY: proxy
proxy:
	VERSION=$(VERSION_PROXY) IMAGE=proxy $(MAKE) $(BUILD_STEPS)

.PHONY: tapa
tapa:
	VERSION=$(VERSION_TAPA) IMAGE=tapa $(MAKE) $(BUILD_STEPS)

.PHONY: ipam
ipam:
	VERSION=$(VERSION_IPAM) IMAGE=ipam $(MAKE) $(BUILD_STEPS)

.PHONY: nsp
nsp:
	VERSION=$(VERSION_NSP) IMAGE=nsp $(MAKE) $(BUILD_STEPS)

.PHONY: ctraffic
ctraffic:
	VERSION=$(VERSION_CTRAFFIC) IMAGE=ctraffic $(MAKE) $(BUILD_STEPS)

.PHONY: frontend
frontend:
	VERSION=$(VERSION_FRONTEND) IMAGE=frontend $(MAKE) $(BUILD_STEPS)

#############################################################################
# Testing & Code check
#############################################################################

.PHONY: lint
lint: golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: e2e
e2e: ginkgo
	$(GINKGO) -v --focus=$(E2E_FOCUS) ./test/e2e/... -- -traffic-generator-cmd=$(TRAFFIC_GENERATOR_CMD) -namespace=${NAMESPACE}

.PHONY: test
test: 
	go test -race -cover -short ./... 

.PHONY: cover
cover: 
	go test -race -coverprofile cover.out -short ./... 
	go tool cover -html=cover.out -o cover.html

.PHONY: check
check: lint test

#############################################################################
# Code generation
#############################################################################

.PHONY: generate
generate: mockgen
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
proto: ipam-proto nsp-proto ambassador-proto

#############################################################################
# Tools
#############################################################################

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
