#!/bin/bash

set -exv

BRANCH_CHANNEL="$1"
# Base image, no tag
OPERATOR_IMAGE_BASE="$2"
GIT_HASH="$3"
OPERATOR_VERSION="$4"

# Get the image URI as repo URL + image digest
IMAGE_TAG_URI=${OPERATOR_IMAGE_BASE}:v${OPERATOR_VERSION}
IMAGE_DIGEST=$(skopeo inspect docker://${IMAGE_TAG_URI} | jq -r .Digest)
if [[ -z "$IMAGE_DIGEST" ]]; then
    echo "Couldn't discover IMAGE_DIGEST for docker://${IMAGE_TAG_URI}!"
    exit 1
fi
REPO_DIGEST=${OPERATOR_IMAGE_BASE}@${IMAGE_DIGEST}


GIT_COMMIT_COUNT=$(git rev-list $(git rev-list --max-parents=0 HEAD)..HEAD --count)

# clone bundle repo
SAAS_OPERATOR_DIR="saas-splunk-forwarder-operator-bundle"
BUNDLE_DIR="$SAAS_OPERATOR_DIR/splunk-forwarder-operator/"

rm -rf "$SAAS_OPERATOR_DIR"

# Allow env override of SaaS bundle repo
if [[ -z "$GIT_PATH" ]]; then
    GIT_PATH=https://app:"${APP_SRE_BOT_PUSH_TOKEN}"@gitlab.cee.redhat.com/service/saas-splunk-forwarder-operator-bundle.git
fi

git clone --branch "$BRANCH_CHANNEL" "$GIT_PATH" "$SAAS_OPERATOR_DIR"

# remove any versions more recent than deployed hash
REMOVED_VERSIONS=""
if [[ "$BRANCH_CHANNEL" == "production" ]]; then
    DEPLOYED_HASH=$(
        curl -s "https://gitlab.cee.redhat.com/service/app-interface/raw/master/data/services/osd-operators/cicd/saas/saas-splunk-forwarder-operator.yaml" | \
            docker run --rm -i quay.io/app-sre/yq:3.4.1 yq r - "resourceTemplates[*].targets(namespace.\$ref==/services/osd-operators/namespaces/hivep01ue1/cluster-scope.yml).ref"
    )

    # Ensure that our query for the current deployed hash worked
    # Validate that our DEPLOYED_HASH var isn't empty.
    # Although we have `set -e` defined the docker container isn't returning
    # an error and allowing the script to continue
    echo "Current deployed production HASH: $DEPLOYED_HASH"

    if [[ ! "${DEPLOYED_HASH}" =~ [0-9a-f]{40} ]]; then
        echo "Error discovering current production deployed HASH"
        exit 1
    fi

    delete=false
    # Sort based on commit number
    for version in $(ls $BUNDLE_DIR | sort -t . -k 3 -g); do
        # skip if not directory
        [ -d "$BUNDLE_DIR/$version" ] || continue

        if [[ "$delete" == false ]]; then
            short_hash=$(echo "$version" | cut -d- -f2)

            if [[ "$DEPLOYED_HASH" == "${short_hash}"* ]]; then
                delete=true
            fi
        else
            rm -rf "${BUNDLE_DIR:?BUNDLE_DIR var not set}/$version"
            REMOVED_VERSIONS="$version $REMOVED_VERSIONS"
        fi
    done
fi

# generate bundle
PREV_VERSION=$(ls "$BUNDLE_DIR" | sort -t . -k 3 -g | tail -n 1)

./hack/generate-operator-bundle.py \
    "$BUNDLE_DIR" \
    "$PREV_VERSION" \
    "$OPERATOR_VERSION" \
    "$REPO_DIGEST"

NEW_VERSION=$(ls "$BUNDLE_DIR" | sort -t . -k 3 -g | tail -n 1)

if [ "$NEW_VERSION" = "$PREV_VERSION" ]; then
    # stopping script as that version was already built, so no need to rebuild it
    exit 0
fi

# create package yaml
cat <<EOF > $BUNDLE_DIR/splunk-forwarder-operator.package.yaml
packageName: splunk-forwarder-operator
channels:
- name: $BRANCH_CHANNEL
  currentCSV: splunk-forwarder-operator.v${NEW_VERSION}
EOF

# add, commit & push
pushd $SAAS_OPERATOR_DIR

git add .

MESSAGE="add version $GIT_COMMIT_COUNT-$GIT_HASH

replaces $PREV_VERSION
removed versions: $REMOVED_VERSIONS"

git commit -m "$MESSAGE"
git push origin "$BRANCH_CHANNEL"

popd

# build the registry image
REGISTRY_IMG="${OPERATOR_IMAGE_BASE}-registry"
DOCKERFILE_REGISTRY="Dockerfile.olm-registry"

cat <<EOF > $DOCKERFILE_REGISTRY
FROM quay.io/openshift/origin-operator-registry:4.7.0

COPY $SAAS_OPERATOR_DIR manifests
RUN initializer --permissive

CMD ["registry-server", "-t", "/tmp/terminate.log"]
EOF

docker build -f $DOCKERFILE_REGISTRY --tag "${REGISTRY_IMG}:${BRANCH_CHANNEL}-${GIT_HASH}" .

# push image
skopeo copy --dest-creds "${QUAY_USER}:${QUAY_TOKEN}" \
    "docker-daemon:${REGISTRY_IMG}:${BRANCH_CHANNEL}-${GIT_HASH}" \
    "docker://${REGISTRY_IMG}:${BRANCH_CHANNEL}-${GIT_HASH}"
