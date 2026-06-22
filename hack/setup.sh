#!/bin/bash

#
# Download and install tools.
#

set -eo pipefail

log() {
  echo >&2 "${0##*/}: $*"
}

readonly name="$1"
if [[ -z "$name" ]] ; then
  log "No tool name!"
  exit 1
fi
log "Try to install ${name}"
readonly dest="$2"
if [[ -z "$dest" ]] ; then
  log "No tool dest!"
  exit 1
fi
readonly version="$3"
if [[ -z "$version" ]] ; then
  log "No tool version!"
  exit 1
fi
log "Start to install ${name}, dest=${dest}, version=${version}"

tmpd="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpd"
}
trap cleanup EXIT
cd "$tmpd"

get_osname() {
  case "$(uname -s)" in
    "Darwin" | "Linux")
      uname -s | tr '[:upper:]' '[:lower:]'
      ;;
    *)
      log "$(uname -s) is not supported!"
      return 1
      ;;
  esac
}

get_arch() {
  arch | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/'
}

install_binary() {
  local -r __bin="$1"
  mv "${__bin}" "$dest"
  chmod +x "$dest"
}

download_and_install() {
  local -r __url="$1"
  curl -L -s -o "$name" "$__url"
  install_binary "$name"
}

install_helm() {
  curl -L -s -o helm.tar.gz "https://get.helm.sh/helm-${version}-$(get_osname)-$(get_arch).tar.gz"
  tar xzf helm.tar.gz
  install_binary "$(get_osname)-$(get_arch)/helm"
}

install_yq() {
  curl -L -s -o- "https://github.com/mikefarah/yq/releases/download/${version}/yq_$(get_osname)_$(get_arch).tar.gz" | tar xz
  install_binary "yq_$(get_osname)_$(get_arch)"
}

case "$name" in
  "kubectl")
    download_and_install "https://dl.k8s.io/release/${version}/bin/$(get_osname)/$(get_arch)/kubectl"
    ;;
  "kind")
    download_and_install "https://kind.sigs.k8s.io/dl/${version}/kind-$(get_osname)-$(get_arch)"
    ;;
  "helm") install_helm ;;
  "yq") install_yq ;;
  *)
    log "unknown tool!: ${name}"
    exit 1
esac
