#!/bin/bash

set -xeuo pipefail

source hack/common.sh

SFI_REF="${1:-master}"
SFI_GIT_URI=${SFI_GIT_URI:-https://github.com/openshift/splunk-forwarder-images}

usage() {
    echo "Usage: $0 FORWARDER_IMAGE_GIT_COMMIT-ISH" >&2
    exit -1
}

[[ $# -gt 1 ]] || [[ "$*" = "-h" ]] || [[ "$*" = "--help" ]] && usage

commit_hash=$(git ls-remote ${SFI_GIT_URI} "refs/heads/${SFI_REF}" "refs/tags/${SFI_REF}" | awk -v ref="${SFI_REF}" '$2 == "refs/heads/"ref || $2 == "refs/tags/"ref {print $1; exit}')
if [[ -z "${commit_hash}" ]]; then
    # Not an exact branch/tag match; assume SFI_REF is already a commit-ish.
    commit_hash=${SFI_REF}
fi
image_tag=$(echo "${commit_hash}" | head -c7)

make FORWARDER_IMAGE_TAG=${image_tag} image-digests

${SED?} -i "s,^\(FORWARDER_IMAGE_TAG\)\b.*=.*$,\1 ?= $image_tag," Makefile
${SED?} -i "s,\(current version\,.\`\|\?tag=\).*\([&\`]\),\1$image_tag\2,g" README.md
