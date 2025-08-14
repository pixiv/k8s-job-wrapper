#!/bin/bash

#
# Install CRDs and deploy controller into the kind cluster
#

log() {
    echo >&2 "$(basename "$0"): $*"
}

readonly kind_image_name="$1"
readonly controller_image_name="$2"
if [[ -z "$kind_image_name" ]] ; then
    log "kind_image_name (\$1) is required!"
    exit 1
fi
if [[ -z "$controller_image_name" ]] ; then
    log "controller_image_name (\$2) is required!"
    exit 1
fi

set -o pipefail
set -ex

make
make docker-build
if ! kind get clusters | grep -q kind ; then
    kind create cluster --image "$kind_image_name"
fi
kind load docker-image "$controller_image_name"
make install
make deploy IMG="$controller_image_name"
