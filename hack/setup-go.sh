#!/bin/bash

#
# Install tools via go.
#

set -eo pipefail

log() {
  echo >&2 "${0##*/}: $*"
}

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

readonly name="$1"
if [[ -z "$name" ]] ; then
  log "No tool name!"
  exit 1
fi
log "Try to install ${name}"

readonly destd="$2"
if [[ -z "$destd" ]] ; then
  log "No tool destd!"
  exit 1
fi
readonly version="$3"
log "Start to install ${name}, destd=${destd}, version=${version}"

install() {
  GOBIN="$destd" go install "$1"
}

gomodver() {
  local -r __name="$1"
  go -C "$topd" list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' "$1" 2>/dev/null
}

# The version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20).
envtest_version() {
  gomodver "sigs.k8s.io/controller-runtime" | sed -E 's/^v?([0-9]+)\.([0-9]+).*/release-\1.\2/'
}

install_setup_envtest() {
  local __version
  __version="$(envtest_version)"
  log "Install envtest@${__version}"
  install "sigs.k8s.io/controller-runtime/tools/setup-envtest@${__version}"
}

install_controller_gen() {
  install "sigs.k8s.io/controller-tools/cmd/controller-gen@${version}"
}

install_kustomize() {
  install "sigs.k8s.io/kustomize/kustomize/v5@${version}"
}

install_golangci_lint() {
  install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${version}"
  log "Building custom golangci-lint with plugins..."
  if ! "${destd}/${name}" custom --destination "$destd" --name golangci-lint-custom ; then
    rm -f "${destd}/golangci-lint"
    return 1
  fi
  mv -f "${destd}/golangci-lint-custom" "${destd}/${name}"
}

install_go_licenses() {
  install "github.com/google/go-licenses/v2@${version}"
}

install_crd_ref_docs() {
  install "github.com/elastic/crd-ref-docs@${version}"
}

install_kubebuilder() {
  install "sigs.k8s.io/kubebuilder/v4@${version}"
}

install_helmify() {
  if [[ -z "$version" ]] ; then
    log "No version!: helmify"
    return 1
  fi
  install "github.com/arttor/helmify/cmd/helmify@${version}"
}

install_helm_schema() {
  install "github.com/dadav/helm-schema/cmd/helm-schema@${version}"
}

if [[ "$name" = "setup-envtest" ]] ; then
  install_setup_envtest
  exit
fi

if [[ -z "$version" ]] ; then
  log "No version!"
  exit 1
fi

case "$name" in
  "controller-gen") install_controller_gen ;;
  "kustomize") install_kustomize ;;
  "golangci-lint") install_golangci_lint ;;
  "go-licenses") install_go_licenses ;;
  "crd-ref-docs") install_crd_ref_docs ;;
  "kubebuilder") install_kubebuilder ;;
  "helmify") install_helmify ;;
  "helm-schema") install_helm_schema ;;
  *)
    log "Unknown name!: ${name}"
    exit 1
    ;;
esac
