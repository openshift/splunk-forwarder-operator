include boilerplate/generated-includes.mk

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

SHELL := /usr/bin/env bash

FORWARDER_VERSION=$(shell cat .splunk-version)
FORWARDER_HASH=$(shell cat .splunk-version-hash)

FORWARDER_NAME=splunk-forwarder
FORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(FORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)

HEAVYFORWARDER_NAME=splunk-heavyforwarder
HEAVYFORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(HEAVYFORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)

## Convenience targets for local dev. Duplicates are for consistent naming.

.PHONY: build-operator
build-operator: docker-build

.PHONY: push-operator
push-operator: docker-push

.PHONY: image-digests
image-digests:
	./hack/populate-image-digests.sh "$(FORWARDER_IMAGE_URI)" "$(HEAVYFORWARDER_IMAGE_URI)"

.PHONY: vuln-check
vuln-check: build-operator
	./hack/check-image-against-osd-sre-clair.sh $(OPERATOR_IMAGE_URI)
