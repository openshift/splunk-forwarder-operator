#!/bin/bash

# check-image-against-osd-sre-clair.sh

# Get the package list and ask the osd-sre-inventory clair if they're ok.
# Exits with a failure if there is a vulnerable package in the container.

# for testing, known state images (on 2020-07-21)
# ./check-image-against-osd-sre-clair.sh ubi8-minimal:latest #RPM good
# ./check-image-against-osd-sre-clair.sh ubi8-minimal:8.1 #RPM bad
# ./check-image-against-osd-sre-clair.sh ubuntu:latest #DEB good
# ./check-image-against-osd-sre-clair.sh debian:buster-20170620 #DEB bad
# ./check-image-against-osd-sre-clair.sh alpine:latest #APK good
# ./check-image-against-osd-sre-clair.sh alpine:3.9.2 #APK bad

CLAIR_SERVER=${CLAIR_SERVER:-https://clair.apps.osd-v4prod-aws.n2a0.p1.openshiftapps.com}

IMAGE="$1"

CONTAINER_ENGINE=${CONTAINER_ENGINE:-docker}

RPM_Q="/usr/bin/rpmquery -qa 2>/dev/null"
DPKG_Q='/usr/bin/dpkg-query -W -f "\${Package}-\${Version}\n" 2>/dev/null'
APK_Q='/sbin/apk version \* |sed -e "s/ .*//g;" 2>/dev/null'

TRY_ALL="$RPM_Q || $DPKG_Q || $APK_Q"

PACKAGES=$($CONTAINER_ENGINE run --rm "$IMAGE" /bin/sh -c "$TRY_ALL" |
	sort -Vu | jq -cR -s  '{Packages: (.|split("\n")|select(.!="")|sort)}')

if jq --exit-status '.Packages|length == 0' <<< "$PACKAGES" &>/dev/null;
then
	echo "$0: FAIL: I don't understand your package manager." >&2
	exit 1
fi

ERRATA=$(curl -s -X POST ${CLAIR_SERVER}/packages -H application/json --data @-<<<"${PACKAGES}")

echo -n "Vulnerabilties: "
jq --exit-status '.Vulnerabilities, (.Vulnerabilities|length == 0)' <<< "$ERRATA"

