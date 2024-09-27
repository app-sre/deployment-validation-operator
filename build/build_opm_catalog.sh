#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
SCRIPT_BUNDLE_CONTENTS="$REPO_ROOT/hack/generate-operator-bundle-contents.py"

OLM_BUNDLE_VERSIONS_REPO="https://gitlab.cee.redhat.com/service/saas-operator-versions.git"

OLM_BUNDLE_IMAGE_VERSION="${OLM_BUNDLE_IMAGE}:g${CURRENT_COMMIT}"
OLM_BUNDLE_IMAGE_LATEST="${OLM_BUNDLE_IMAGE}:latest"


function log() {
    echo "$(date "+%Y-%m-%d %H:%M:%S") -- ${1}"
}

function clone_versions_repo() {
    log "Cloning $OLM_BUNDLE_VERSIONS_REPO"
    local folder="$BASE_FOLDER/versions_repo"
    git clone $OLM_BUNDLE_VERSIONS_REPO $folder --quiet
    log "  path: $folder"
}

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
    log "  path: $BASE_FOLDER"

    local DIR_BUNDLE=$(mktemp -d -p "$BASE_FOLDER" bundle.XXXX)
    local DIR_MANIFESTS=$(mktemp -d -p "$DIR_BUNDLE" manifests.XXXX)

    
    clone_versions_repo

    # move this to function
    python3 -m venv .venv
    source .venv/bin/activate
    pip install pyyaml

    log "Generating patched bundle contents"
    $SCRIPT_BUNDLE_CONTENTS --name "$OPERATOR_NAME" \
                         --current-version "$OPERATOR_VERSION" \
                         --image "$OPERATOR_IMAGE" \
                         --image-tag "$OPERATOR_IMAGE_TAG" \
                         --output-dir "$DIR_MANIFESTS" \
                        # missing args from versioning

    log "Creating bundle image $OLM_BUNDLE_IMAGE_VERSION"
    cd $DIR_BUNDLE
    opm alpha bundle build --directory "$DIR_MANIFESTS" \
                        --channels "$OLM_CHANNEL" \
                        --default "$OLM_CHANNEL" \
                        --package "$OPERATOR_NAME" \
                        --tag "$OLM_BUNDLE_IMAGE_VERSION" \
                        --image-builder $(basename "$CONTAINER_ENGINE" | awk '{print $1}') \
                        --overwrite \
                        1>&2
    cd -

}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi