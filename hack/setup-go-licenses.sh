#!/bin/bash

#
# Install go-licenses binary.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly bind="${d}/../bin"
readonly binary="${bind}/go-licenses"
readonly toold="${d}/tools"

log() {
    echo >&2 "$(basename "$0"): $*"
}

readonly dest="${1:-${binary}}"
if [[ -x "$dest" ]] ; then
    exit
fi

echo >&2 "$(basename "$0"): build ${binary}..."
go -C "$toold" build -o "$binary" github.com/google/go-licenses/v2
