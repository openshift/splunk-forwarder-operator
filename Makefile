include boilerplate/generated-includes.mk

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

SHELL := /usr/bin/env bash

FORWARDER_VERSION=$(shell cat .splunk-version)
FORWARDER_HASH=$(shell cat .splunk-version-hash)

FORWARDER_NAME=splunk-forwarder
FORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(FORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)
FORWARDER_DOCKERFILE = ./containers/forwarder/Dockerfile

HEAVYFORWARDER_NAME=splunk-heavyforwarder
HEAVYFORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(HEAVYFORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)
HEAVYFORWARDER_DOCKERFILE = ./containers/heavy_forwarder/Dockerfile

define ADDITIONAL_IMAGE_SPECS
# The splunk-forwarder image
$(FORWARDER_DOCKERFILE) $(FORWARDER_IMAGE_URI)
# The splunk-heavyforwarder image
$(HEAVYFORWARDER_DOCKERFILE) $(HEAVYFORWARDER_IMAGE_URI)
endef

## Convenience targets for local dev. Duplicates are for consistent naming.

.PHONY: build-operator
build-operator: docker-build

.PHONY: build-forwarder
build-forwarder:
	$(CONTAINER_ENGINE) build . -f $(FORWARDER_DOCKERFILE) -t $(FORWARDER_IMAGE_URI)

.PHONY: build-heavyforwarder
build-heavyforwarder:
	$(CONTAINER_ENGINE) build . -f $(HEAVYFORWARDER_DOCKERFILE) -t $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: push-operator
push-operator: docker-push

.PHONY: push-forwarder
push-forwarder:
	skopeo copy --dest-creds "$(QUAY_USER):$(QUAY_TOKEN)" "docker-daemon:$(FORWARDER_IMAGE_URI)" "docker://$(FORWARDER_IMAGE_URI)"

# Use caution: this is huge
.PHONY: push-heavyforwarder
push-heavyforwarder:
	skopeo copy --dest-creds "$(QUAY_USER):$(QUAY_TOKEN)" "docker-daemon:$(HEAVYFORWARDER_IMAGE_URI)" "docker://$(HEAVYFORWARDER_IMAGE_URI)"

.PHONY: build-all
build-all: isclean build-operator build-forwarder build-heavyforwarder

.PHONY: push-all
push-all: push-operator push-forwarder push-heavyforwarder

## USED BY CI
# TODO: Temporary until prow jobs are standardized
.PHONY: verify
verify: go-check go-test

.PHONY: vuln-check
vuln-check: build-all
	./hack/check-image-against-osd-sre-clair.sh $(OPERATOR_IMAGE_URI)
	./hack/check-image-against-osd-sre-clair.sh $(FORWARDER_IMAGE_URI)
	./hack/check-image-against-osd-sre-clair.sh $(HEAVYFORWARDER_IMAGE_URI)
