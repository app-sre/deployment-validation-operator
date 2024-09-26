#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
SCRIPT_BUNDLE_CONTENTS="$REPO_ROOT/hack/generate-operator-bundle-contents.py"

OLM_BUNDLE_VERSIONS_REPO=${OLM_BUNDLE_VERSIONS_REPO:-gitlab.cee.redhat.com/service/saas-operator-versions.git}
OLM_BUNDLE_VERSIONS_REPO_BRANCH=${OLM_BUNDLE_VERSIONS_REPO_BRANCH:-master}

function log() {
    echo "$(date "+%Y-%m-%d %H:%M:%S") -- ${1}"
}

# function clone_olm_bundle_versions_repo() {
#     local saas_root_dir=${1}

#     local bundle_versions_repo_url
#     if [[ -n "${APP_SRE_BOT_PUSH_TOKEN:-}" ]]; then
#         log "Using APP_SRE_BOT_PUSH_TOKEN credentials to authenticate"
#         bundle_versions_repo_url="https://app:${APP_SRE_BOT_PUSH_TOKEN}@$OLM_BUNDLE_VERSIONS_REPO"
#     else
#         bundle_versions_repo_url="https://$OLM_BUNDLE_VERSIONS_REPO"
#     fi

#     log "Cloning $OLM_BUNDLE_VERSIONS_REPO into $saas_root_dir"
#     git clone --branch "$OLM_BUNDLE_VERSIONS_REPO_BRANCH" "$bundle_versions_repo_url" "$saas_root_dir"
# }


function main() {
    # guess the env vars
    #log "Building $OPERATOR_NAME version $OPERATOR_VERSION"

    # research if this is worthy when we know all env vars we need
    #check_required_environment || return 1
    if [[ ! -x "$SCRIPT_BUNDLE_CONTENTS" ]]; then
        log "The script $SCRIPT_BUNDLE_CONTENTS cannot be run. Exiting."
        return 1
    fi

    # check versioning with this awful function
    # get_prev_operator_version

    log "Generating temporary folder to contain artifacts"
    BASE_FOLDER=$(mktemp -d --suffix "-$(basename "$0")")
    log "  folder: $BASE_FOLDER"

    local DIR_BUNDLE=$(mktemp -d -p "$BASE_FOLDER" bundle.XXXX)
    local DIR_MANIFESTS=$(mktemp -d -p "$DIR_BUNDLE" manifests.XXXX)

    log $OPERATOR_VERSION

    python3 -m venv .venv
    source .venv/bin/activate
    pip install pyyaml
    $SCRIPT_BUNDLE_CONTENTS --name "$OPERATOR_NAME" \
                         --current-version "$OPERATOR_VERSION" \
                         --image "$OPERATOR_IMAGE" \
                         --image-tag "$OPERATOR_IMAGE_TAG" \
                         --output-dir "$DIR_MANIFESTS" \

}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main
fi