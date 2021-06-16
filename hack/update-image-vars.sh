#!/bin/bash

set -xeuo pipefail

source hack/common.sh

SFI_REF="${1:-master}"
SFI_GIT_URI=${SFI_GIT_URI:-https://github.com/openshift/splunk-forwarder-images}
SFI_DIR=build/_sfi

usage() {
    echo "Usage: $0 FORWARDER_IMAGE_GIT_COMMIT-ISH" >&2
    exit -1
}

clone_or_fetch_sfi_repo() {
    if [ -d ${SFI_DIR} ]; then
        git -C ${SFI_DIR} fetch
    else
        git clone ${SFI_GIT_URI} ${SFI_DIR}
    fi
    (cd ${SFI_DIR} && git checkout --force --quiet $1)
}

[[ $# -gt 1 ]] || [[ "$*" = "-h" ]] || [[ "$*" = "--help" ]] && usage

clone_or_fetch_sfi_repo $SFI_REF

forwarder_version=$(<${SFI_DIR}/.splunk-version)
forwarder_hash=$(<${SFI_DIR}/.splunk-version-hash)
commit_hash=$(git -C ${SFI_DIR} rev-parse --short=7 HEAD)
image_tag=${forwarder_version}-${forwarder_hash}-${commit_hash}

make FORWARDER_IMAGE_TAG=${image_tag} image-digests

${SED?} -i "s,^\(FORWARDER_IMAGE_TAG\)\b.*=.*$,\1 ?= $image_tag," Makefile
${SED?} -i "s,\(current version\,.\`\|\?tag=\).*\([&\`]\),\1$image_tag\2,g" README.md
