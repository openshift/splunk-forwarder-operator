include boilerplate/generated-includes.mk

SHELL := /usr/bin/env bash

OPERATOR_NAME?=$(shell sed -n 's/.*OperatorName .*"\([^"]*\)".*/\1/p' config/config.go)
OPERATOR_NAMESPACE?=$(shell sed -n 's/.*OperatorNamespace .*"\([^"]*\)".*/\1/p' config/config.go)

IMAGE_REGISTRY?=quay.io
IMAGE_REPOSITORY?=app-sre
IMAGE_NAME?=$(OPERATOR_NAME)
FORWARDER_NAME=splunk-forwarder
HEAVYFORWARDER_NAME=splunk-heavyforwarder

FORWARDER_VERSION=$(shell cat .splunk-version)
FORWARDER_HASH=$(shell cat .splunk-version-hash)

VERSION_MAJOR?=0
VERSION_MINOR?=1

# Generate version and tag information from inputs
COMMIT_NUMBER=$(shell git rev-list `git rev-list --parents HEAD | egrep "^[a-f0-9]{40}$$"`..HEAD --count)
CURRENT_COMMIT=$(shell git rev-parse --short=7 HEAD)
CATALOG_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$(COMMIT_NUMBER)-$(CURRENT_COMMIT)

OPERATOR_IMAGE_BASE=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME)
OPERATOR_IMAGE_URI=$(OPERATOR_IMAGE_BASE):$(CURRENT_COMMIT)
OPERATOR_DOCKERFILE = ./build/Dockerfile

FORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(FORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)
FORWARDER_DOCKERFILE = ./containers/forwarder/Dockerfile

HEAVYFORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(HEAVYFORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)
HEAVYFORWARDER_DOCKERFILE = ./containers/heavy_forwarder/Dockerfile

BINFILE=build/_output/bin/$(OPERATOR_NAME)
MAINPACKAGE=./cmd/manager
GOENV=GOOS=linux GOARCH=amd64 CGO_ENABLED=0
GOFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

TESTTARGETS := $(shell go list -e ./... | egrep -v "/(vendor)/")
# ex, -v
TESTOPTS :=

CONTAINER_ENGINE=$(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)

ALLOW_DIRTY_CHECKOUT?=false

default: gobuild

.PHONY: clean
clean:
	rm -rf ./build/_output

.PHONY: isclean
isclean:
	@(test "$(ALLOW_DIRTY_CHECKOUT)" != "false" || test 0 -eq $$(git status --porcelain | wc -l)) || (echo "Local git checkout is not clean, commit changes and try again." && exit 1)


## >>> USED BY APP-SRE PIPELINE

.PHONY: build-operator
build-operator:
	$(CONTAINER_ENGINE) build . -f $(OPERATOR_DOCKERFILE) -t $(OPERATOR_IMAGE_URI)

.PHONY: build-forwarder
build-forwarder:
	$(CONTAINER_ENGINE) build . -f $(FORWARDER_DOCKERFILE) -t $(FORWARDER_IMAGE_URI)

.PHONY: build-heavyforwarder
build-heavyforwarder:
	$(CONTAINER_ENGINE) build . -f $(HEAVYFORWARDER_DOCKERFILE) -t $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: push-operator
push-operator:
	skopeo copy --dest-creds "$(QUAY_USER):$(QUAY_TOKEN)" "docker-daemon:$(OPERATOR_IMAGE_URI)" "docker://$(OPERATOR_IMAGE_URI)"

.PHONY: push-forwarder
push-forwarder:
	skopeo copy --dest-creds "$(QUAY_USER):$(QUAY_TOKEN)" "docker-daemon:$(FORWARDER_IMAGE_URI)" "docker://$(FORWARDER_IMAGE_URI)"

.PHONY: push-heavyforwarder
push-heavyforwarder:
	skopeo copy --dest-creds "$(QUAY_USER):$(QUAY_TOKEN)" "docker-daemon:$(HEAVYFORWARDER_IMAGE_URI)" "docker://$(HEAVYFORWARDER_IMAGE_URI)"

.PHONY: build-push
build-push:
	hack/app_sre_build_push.sh $(OPERATOR_IMAGE_BASE) $(CURRENT_COMMIT) $(FORWARDER_IMAGE_URI) $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: build-push-catalog
build-push-catalog:
	@(if [[ -z "${CHANNEL}" ]]; then echo "Must specify CHANNEL"; exit 1; fi)
	hack/app_sre_create_image_catalog.sh $(CHANNEL) $(OPERATOR_IMAGE_BASE) $(CURRENT_COMMIT) $(CATALOG_VERSION)

## <<< USED BY APP-SRE PIPELINE


.PHONY: build
build: isclean build-operator build-forwarder build-heavyforwarder

.PHONY: docker-push
docker-push:
	$(CONTAINER_ENGINE) push $(OPERATOR_IMAGE_URI)
	$(CONTAINER_ENGINE) push $(FORWARDER_IMAGE_URI)
	$(CONTAINER_ENGINE) push $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: gocheck
gocheck: ## Lint code
	gofmt -s -l $(shell go list -f '{{ .Dir }}' ./... ) | grep ".*\.go"; if [ "$$?" = "0" ]; then gofmt -s -d $(shell go list -f '{{ .Dir }}' ./... ); exit 1; fi
	go vet ./cmd/... ./pkg/...

## USED BY CI
# TODO: Include gocheck and gotest
.PHONY: verify
verify: ## Lint code
	golangci-lint run

.PHONY: vuln-check
vuln-check: build
	./hack/check-image-against-osd-sre-clair.sh $(OPERATOR_IMAGE_URI)
	./hack/check-image-against-osd-sre-clair.sh $(FORWARDER_IMAGE_URI)
	./hack/check-image-against-osd-sre-clair.sh $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: gobuild
gobuild: gocheck gotest ## Build binary
	${GOENV} go build ${GOFLAGS} -o ${BINFILE} ${MAINPACKAGE}

.PHONY: gotest
gotest:
	go test $(TESTOPTS) $(TESTTARGETS)

.PHONY: test
test: gotest

default: gobuild

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
