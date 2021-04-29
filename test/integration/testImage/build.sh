#!/bin/bash

REGISTRY=sonobuoy
TARGET=testimage
IMAGE_VERSION=v0.1

build(){
	docker build \
    	-t $REGISTRY/$TARGET:$IMAGE_VERSION \
    	-f Dockerfile \
		.
}

build
