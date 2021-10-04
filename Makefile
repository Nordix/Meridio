VERSION ?= latest
VERSION_LOAD_BALANCER ?= $(VERSION)
VERSION_PROXY ?= $(VERSION)
VERSION_TARGET ?= $(VERSION)
VERSION_IPAM ?= $(VERSION)
VERSION_NSP ?= $(VERSION)
VERSION_CTRAFFIC ?= $(VERSION)
VERSION_FRONTEND ?= $(VERSION)

E2E_FOCUS ?= ""
TRAFFIC_GENERATOR_CMD ?= "docker exec -i {trench}"

REGISTRY ?= localhost:5000/meridio

.PHONY: all
all: default

.PHONY: build
build:
	docker build -t $(IMAGE) -f ./build/$(IMAGE)/Dockerfile .
.PHONY: tag
tag:
	docker tag $(IMAGE) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

.PHONY: load-balancer
load-balancer:
	VERSION=$(VERSION_LOAD_BALANCER) IMAGE=load-balancer $(MAKE) build tag push

.PHONY: proxy
proxy:
	VERSION=$(VERSION_PROXY) IMAGE=proxy $(MAKE) build tag push

.PHONY: target
target:
	VERSION=$(VERSION_TARGET) IMAGE=target $(MAKE) build tag push

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
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/ipam/ipam.proto

.PHONY: nsp-proto
nsp-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/nsp/**/*.proto 

.PHONY: target-proto
target-proto:
	protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative api/target/target.proto

.PHONY: proto
proto: ipam-proto nsp-proto target-proto

.PHONY: clear
clear:

.PHONY: default
default: load-balancer proxy target ipam nsp ctraffic frontend

.PHONY: lint
lint: 
	golangci-lint run ./...

.PHONY: e2e
e2e: 
	ginkgo --failFast --focus=$(E2E_FOCUS) ./test/e2e/... -- -traffic-generator-cmd=$(TRAFFIC_GENERATOR_CMD)

.PHONY: test
test: 
	go test -race -cover -short ./... 

.PHONY: check
check: lint test
