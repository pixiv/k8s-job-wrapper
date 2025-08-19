#!/bin/bash

#
# Entrypoint of dev tools.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly name="$1"
if [[ -z "$name" ]] ; then
    echo >&2 "$(basename "$0"): no tool name!"
    exit 1
fi
shift

# Install the tool
readonly toold="${d}/tools"
find_package() {
    grep "$1" "${toold}/go.mod" | grep -v 'indirect' | xargs
}

readonly bind="${d}/../bin"
mkdir -p "${bind}"
readonly binary="${bind}/${name}"
case "$name" in
    "helm")
        # The build took too long when helm was incorporated into our tools, so we download the binary directly instead.
        "${d}/setup-helm.sh" "$HELM_VERSION"
        ;;
    "go-licenses")
        # Separated from tools/ because the build won't pass, probably due to an outdated Go version.
        "${d}/setup-go-licenses.sh"
        ;;
    "pandoc")
        "${d}/setup-pandoc.sh" "$PANDOC_VERSION"
        ;;
    "setup-envtest")
        # setup-envtest requires a dedicated setup because of its dependencies on other library versions.
        "${d}/setup-envtest.sh"
        # You don't need to pass any particular arguments to run this.
        exit
        ;;
    "kubectl")
        # To avoid an excessive number of dependencies, we download the kubectl binary rather than managing it as part of tools/.
        "${d}/setup-kubectl.sh" "$KUBECTL_VERSION"
        ;;
    *)
        # Some tools rely on the calling directory, so invoking them directly from go tool can lead to unintended results.
        # Therefore, we build the binary first and then execute it.
        if [[ ! -x "$binary" ]] ; then
            echo >&2 "$(basename "$0"): build ${binary}..."
            go -C "$toold" build -o "$binary" "$(find_package "$name")"
        fi
        ;;
esac
# Run the tool
"$binary" "$@"
