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

OLM_CATALOG_IMAGE_VERSION="${OLM_CATALOG_IMAGE}:${CURRENT_COMMIT}"
OLM_CATALOG_IMAGE_LATEST="${OLM_CATALOG_IMAGE}:latest"

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

function validate_opm_bundle() {
    log "Pushing bundle image $OLM_BUNDLE_IMAGE_VERSION"
    $CONTAINER_ENGINE push "$OLM_BUNDLE_IMAGE_VERSION"

    log "Validating bundle $OLM_BUNDLE_IMAGE_VERSION"
    opm alpha bundle validate --tag "$OLM_BUNDLE_IMAGE_VERSION" \
                            --image-builder $(basename "$CONTAINER_ENGINE" | awk '{print $1}')
}

function build_opm_catalog() {
    local FROM_INDEX=""
    local PREV_COMMIT=${PREV_VERSION#*g} # remove versioning and the g commit hash prefix
    # check if the previous catalog image is available
    if [ $(${CONTAINER_ENGINE} pull ${OLM_CATALOG_IMAGE}:${PREV_COMMIT} &> /dev/null; echo $?) -eq 0 ]; then
        FROM_INDEX="--from-index ${OLM_CATALOG_IMAGE}:${PREV_COMMIT}"
        log "Index argument is $FROM_INDEX"
    fi

    log "Creating catalog image $OLM_CATALOG_IMAGE_VERSION using opm"

    opm index add --bundles "$OLM_BUNDLE_IMAGE_VERSION" \
                --tag "$OLM_CATALOG_IMAGE_VERSION" \
                --build-tool $(basename "$CONTAINER_ENGINE" | awk '{print $1}') \
                $FROM_INDEX
}

function validate_opm_catalog() {
    log "Checking that catalog we have built returns the correct version $OPERATOR_VERSION"

    local FREE_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()')

    log "Running $OLM_CATALOG_IMAGE_VERSION and exposing $FREE_PORT"
    local CONTAINER_ID=$(${CONTAINER_ENGINE} run -d -p "$FREE_PORT:50051" "$OLM_CATALOG_IMAGE_VERSION")

    log "Getting current version from running catalog"
    local CATALOG_CURRENT_VERSION=$(
        grpcurl -plaintext -d '{"name": "'"$OPERATOR_NAME"'"}' \
            "localhost:$FREE_PORT" api.Registry/GetPackage | \
                jq -r '.channels[] | select(.name=="'"$OLM_CHANNEL"'") | .csvName' | \
                sed "s/$OPERATOR_NAME\.//"
    )
    log "  catalog version: $CATALOG_CURRENT_VERSION"

    log "Removing docker container $CONTAINER_ID"
    ${CONTAINER_ENGINE} rm -f "$CONTAINER_ID"

    if [[ "$CATALOG_CURRENT_VERSION" != "v$OPERATOR_VERSION" ]]; then
        log "Version from catalog $CATALOG_CURRENT_VERSION != v$OPERATOR_VERSION"
        return 1
    fi
}

function main() {
    log "Building $OPERATOR_NAME version $OPERATOR_VERSION"

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
    validate_opm_bundle

    build_opm_catalog
    validate_opm_catalog

    log "Tagging bundle image $OLM_BUNDLE_IMAGE_VERSION as $OLM_BUNDLE_IMAGE_LATEST"
    $CONTAINER_ENGINE tag "$OLM_BUNDLE_IMAGE_VERSION" "$OLM_BUNDLE_IMAGE_LATEST"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi