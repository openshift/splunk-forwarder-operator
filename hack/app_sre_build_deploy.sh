#!/bin/bash

# AppSRE team CD

set -exv

CURRENT_DIR=$(dirname "$0")

BASE_IMG="splunk-forwarder-operator"
FORWARDER_BASE_IMG="splunk-forwarder"
HEAVYFORWARDER_BASE_IMG="splunk-heavyforwarder"
QUAY_IMAGE="quay.io/app-sre/${BASE_IMG}"
QUAY_FORWARDER_IMAGE="quay.io/app-sre/${FORWARDER_BASE_IMG}"
QUAY_HEAVYFORWARDER_IMAGE="quay.io/app-sre/${HEAVYFORWARDER_BASE_IMG}"
IMG="${BASE_IMG}:latest"
FORWARDER_IMG="${FORWARDER_BASE_IMG}:latest"
HEAVYFORWARDER_IMG="${HEAVYFORWARDER_BASE_IMG}:latest"

GIT_HASH=$(git rev-parse --short=7 HEAD)

# build the image
BUILD_CMD="docker build" IMG="$IMG" FORWARDER_IMG="$FORWARDER_IMG" HEAVYFORWARDER_IMG="$HEAVYFORWARDER_IMG" make docker-build

# push the image
skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${IMG}" \
    "docker://${QUAY_IMAGE}:latest"

skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${IMG}" \
    "docker://${QUAY_IMAGE}:${GIT_HASH}"

skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${FORWARDER_IMG}" \
    "docker://${QUAY_FORWARDER_IMAGE}:latest"

skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${HEAVYFORWARDER_IMG}" \
    "docker://${QUAY_HEAVYFORWARDER_IMAGE}:latest"

FORWARDER_VERSION=$(cat .splunk-version)
FORWARDER_HASH=$(cat .splunk-version-hash)

skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${FORWARDER_IMG}" \
    "docker://${QUAY_FORWARDER_IMAGE}:${FORWARDER_VERSION}-${FORWARDER_HASH}"

skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${HEAVYFORWARDER_IMG}" \
    "docker://${QUAY_HEAVYFORWARDER_IMAGE}:${FORWARDER_VERSION}-${FORWARDER_HASH}"

# create and push staging image catalog
"$CURRENT_DIR"/app_sre_create_image_catalog.sh staging "$QUAY_IMAGE"

# create and push production image catalog
REMOVE_UNDEPLOYED=true "$CURRENT_DIR"/app_sre_create_image_catalog.sh production "$QUAY_IMAGE"
