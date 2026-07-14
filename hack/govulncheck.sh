#!/bin/bash

set -eo pipefail

readonly d="$(cd "$(dirname "$0")" || exit 1 ; pwd)"
readonly root="${d}/.."

# Run govulncheck in the repository root to ensure go.mod is correctly discovered.
# We temporarily disable 'set -e' to capture govulncheck's exit code without aborting the script.
govulncheck() {
  pushd "$root" >/dev/null
  set +e
  "${d}/tools.sh" govulncheck "$@"
  __exit_code=$?
  set -e
  popd >/dev/null
  return "$__exit_code"
}

# Extract only the vulnerability ID, description, and the "Fixed in" line from the raw output.
# This omits the noisy, long call stacks to provide a cleaner summary at the end of the log.
summary() {
  awk '
$1 ~ /^Vulnerability/ {x = 1}
$0 ~ /Fixed in:/ {print; x = 0}
{if (x == 1) print}'
}

log() {
  echo >&2 "${0##*/}: $*"
}

log "Running govulncheck to scan for vulnerabilities..."

# Use a temporary file to store the potentially large scan log.
# A trap is set to guarantee the file is cleaned up even if the script is interrupted.
out="$(mktemp)"
cleanup() {
  rm -f "$out"
}
trap cleanup EXIT

if govulncheck "$@" > "$out" 2>&1 ; then
  log "Success: No vulnerabilities found."
  exit
fi

# Count vulnerabilities that have an available fix.
# A line containing "Fixed in:" represents a vulnerability, and if it does not contain "N/A", a fix is available.
# We temporarily disable 'set -e' because grep will exit with code 1 if no matches are found.
set +e
fixable_count=$(grep "Fixed in:" "$out" | grep -vc "N/A")
set -e

if [[ "$fixable_count" -gt 0 ]] ; then
  echo "::error title=govulncheck::Found ${fixable_count} vulnerabilities with available fixes."
else
  echo "::warning title=govulncheck::Found vulnerabilities, but none have known fixes. CI will pass."
fi

summary < "$out"

echo "::group::govulncheckのログを表示する"
cat "$out"
echo "::endgroup::"

# Exit with 0 (success) if there are no fixable vulnerabilities, allowing the CI to pass.
# Otherwise, exit with a non-zero code to fail the CI run.
[[ "$fixable_count" = 0 ]]
