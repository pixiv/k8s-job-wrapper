#!/bin/bash

#
# Install helm binary.
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
    log "version of helm (\$1) is required!"
    exit 1
fi
readonly dest="${2:-${bind}/helm}"
if [[ -x "$dest" ]] ; then
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
readonly url="https://get.helm.sh/helm-${version}-${OS_NAME}-${ARCH}.tar.gz"
tmpd="$(mktemp -d)"
cleanup() {
    rm -rf "$tmpd"
}
trap cleanup EXIT
cd "$tmpd"
curl -L -s -o helm.tar.gz "$url"
tar xzf helm.tar.gz
rm -f helm.tar.gz
mv "${OS_NAME}-${ARCH}/helm" "$dest"
chmod +x "$dest"
