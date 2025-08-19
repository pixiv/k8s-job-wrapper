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

readonly dest="$1"
if [[ -z "$dest" ]] ; then
    log "dest (\$1) is required!"
    exit 1
fi

k8s_version() {
    go -C "$topd" list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $3}'
}

mkdir -p "$dest"
"${d}/docs/make.sh" "${dest}/index.html" "$(k8s_version)"
