SHELL := /usr/bin/env bash

OPERATOR_DOCKERFILE = ./build/Dockerfile
FORWARDER_DOCKERFILE = ./containers/forwarder/Dockerfile
HEAVYFORWARDER_DOCKERFILE = ./containers/heavy_forwarder/Dockerfile

# Include shared Makefiles
include project.mk
include standard.mk

default: gobuild

# Extend Makefile after here

.PHONY: docker-build
docker-build: build