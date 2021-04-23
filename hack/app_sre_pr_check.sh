#!/bin/bash

# AppSRE team CD

set -exv

CURRENT_DIR=$(dirname "$0")

python "$CURRENT_DIR"/validate_yaml.py "$CURRENT_DIR"/../deploy/crds
if [ "$?" != "0" ]; then
    exit 1
fi

make build
