#!/bin/bash

#
# Generate HTML to redirect GitHub Pages to docs.md.
#

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"

log() {
  echo >&2 "$(basename "$0"): $*"
}

copy_assets() {
  local -r __dest="$1"
  cp -r "${d}/kustomize" "$__dest"
}

# Convert markdown into html.
# $1: k8s-job-wrapper-version
# $2: html title
generate_html() {
  local -r __version="$1"
  local -r __title="$2"
  local -r __url="https://github.com/pixiv/k8s-job-wrapper/blob/v${__version}/hack/docs/docs.md"
  cat <<EOS
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" lang xml:lang>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=yes" />
  <meta http-equiv="refresh" content="0; url=${__url}">
  <title>${__title}</title>
</head>
<body>
  <p>This content has moved to <a href="${__url}">${__url}</a>.</p>
</body>
</html>
EOS
}

set -e
set -o pipefail

readonly dest="$1"
readonly version="$2"
if [[ -z "$dest" ]] ; then
  log "dest (\$1) is required!"
  exit 1
fi
if [[ -z "$version" ]] ; then
  log "version (\$2) is required!"
  exit 1
fi

readonly title="CRD of pixiv.net"
generate_html "${version}" "${title}" > "${dest}/index.html"
copy_assets "$dest"
