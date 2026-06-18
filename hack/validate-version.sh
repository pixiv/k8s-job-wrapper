#!/bin/bash

readonly version="$1"

log() {
  echo >&2 "${0##*/}: $*"
}

__grep="grep"
if [[ "$(uname -s)" = "Darwin" ]] ; then
  __grep="ggrep"
fi

is_semver() {
  log "Validate SemVer format: ${version}"
  # https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
  if echo "$version" | "$__grep" -qP '^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$' ; then
    log "${version} is SemVer"
  else
    log "${version} is not SemVer"
    return 1
  fi
}

is_not_duplicated() {
  log "Validate duplicate tag: ${version}"
  check() {
    git rev-parse "refs/tags/$1" >/dev/null 2>&1
  }
  if check "$version" || check "v${version}" ; then
    log "${version} or v${version} is already exist"
    return 1
  else
    log "${version} and v${version} are not duplicated"
  fi
}


set -eo pipefail
is_semver
is_not_duplicated
