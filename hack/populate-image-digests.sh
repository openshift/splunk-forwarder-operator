#!/bin/bash

set -xeuo pipefail

source hack/common.sh

usage() {
    echo "Usage: $0 FORWARDER_IMAGE_URI HEAVYFORWARDER_IMAGE_URI" >&2
    exit -1
}

discover_digest_for_image() {
    local img=$1
    skopeo inspect docker://${img} | jq -r .Digest
}

[[ $# -eq 2 ]] || usage

f_img=$1
hf_img=$2

f_digest=$(discover_digest_for_image "$f_img")
hf_digest=$(discover_digest_for_image "$hf_img")

${SED?} -i "s,^\(  *imageDigest:\).*$,\1 $f_digest,; s,^\(  *heavyForwarderDigest:\).*$,\1 $hf_digest," hack/olm-registry/olm-artifacts-template.yaml README.md
