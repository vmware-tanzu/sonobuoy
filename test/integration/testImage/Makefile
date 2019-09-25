GOPATH = $(shell go env GOPATH)
REGISTRY ?= sonobuoy
TARGET = testimage
IMAGE = $(REGISTRY)/$(TARGET)
IMAGE_VERSION=v0.1
DIR := ${CURDIR}
DOCKERFILE := Dockerfile
DOCKER ?= docker

build:
	$(DOCKER) build \
       -t $(REGISTRY)/$(TARGET):$(IMAGE_VERSION) \
       -f $(DOCKERFILE) \
		.