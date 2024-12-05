FIPS_ENABLED=true

export LATEST_IMAGE_TAG=image-v6.0.1

include boilerplate/generated-includes.mk

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

SHELL := /usr/bin/env bash

FORWARDER_IMAGE_TAG ?= 9.3.2-d8bb32809498-c43b7a7

FORWARDER_NAME=splunk-forwarder
FORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(FORWARDER_NAME):$(FORWARDER_IMAGE_TAG)

HEAVYFORWARDER_NAME=splunk-heavyforwarder
HEAVYFORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(HEAVYFORWARDER_NAME):$(FORWARDER_IMAGE_TAG)

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

.PHONY: image-update
image-update:
	./hack/update-image-vars.sh $(SFI_UPDATE)
