FIPS_ENABLED=true

include boilerplate/generated-includes.mk

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

SHELL := /usr/bin/env bash

FORWARDER_IMAGE_TAG ?= 10.0.2-e2d18b4767e9-fb655ad

FORWARDER_NAME=splunk-forwarder
FORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(FORWARDER_NAME):$(FORWARDER_IMAGE_TAG)


## Convenience targets for local dev. Duplicates are for consistent naming.

.PHONY: build-operator
build-operator: docker-build

.PHONY: push-operator
push-operator: docker-push

.PHONY: image-digests
image-digests:
	./hack/populate-image-digests.sh "$(FORWARDER_IMAGE_URI)"

.PHONY: vuln-check
vuln-check: build-operator
	./hack/check-image-against-osd-sre-clair.sh $(OPERATOR_IMAGE_URI)

.PHONY: image-update
image-update:
	./hack/update-image-vars.sh $(SFI_UPDATE)
