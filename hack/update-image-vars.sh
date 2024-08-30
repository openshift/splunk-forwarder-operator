#!/bin/bash

set -xeuo pipefail

source hack/common.sh

SFI_REF="${1:-master}"
SFI_GIT_URI=${SFI_GIT_URI:-https://github.com/openshift/splunk-forwarder-images}
SFI_RAW=https://raw.githubusercontent.com/openshift/splunk-forwarder-images/${SFI_REF}

usage() {
    echo "Usage: $0 FORWARDER_IMAGE_GIT_COMMIT-ISH" >&2
    exit -1
}

[[ $# -gt 1 ]] || [[ "$*" = "-h" ]] || [[ "$*" = "--help" ]] && usage

forwarder_version=$(curl ${SFI_RAW}/.splunk-version)
forwarder_hash=$(curl ${SFI_RAW}/.splunk-version-hash)
commit_hash=$(git ls-remote ${SFI_GIT_URI} ${SFI_REF} | awk '{print $1}' | head -c7)
image_tag=${forwarder_version}-${forwarder_hash}-${commit_hash}

make FORWARDER_IMAGE_TAG=${image_tag} image-digests

${SED?} -i "s,^\(FORWARDER_IMAGE_TAG\)\b.*=.*$,\1 ?= $image_tag," Makefile
${SED?} -i "s,\(current version\,.\`\|\?tag=\).*\([&\`]\),\1$image_tag\2,g" README.md
