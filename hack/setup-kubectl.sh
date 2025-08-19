#!/bin/bash

#
# Install kubectl binary.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly bind="${d}/../bin"

log() {
    echo >&2 "$(basename "$0"): $*"
}

readonly version="$1"
if [[ -z "$version" ]] ; then
    log "version of kubectl (\$1) is required!"
    exit 1
fi
readonly dest="${2:-${bind}/kubectl}"
if [[ -x "$dest" ]] ; then
    # The e2e tests also check stderr, so any output to stderr will cause a failure.
    exit
fi

OS_NAME="$(uname -s)"
case "${OS_NAME}" in
    "Darwin" | "Linux")
        OS_NAME="$(echo "$OS_NAME" | tr '[:upper:]' '[:lower:]')"
    ;;
    *)
        log "${OS_NAME} is not supported!"
        exit 1
        ;;
esac

ARCH="$(arch | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')"
readonly url="https://dl.k8s.io/release/${version}/bin/${OS_NAME}/${ARCH}/kubectl"
curl -L -s -o "$dest" "$url"
chmod +x "$dest"
