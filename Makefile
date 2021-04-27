include boilerplate/generated-includes.mk

SHELL := /usr/bin/env bash

FORWARDER_VERSION=$(shell cat .splunk-version)
FORWARDER_HASH=$(shell cat .splunk-version-hash)

FORWARDER_NAME=splunk-forwarder
FORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(FORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)
FORWARDER_DOCKERFILE = ./containers/forwarder/Dockerfile

HEAVYFORWARDER_NAME=splunk-heavyforwarder
HEAVYFORWARDER_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(HEAVYFORWARDER_NAME):$(FORWARDER_VERSION)-$(FORWARDER_HASH)
HEAVYFORWARDER_DOCKERFILE = ./containers/heavy_forwarder/Dockerfile

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

# TODO: Temporary override until we figure out how to (conditionally)
# build the splunk-forwarder and splunk-heavyforwarder images via the
# boilerplate standardized appsre pipeline scripts.
.PHONY: build-push
build-push:
	hack/app_sre_build_push.sh $(OPERATOR_IMAGE) $(CURRENT_COMMIT) $(OPERATOR_VERSION) $(FORWARDER_IMAGE_URI) $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: build-push-catalog
build-push-catalog:
	@(if [[ -z "${CHANNEL}" ]]; then echo "Must specify CHANNEL"; exit 1; fi)
	hack/app_sre_create_image_catalog.sh $(CHANNEL) $(OPERATOR_IMAGE) $(CURRENT_COMMIT) $(OPERATOR_VERSION)

## <<< USED BY APP-SRE PIPELINE


.PHONY: build
build: isclean build-operator build-forwarder build-heavyforwarder

.PHONY: docker-push
docker-push:
	$(CONTAINER_ENGINE) push $(OPERATOR_IMAGE_URI)
	$(CONTAINER_ENGINE) push $(FORWARDER_IMAGE_URI)
	$(CONTAINER_ENGINE) push $(HEAVYFORWARDER_IMAGE_URI)

## USED BY CI
# TODO: Temporary until prow jobs are standardized
.PHONY: verify
verify: go-check go-test

.PHONY: vuln-check
vuln-check: build
	./hack/check-image-against-osd-sre-clair.sh $(OPERATOR_IMAGE_URI)
	./hack/check-image-against-osd-sre-clair.sh $(FORWARDER_IMAGE_URI)
	./hack/check-image-against-osd-sre-clair.sh $(HEAVYFORWARDER_IMAGE_URI)

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
