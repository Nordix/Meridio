VERSION ?= latest
VERSION_LOAD_BALANCER ?= $(VERSION)

REGISTRY ?= localhost:5000/nvip

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

.PHONY: clear
clear:

.PHONY: default
default: load-balancer
