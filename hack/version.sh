#!/bin/bash

#
# Read the version of the project.
#

set -eo pipefail

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly topd="${d}/.."

cat "${topd}/VERSION" | tr -d "\n"
