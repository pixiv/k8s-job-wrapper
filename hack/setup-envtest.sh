#!/bin/bash

#
# Install envtest binary.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly bind="${d}/../bin"
readonly dest="${bind}/setup-envtest"
readonly topd="${d}/.."

log() {
    echo >&2 "$(basename "$0"): $*"
}

# The version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20).
envtest_version() {
    go -C "$topd" list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $2, $3}'
}

# The version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31).
envtest_k8s_version() {
    go -C "$topd" list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $3}'
}

install_envtest() {
    GOBIN="${bind}" go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@$(envtest_version)"
}

use_envtest() {
    log "Setting up envtest binaries for Kubernetes version $(envtest_k8s_version)..."
    "${dest}" use "$(envtest_k8s_version)" --bin-dir "${bind}" -p path || {
        log "Error: Failed to set up envtest binaries for version $(envtest_k8s_version)."
        exit 1
    }
}

if [[ ! -x "$dest" ]] ; then
    install_envtest
fi

use_envtest
