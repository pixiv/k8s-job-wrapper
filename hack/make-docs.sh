#!/bin/bash

#
# Generate CRD documents.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly topd="${d}/.."

log() {
    echo >&2 "$(basename "$0"): $*"
}

k8s_version() {
    go -C "$topd" list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $3}'
}

mkdir -p "$dest"
"${d}/docs/make.sh" "$(k8s_version)"
