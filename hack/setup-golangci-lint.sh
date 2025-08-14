#!/bin/bash

#
# Install golangci-lint plugins
#

set -e

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly bind="${d}/../bin"
readonly toold="${d}/tools"

build() {
    if [[ -f "${bind}/$1" ]] ; then
        echo >&2 "$(basename "$0"): ${bind}/$1 already exists!"
        return
    fi
    echo >&2 "$(basename "$0"): build ${bind}/$1..."
    go -C "$toold" build -buildmode=plugin -o "${bind}/$1" "$2"
}

build kube-api-linter.so sigs.k8s.io/kube-api-linter/pkg/plugin
