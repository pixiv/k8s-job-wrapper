#!/bin/bash

#
# Entrypoint of dev tools.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"

log() {
  echo >&2 "${0##*/}: $*"
}

readonly name="$1"
if [[ -z "$name" ]] ; then
  log "$(basename "$0"): no tool name!"
  exit 1
fi
shift

# Install the tool.
# If a tool called by this script does not exist, its binary must be installed.

readonly topd="${d}/.."
readonly bind="${topd}/bin"
mkdir -p "${bind}"
readonly binary="${bind}/${name}"

case "$name" in
  "clean")
    log "Clean all binaries"
    find "$bind" -type f -maxdepth 1 -delete
    exit
    ;;
esac


gomodver() {
  local -r __name="$1"
  go -C "$topd" list -m -f '{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}' "$1" 2>/dev/null
}

# The version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31).
envtest_k8s_version() {
  gomodver "k8s.io/api" | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/'
}

use_envtest() {
  log "Setting up envtest binaries for Kubernetes version $(envtest_k8s_version)..."
  "${binary}" use "$(envtest_k8s_version)" --bin-dir "${bind}" -p path || {
    log "Error: Failed to set up envtest binaries for version $(envtest_k8s_version)."
    exit 1
  }
}

setup() {
  "${d}/setup.sh" "$name" "$binary" "$@"
}

setup_go() {
  "${d}/setup-go.sh" "$name" "$bind" "$@"
}

if [[ ! -x "$binary" ]] ; then
  log "Install ${binary} ..."
  case "$name" in
    "kubectl") setup "$KUBECTL_VERSION" ;;
    "setup-envtest") setup_go ;;
    "controller-gen") setup_go "$CONTROLLER_GEN_VERSION" ;;
    "kustomize") setup_go "$KUSTOMIZE_VERSION" ;;
    "golangci-lint") setup_go "$GOLANGCI_LINT_VERSION" ;;
    "go-licenses") setup_go "$GO_LICENSES_VERSION" ;;
    "crd-ref-docs") setup_go "$CRD_REF_DOCS_VERSION" ;;
    "helm") setup "$HELM_VERSION" ;;
    "kubebuilder") setup_go "$KUBEBUILDER_VERSION" ;;
    "kind") setup "$KIND_VERSION" ;;
    "helmify") setup_go "$HELMIFY_VERSION" ;;
    "yq") setup "$YQ_VERSION" ;;
    "helm-schema") setup_go "$HELM_SCHEMA_VERSION" ;;
    *)
      log "Unknown tool!: ${name}"
      exit 1
      ;;
  esac
fi

# Run the tool.
case "$name" in
  "setup-envtest") "$binary" use "$(envtest_k8s_version)" --bin-dir "$bind" -p path ;;
  *) "$binary" "$@" ;;
esac
