#!/bin/bash

usage() {
    echo "Usage: $0 operator_uri_base git_hash operator_version forwarder_uri heavyforwarder_uri" >&2
    exit -1
}

set -exv

[[ $# -eq 5 ]] || usage

# Base URI, no tag
operator_uri_base=$1
# Used to tag registry images
git_hash=$2
# Used to tag operator image
# Format: {major}.{minor}.{commit-number}-{commit-hash-7}
# (no leading 'v')
operator_version=$3
# Full URI with tag
forwarder_uri=$4
# Full URI with tag
heavyforwarder_uri=$5

for param_name in operator_uri_base git_hash forwarder_uri heavyforwarder_uri; do
    eval param_val=\$$param_name
    if [[ -z "$param_val" ]]; then
        echo "Parameter $param_name missing or empty."
        usage
    fi
done

# FIXME: Dedup these computations from the Makefile.
operator_uri=${operator_uri_base}:v${operator_version}
catalog_uri_base=${operator_uri_base}-registry

source ${0%/*}/common.sh

# NOTE(efried): Since we reference images by digest, rebuilding an image
# with the same tag can be Bad. This is because the digest calculation
# includes metadata such as date stamp, meaning that even though the
# contents may be identical, the digest may change. In this situation,
# the original digest URI no longer has any tags referring to it, so the
# repository deletes it. This can break existing deployments referring
# to the old digest. We could have solved this issue by generating a
# permanent tag tied to each digest. We decided to do it this way
# instead.
# For testing purposes, if you need to force the build/push to rerun,
# delete the image manually.

for container in operator forwarder heavyforwarder; do
    eval container_uri=\$${container}_uri
    if image_exists_in_repo "$container_uri"; then
        echo "Image $container_uri already exists. Skipping build/push."
    else
        make build-${container} push-${container}
    fi
done

for channel in staging production; do
    catalog_uri="${catalog_uri_base}:${channel}-${git_hash}"
    # If the catalog image already exists, short out
    if image_exists_in_repo "${catalog_uri}"; then
        echo "Catalog image ${catalog_uri} already "
        echo "exists. Assuming this means the saas bundle work has also been done "
        echo "properly. Nothing to do!"
    else
        # build the CSV and create & push image catalog for the appropriate channel
        make CHANNEL=$channel build-push-catalog
    fi
done
