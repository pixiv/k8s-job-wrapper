#!/bin/bash

set -eo pipefail

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

cd "$topd"

build() {
  docker buildx bake \
         --set "*.cache-from=type=local,src=${DOCKER_CACHE_DIR}" \
         --allow=fs=${CACHE_DIR} \
         --load \
         "$@"
}

build_with_cache_write() {
  build --set "*.cache-to=type=local,mode=max,dest=${DOCKER_CACHE_DIR}"
}


if [[ "$SKIP_DOCKER_BUILD" = "true" ]] ; then
  echo Skip docker-build because SKIP_DOCKER_BUILD is true.
  exit
fi
if [[ "$DOCKER_CACHE_READONLY" = "true" ]] ; then
  build
  exit
fi

build_with_cache_write
