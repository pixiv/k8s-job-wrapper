#!/bin/bash

#
# Install pandoc binary.
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
    log "version of pandoc (\$1) is required!"
    exit 1
fi
readonly dest="${2:-${bind}/pandoc}"
if [[ -x "$dest" ]] ; then
    exit
fi

tmpd="$(mktemp -d)"
cleanup() {
    rm -rf "$tmpd"
}
trap cleanup EXIT

get_darwin() {
    curl -L -s -o "${tmpd}/pandoc.zip" "https://github.com/jgm/pandoc/releases/download/${version}/pandoc-${version}-$(arch)-macOS.zip"
    cd "$tmpd"
    unzip -q pandoc.zip
    rm -f pandoc.zip
    mv pandoc* pandoc
    mv "pandoc/bin/pandoc" "$dest"
}

get_linux() {
    local -r _arch="$(arch | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')"
    curl -L -s -o "${tmpd}/pandoc.tar.gz" "https://github.com/jgm/pandoc/releases/download/${version}/pandoc-${version}-linux-${_arch}.tar.gz"
    cd "$tmpd"
    tar xzf pandoc.tar.gz
    rm -f pandoc.tar.gz
    mv pandoc* pandoc
    mv "pandoc/bin/pandoc" "$dest"
}

case "$(uname -s)" in
    "Darwin") get_darwin ;;
    "Linux") get_linux ;;
    *)
        log "$(uname -s) is not supported!"
        exit 1
        ;;
esac
