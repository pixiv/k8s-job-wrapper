#!/bin/bash

#
# Generate Helm chart.
#

set -e
set -o pipefail

readonly d="$(cd "$(dirname "$0")" || exit ; pwd)"
readonly topd="${d}/../.."
cd "$topd" || exit 1
readonly chartd="${topd}/dist/chart"
readonly tools="${topd}/hack/tools.sh"
readonly chartyaml="${d}/Chart.yaml.tmpl"

log() {
    echo >&2 "$(basename "$0"): $*"
}

kubebuilder() {
    "${tools}" kubebuilder "$@"
}

helm() {
    "${tools}" helm "$@"
}

generate_chart() {
    rm -rf "$chartd"
    # https://book.kubebuilder.io/plugins/available/helm-v1-alpha
    kubebuilder edit --plugins=helm.kubebuilder.io/v1-alpha
}

_sed() {
    if [[ "$(uname -s)" == "Darwin" ]] ; then
        gsed "$@"
    else
        sed "$@"
    fi
}

edit_chart() {
    set -x
    local -r _version="$1"
    local -r _repo="$2"
    local -r _tag="$3"

    VERSION="$_version" envsubst < "$chartyaml" > "${chartd}/Chart.yaml"
    _sed -i -E "s|repository: .+$|repository: ${_repo}|" "${chartd}/values.yaml"
    _sed -i -E "s|tag: .+$|tag: ${_tag}|" "${chartd}/values.yaml"
}


readonly chart_version="$1"
readonly controller_image_repo="$2"
readonly controller_image_tag="$3"
readonly destination="${4:-.}"  # directory to write the chart
if [[ -z "$chart_version" ]] ; then
    log "chart_version (\$1) is required!"
    exit 1
fi
if [[ -z "$controller_image_repo" ]] ; then
    log "controller_image_repo (\$2) is required!"
    exit 1
fi
if [[ -z "$controller_image_tag" ]] ; then
    log "controller_image_tag (\$3) is requried!"
    exit 1
fi

generate_chart
edit_chart "$chart_version" "$controller_image_repo" "$controller_image_tag"
helm lint --strict "$chartd"
helm package --destination "$destination" "$chartd"
