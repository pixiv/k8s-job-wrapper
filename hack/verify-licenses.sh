#!/bin/bash

#
# Check for licenses that are not allowed
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly tools="${d}/tools.sh"
readonly topd="${d}/.."

go_licenses() {
    "${tools}" go-licenses "$@"
}

allowed_licenses() {
    cat <<EOS | xargs | tr ' ' ','
Apache-2.0
BSD-2-Clause
BSD-3-Clause
ISC
MIT
EOS
}

cd "${topd}"
go_licenses check ./... --allowed_licenses="$(allowed_licenses)"
