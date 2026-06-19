#!/bin/bash

#
# Build the helm chart.
#

set -eo pipefail

log() {
  echo >&2 "${0##*/}: $*"
}

readonly chart_name="k8s-job-wrapper"
readonly chart_dir="charts"
readonly chart="${chart_dir}/${chart_name}"
readonly chart_yaml="${chart}/Chart.yaml"

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

cd "$topd"

tool() {
  "${topd}/hack/tools.sh" "$@"
}

yq() {
  tool yq "$@"
}

kustomize() {
  tool kustomize "$@"
}

helmify() {
  tool helmify "$@"
}

helm() {
  tool helm "$@"
}

helm_schema() {
  tool helm-schema "$@"
}

build_manifest() {
  local -r __image="$1"
  local -r __manifest="$2"
  mkdir -p "$(dirname "$__manifest")"
  pushd "${topd}/config/manager" >/dev/null
  kustomize edit set image controller="$__image"
  popd >/dev/null
  kustomize build config/default > "$__manifest"
}

generate_chart_yaml() {
  local -r __version="$1"
  log "Version: ${__version}"
  cat <<EOS
apiVersion: v2
name: k8s-job-wrapper
description: A Helm chart to distribute the project k8s-job-wrapper
type: application
version: ${__version}
keywords:
  - job
  - cronjob
home: https://github.com/pixiv/k8s-job-wrapper
sources:
  - https://github.com/pixiv/k8s-job-wrapper
EOS
}

generate_values_schema() {
  pushd "$chart" > /dev/null
  helm_schema
  popd > /dev/null
}

fix_serviceaccount_name() {
  pushd "$chart" > /dev/null
  local __tmp
  __tmp="$(mktemp)"
  yq '.serviceAccount.name = "k8s-job-wrapper-controller-manager"' values.yaml > "$__tmp"
  cat "$__tmp" > values.yaml
  popd > /dev/null
}

build_chart() {
  local -r __version="$1"
  local -r __manifest="$2"
  rm -rf "$chart_dir"
  helmify "$chart" < "$__manifest"
  generate_chart_yaml "$__version" > "$chart_yaml"
  generate_values_schema
  fix_serviceaccount_name
  helm lint --strict "$chart"
}


readonly version="$1"
readonly image="$2"
readonly destination="$3"
if [[ -z "$version" ]] ; then
  log "No version!"
  exit 1
fi
if [[ -z "$image" ]] ; then
  log "No image!"
  exit 1
fi
if [[ -z "$destination" ]] ; then
  log "No destination!"
  exit 1
fi

tmpd="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpd"
}
trap cleanup EXIT
readonly manifest="${tmpd}/install.yaml"
build_manifest "$image" "$manifest"
rm -rf "$chart"
build_chart "$version" "$manifest"
helm package --destination "$destination" "$chart"
