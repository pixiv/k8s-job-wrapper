#!/bin/bash

#
# Test the chart is installable
#

set -eo pipefail

log() {
  echo >&2 "${0##*/}: $*"
}

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

readonly chart="$1"
if [[ -z "$chart" ]] ; then
  log "No chart!"
  exit 1
fi

__make() {
  make -C "$topd" "$@"
}

helm() {
  "${d}/tools.sh" helm "$@"
}

kubectl() {
  "${d}/tools.sh" kubectl "$@"
}

if [[ "$TEST_CHART_SKIP_CHART_BUILD" != "true" ]]; then
  __make chart-internal
fi
if [[ "$TEST_CHART_SKIP_DOCKER_BUILD" != "true" ]]; then
  DOCKER_CACHE_READONLY=true __make docker-build
  __make load-image
fi

cd "$topd"
helm install -n k8s-job-wrapper-system --create-namespace k8s-job-wrapper "$chart" --wait --timeout=5m
kubectl apply -k config/samples
kubectl wait --for=condition=Available jobs.pixiv.net job-sample --timeout=1m
kubectl wait --for=condition=Complete job -l jobs.pixiv.net/name=job-sample --timeout=2m
