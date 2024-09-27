#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
SCRIPT_BUNDLE_CONTENTS="$REPO_ROOT/hack/generate-operator-bundle-contents.py"
BASE_FOLDER=""
DIR_BUNDLE=""
DIR_MANIFESTS=""

OLM_BUNDLE_VERSIONS_REPO="https://gitlab.cee.redhat.com/service/saas-operator-versions.git"
OLM_BUNDLE_VERSIONS_REPO_FOLDER="versions_repo"
VERSIONS_FILE="deployment-validation-operator/deployment-validation-operator-versions.txt"
PREV_VERSION=""

OLM_BUNDLE_IMAGE_VERSION="${OLM_BUNDLE_IMAGE}:g${CURRENT_COMMIT}"
OLM_BUNDLE_IMAGE_LATEST="${OLM_BUNDLE_IMAGE}:latest"


function log() {
    echo "$(date "+%Y-%m-%d %H:%M:%S") -- ${1}"
}

function prepare_temporary_folders() {
    log "Generating temporary folders to contain artifacts"
    BASE_FOLDER=$(mktemp -d --suffix "-$(basename "$0")")
    DIR_BUNDLE=$(mktemp -d -p "$BASE_FOLDER" bundle.XXXX)
    DIR_MANIFESTS=$(mktemp -d -p "$DIR_BUNDLE" manifests.XXXX)
    log "  base path: $BASE_FOLDER"
}

function clone_versions_repo() {
    log "Cloning $OLM_BUNDLE_VERSIONS_REPO"
    local folder="$BASE_FOLDER/$OLM_BUNDLE_VERSIONS_REPO_FOLDER"
    git clone $OLM_BUNDLE_VERSIONS_REPO $folder --quiet
    log "  path: $folder"
}

function set_previous_operator_version() {
    log "Determining previous operator version checking $VERSIONS_FILE file"
    local filename="$BASE_FOLDER/$OLM_BUNDLE_VERSIONS_REPO_FOLDER/$VERSIONS_FILE"
    if [[ ! -a "$filename" ]]; then
        log "No file $VERSIONS_FILE exist. Exiting."
        exit 1
    fi
    PREV_VERSION=$(tail -n 1 "$filename" | awk '{print $1}')
    log "  previous version: $PREV_VERSION"
}

function build_opm_bundle() {
    # set venv with needed dependencies
    python3 -m venv .venv; source .venv/bin/activate; pip install pyyaml

    log "Generating patched bundle contents"
    $SCRIPT_BUNDLE_CONTENTS --name "$OPERATOR_NAME" \
                         --current-version "$OPERATOR_VERSION" \
                         --image "$OPERATOR_IMAGE" \
                         --image-tag "$OPERATOR_IMAGE_TAG" \
                         --output-dir "$DIR_MANIFESTS" \
                         --replaces "$PREV_VERSION"

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

function main() {
    # guess the env vars
    #log "Building $OPERATOR_NAME version $OPERATOR_VERSION"

    # research if this is worthy when we know all env vars we need
    #check_required_environment || return 1
    if [[ ! -x "$SCRIPT_BUNDLE_CONTENTS" ]]; then
        log "The script $SCRIPT_BUNDLE_CONTENTS cannot be run. Exiting."
        return 1
    fi

    prepare_temporary_folders
    clone_versions_repo
    set_previous_operator_version
    build_opm_bundle


}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi