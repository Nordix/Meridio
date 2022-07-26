VERSION ?= latest
VERSION_LOAD_BALANCER ?= $(VERSION)
VERSION_PROXY ?= $(VERSION)
VERSION_TAPA ?= $(VERSION)
VERSION_IPAM ?= $(VERSION)
VERSION_NSP ?= $(VERSION)
VERSION_CTRAFFIC ?= $(VERSION)
VERSION_FRONTEND ?= $(VERSION)

E2E_FOCUS ?= ""
TRAFFIC_GENERATOR_CMD ?= "docker exec -i {trench}"

REGISTRY ?= localhost:5000/meridio
BASE_IMAGE ?= $(REGISTRY)/base:latest
DEBUG_IMAGE ?= $(REGISTRY)/debug:$(VERSION)

.PHONY: all
all: default

.PHONY: build
build:
	docker build -t $(IMAGE) --build-arg meridio_version=$(shell git describe --dirty --tags) --build-arg base_image=$(BASE_IMAGE) -f ./build/$(IMAGE)/Dockerfile .
.PHONY: tag
tag:
	docker tag $(IMAGE) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

.PHONY: base-image
base-image:
	docker build -t $(BASE_IMAGE) -f ./build/base/Dockerfile .

.PHONY: debug-image
debug-image:
	docker build -t $(DEBUG_IMAGE) -f ./build/debug/Dockerfile .

.PHONY: load-balancer
load-balancer:
	VERSION=$(VERSION_LOAD_BALANCER) IMAGE=load-balancer $(MAKE) build tag push

.PHONY: proxy
proxy:
	VERSION=$(VERSION_PROXY) IMAGE=proxy $(MAKE) build tag push

.PHONY: tapa
tapa:
	VERSION=$(VERSION_TAPA) IMAGE=tapa $(MAKE) build tag push

.PHONY: ipam
ipam:
	VERSION=$(VERSION_IPAM) IMAGE=ipam $(MAKE) build tag push

.PHONY: nsp
nsp:
	VERSION=$(VERSION_NSP) IMAGE=nsp $(MAKE) build tag push

.PHONY: ctraffic
ctraffic:
	VERSION=$(VERSION_CTRAFFIC) IMAGE=ctraffic $(MAKE) build tag push

.PHONY: frontend
frontend:
	VERSION=$(VERSION_FRONTEND) IMAGE=frontend $(MAKE) build tag push

.PHONY: ipam-proto
ipam-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/ipam/**/*.proto

.PHONY: nsp-proto
nsp-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/nsp/**/*.proto 

.PHONY: ambassador-proto
ambassador-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/ambassador/**/*.proto

.PHONY: proto
proto: ipam-proto nsp-proto ambassador-proto

.PHONY: clear
clear:

.PHONY: default
default: base-image load-balancer proxy tapa ipam nsp ctraffic frontend

.PHONY: lint
lint: 
	golangci-lint run ./...

NAMESPACE ?= red
.PHONY: e2e
e2e: 
	ginkgo -v --focus=$(E2E_FOCUS) ./test/e2e/... -- -traffic-generator-cmd=$(TRAFFIC_GENERATOR_CMD) -namespace=${NAMESPACE}

.PHONY: test
test: 
	go test -race -cover -short ./... 

.PHONY: cover
cover: 
	go test -race -coverprofile cover.out -short ./... 
	go tool cover -html=cover.out -o cover.html

.PHONY: check
check: lint test

.PHONY: generate
generate: 
	go generate ./... 
