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

export E2E_HELM=true
if [[ "$TEST_CHART_SKIP_CHART_BUILD" != "true" ]]; then
  __make chart-internal
fi
export E2E_SKIP_BUILD="$TEST_CHART_SKIP_DOCKER_BUILD"
export E2E_SKIP_LOAD="$TEST_CHART_SKIP_LOAD_IMAGE"
__make test-e2e
