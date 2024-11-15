#!/usr/bin/env bash

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
SCRIPT_BUNDLE_CONTENTS="$REPO_ROOT/hack/generate-operator-bundle-contents.py"
BASE_FOLDER=""
DIR_BUNDLE=""
DIR_EXEC=""
DIR_MANIFESTS=""

GOOS=$(go env GOOS)
OPM_VERSION="v1.23.2"
COMMAND_OPM=""
GRPCURL_VERSION="1.7.0"
COMMAND_GRPCURL=""

export REGISTRY_AUTH_FILE=${CONTAINER_ENGINE_CONFIG_DIR}/config.json

OLM_BUNDLE_VERSIONS_REPO="gitlab.cee.redhat.com/service/saas-operator-versions.git"
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

function precheck_required_files() {
    if [[ ! -x "$SCRIPT_BUNDLE_CONTENTS" ]]; then
        log "The script $SCRIPT_BUNDLE_CONTENTS cannot be run. Exiting."
        return 1
    fi
    return 0
}

function login_image_registry() {
    ${CONTAINER_ENGINE} login -u="${REGISTRY_USER}" -p="${REGISTRY_TOKEN}" ${IMAGE_REGISTRY}
}

function prepare_temporary_folders() {
    BASE_FOLDER=$(mktemp -d --suffix "-$(basename "$0")")
    DIR_BUNDLE=$(mktemp -d -p "$BASE_FOLDER" bundle.XXXX)
    DIR_MANIFESTS=$(mktemp -d -p "$DIR_BUNDLE" manifests.XXXX)
    DIR_EXEC=$(mktemp -d -p "$BASE_FOLDER" bin.XXXX)
}

function download_dependencies() {
    cd "$DIR_EXEC"

    local opm_url="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/$GOOS-amd64-opm"
    curl -sfL "${opm_url}" -o opm
    chmod +x opm
    COMMAND_OPM="$DIR_EXEC/opm"

    local grpcurl_url="https://github.com/fullstorydev/grpcurl/releases/download/v$GRPCURL_VERSION/grpcurl_${GRPCURL_VERSION}_${GOOS}_x86_64.tar.gz"
    curl -sfL "$grpcurl_url" | tar -xz -O grpcurl > "grpcurl"
    chmod +x grpcurl
    COMMAND_GRPCURL="$DIR_EXEC/grpcurl"

    cd ~-
}


function clone_versions_repo() {
    local folder="$BASE_FOLDER/$OLM_BUNDLE_VERSIONS_REPO_FOLDER"
    log "  path: $folder"

    if [[ -n "${APP_SRE_BOT_PUSH_TOKEN:-}" ]]; then
        log "Using APP_SRE_BOT_PUSH_TOKEN credentials to authenticate"
        git clone "https://app:${APP_SRE_BOT_PUSH_TOKEN}@$OLM_BUNDLE_VERSIONS_REPO" "$folder" --quiet
    else
        git clone "https://$OLM_BUNDLE_VERSIONS_REPO" "$folder" --quiet
    fi
}

function set_previous_operator_version() {
    local filename="$BASE_FOLDER/$OLM_BUNDLE_VERSIONS_REPO_FOLDER/$VERSIONS_FILE"

    if [[ ! -a "$filename" ]]; then
        log "No file $VERSIONS_FILE exist. Exiting."
        exit 1
    fi
    PREV_VERSION=$(tail -n 1 "$filename" | awk '{print $1}')
}

function setup_environment() {
    log "Login Image registry"
    login_image_registry
    log "  Successfully login to $IMAGE_REGISTRY"

    log "Generating temporary folders to contain artifacts"
    prepare_temporary_folders
    log "  base path: $BASE_FOLDER"

    log "Downloading needed commands: opm and grpcurl"
    download_dependencies
    log "  path: $DIR_EXEC"

    log "Cloning $OLM_BUNDLE_VERSIONS_REPO"
    clone_versions_repo

    log "Determining previous operator version checking $VERSIONS_FILE file"
    set_previous_operator_version
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
    cd "$DIR_BUNDLE"
    ${COMMAND_OPM} alpha bundle build --directory "$DIR_MANIFESTS" \
                        --channels "$OLM_CHANNEL" \
                        --default "$OLM_CHANNEL" \
                        --package "$OPERATOR_NAME" \
                        --tag "$OLM_BUNDLE_IMAGE_VERSION" \
                        --image-builder "$(basename "$CONTAINER_ENGINE" | awk '{print $1}')" \
                        --overwrite \
                        1>&2
    cd ~-
}

function validate_opm_bundle() {
    log "Pushing bundle image $OLM_BUNDLE_IMAGE_VERSION"
    $CONTAINER_ENGINE push "$OLM_BUNDLE_IMAGE_VERSION"

    log "Validating bundle $OLM_BUNDLE_IMAGE_VERSION"
    ${COMMAND_OPM} alpha bundle validate --tag "$OLM_BUNDLE_IMAGE_VERSION" \
                            --image-builder "$(basename "$CONTAINER_ENGINE" | awk '{print $1}')"
}

function build_opm_catalog() {
    local FROM_INDEX=""
    local PREV_COMMIT=${PREV_VERSION#*g} # remove versioning and the g commit hash prefix
    # check if the previous catalog image is available
    if [ "$(${CONTAINER_ENGINE} pull "${OLM_CATALOG_IMAGE}":"${PREV_COMMIT}" &> /dev/null; echo $?)" -eq 0 ]; then
        FROM_INDEX="--from-index ${OLM_CATALOG_IMAGE}:${PREV_COMMIT}"
        log "Index argument is $FROM_INDEX"
    fi

    log "Creating catalog image $OLM_CATALOG_IMAGE_VERSION using opm"

    ${COMMAND_OPM} index add --bundles "$OLM_BUNDLE_IMAGE_VERSION" \
                --tag "$OLM_CATALOG_IMAGE_VERSION" \
                --build-tool "$(basename "$CONTAINER_ENGINE" | awk '{print $1}')" \
                $FROM_INDEX
}

function validate_opm_catalog() {
    log "Checking that catalog we have built returns the correct version $OPERATOR_VERSION"

    local free_port=""
    local container_id=""
    local catalog_current_version=""

    free_port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()')

    log "Running $OLM_CATALOG_IMAGE_VERSION and exposing $free_port"
    container_id=$(${CONTAINER_ENGINE} run -d -p "$free_port:50051" "$OLM_CATALOG_IMAGE_VERSION")

    log "Getting current version from running catalog"
    catalog_current_version=$(
        ${COMMAND_GRPCURL} -plaintext -d '{"name": "'"$OPERATOR_NAME"'"}' \
            "localhost:$free_port" api.Registry/GetPackage | \
                jq -r '.channels[] | select(.name=="'"$OLM_CHANNEL"'") | .csvName' | \
                sed "s/$OPERATOR_NAME\.//"
    )
    log "  catalog version: $catalog_current_version"

    log "Removing docker container $container_id"
    ${CONTAINER_ENGINE} rm -f "$container_id"

    if [[ "$catalog_current_version" != "v$OPERATOR_VERSION" ]]; then
        log "Version from catalog $catalog_current_version != v$OPERATOR_VERSION"
        return 1
    fi
}

function update_versions_repo() {
    log "Adding the current version $OPERATOR_VERSION to the bundle versions file in $OLM_BUNDLE_VERSIONS_REPO"
    local folder="$BASE_FOLDER/$OLM_BUNDLE_VERSIONS_REPO_FOLDER"
    
    cd "$folder"
    
    echo "$OPERATOR_VERSION" >> "$VERSIONS_FILE"
    git add .
    message="add version $OPERATOR_VERSION

    replaces $PREV_VERSION"
    git commit -m "$message"

    log "Pushing the repository changes to $OLM_BUNDLE_VERSIONS_REPO into master branch"
    git push origin master
    cd ~-
}

function tag_and_push_images() {
    log "Tagging bundle image $OLM_BUNDLE_IMAGE_VERSION as $OLM_BUNDLE_IMAGE_LATEST"
    ${CONTAINER_ENGINE} tag "$OLM_BUNDLE_IMAGE_VERSION" "$OLM_BUNDLE_IMAGE_LATEST"

    log "Tagging catalog image $OLM_CATALOG_IMAGE_VERSION as $OLM_CATALOG_IMAGE_LATEST"
    ${CONTAINER_ENGINE} tag "$OLM_CATALOG_IMAGE_VERSION" "$OLM_CATALOG_IMAGE_LATEST"

    log "Pushing catalog image $OLM_CATALOG_IMAGE_VERSION"
    ${CONTAINER_ENGINE} push "$OLM_CATALOG_IMAGE_VERSION"

    log "Pushing bundle image $OLM_CATALOG_IMAGE_LATEST"
    ${CONTAINER_ENGINE} push "$OLM_CATALOG_IMAGE_LATEST"

    log "Pushing bundle image $OLM_BUNDLE_IMAGE_LATEST"
    ${CONTAINER_ENGINE} push "$OLM_BUNDLE_IMAGE_LATEST"
}

function main() {
    log "Building $OPERATOR_NAME version $OPERATOR_VERSION"

    precheck_required_files || return 1

    setup_environment

    build_opm_bundle
    validate_opm_bundle

    build_opm_catalog
    validate_opm_catalog

    if [[ -n "${APP_SRE_BOT_PUSH_TOKEN:-}" ]]; then
        update_versions_repo
    else
        log "APP_SRE_BOT_PUSH_TOKEN credentials were not found"
        log "it will be necessary to manually update $OLM_BUNDLE_VERSIONS_REPO repo"
    fi
    tag_and_push_images
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main
fi