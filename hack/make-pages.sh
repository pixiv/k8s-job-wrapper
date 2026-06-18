#!/bin/bash

#
# Generate CRD pages.
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

k8s_job_wrapper_version() {
    "${d}/version.sh"
}

mkdir -p "$dest"
"${d}/docs/pages.sh" "$dest" "$(k8s_job_wrapper_version)"
