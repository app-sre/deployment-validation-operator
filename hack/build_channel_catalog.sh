#!/bin/bash

set -euo pipefail

# General vars to modify behaviour of the script
DRY_RUN=${DRY_RUN:-false}
DELETE_TEMP_DIR=${DELETE_TEMP_DIR:-true}
BRANCH=${BRANCH:-master}
SET_X=${SET_X:-false}

[[ "$SET_X" != "false" ]] && set -x

function log() {
    msg="$(date "+%Y-%m-%d %H:%M:%S")"
    [[ "$DRY_RUN" != "false" ]] && msg="$msg -- DRY-RUN"
    echo "$msg -- $1"
}

count=0
for var in BUNDLE_IMAGE \
           CATALOG_IMAGE \
           CONTAINER_ENGINE \
           CONFIG_DIR \
           CURRENT_COMMIT \
           COMMIT_NUMBER \
           OPERATOR_VERSION \
           OPERATOR_NAME \
           OPERATOR_IMAGE \
           OPERATOR_IMAGE_TAG \
           GOOS \
           GOARCH \
           OPM_VERSION \
           GRPCURL_VERSION \
           BUNDLE_VERSIONS_REPO
do
    if [ ! "${!var:-}" ]; then
      log "$var is not set"
      count=$((count + 1))
    fi
done

[[ $count -gt 0 ]] && exit 1

temp_dir=$(mktemp -d --suffix "-$(basename "$0")")
[[ "$DELETE_TEMP_DIR" == "true" ]] && trap 'rm -rf $temp_dir' EXIT

# Check we have the needed commands
# opm
opm_local_executable=$(which opm || true)
if [[ "$opm_local_executable" ]]; then
    opm_local_version=$(opm version | sed 's/.*OpmVersion:"//;s/".*//')
fi

if [[ -n "$opm_local_executable" && "$opm_local_version" == "$OPM_VERSION" ]]; then
    log "Using local opm version $opm_local_executable"
else
    opm_download_url="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/${GOOS}-${GOARCH}-opm"
    log "Downloading opm from $opm_download_url to $temp_dir/opm"
    curl -s -L "$opm_download_url" -o "$temp_dir/opm"
    chmod u+x "$temp_dir/opm"
    opm_local_executable="$temp_dir/opm"
fi

# grpcurl
grpcurl_local_executable=$(which grpcurl || true)
if [[ "$grpcurl_local_executable" ]]; then
    grpcurl_local_version=$(grpcurl -version 2>&1 | cut -d " " -f 2)
fi

if [[ -n "$grpcurl_local_executable" && "$grpcurl_local_version" == "$GRPCURL_VERSION" ]]; then
    log "Using local grpcurl version $grpcurl_local_executable"
else
    # mappings from https://github.com/fullstorydev/grpcurl/blob/master/.goreleaser.yml
    os=$GOOS
    arch=$GOARCH
    [[ "$GOOS" == "darwin" ]] && os=osx
    [[ "$GOARCH" == "386" ]] && arch=x86_32
    [[ "$GOARCH" == "amd64" ]] && arch=x86_64
    grpcurl_download_url="https://github.com/fullstorydev/grpcurl/releases/download/v$GRPCURL_VERSION/grpcurl_${GRPCURL_VERSION}_${os}_${arch}.tar.gz"
    log "Downloading opm from $grpcurl_download_url to $temp_dir/grpcurl"
    curl -s -L "$grpcurl_download_url" | tar -xzf - -C "$temp_dir" grpcurl
    chmod u+x "$temp_dir/grpcurl"
    grpcurl_local_executable="$temp_dir/grpcurl"
fi

# ./hack/generate-operator-bundle-contents.py
if [[ ! -x "./hack/generate-operator-bundle-contents.py" ]]; then
    echo "./hack/generate-operator-bundle-contents.py is either missing or non-executable"
    exit 1
fi

# Check we are running an opm supported container engine
image_builder=$(basename "$CONTAINER_ENGINE")
if [[ "$image_builder" != "docker" && "$image_builder" != "podman" ]]; then
    # opm error messages are obscure. Let's make this clear
    log "image_builder $image_builder is not one of docker or podman"
    exit 1
fi

# This is where the action starts
log "Building $OPERATOR_NAME version $OPERATOR_VERSION"

engine_cmd="$CONTAINER_ENGINE --config=$CONFIG_DIR"

# clone bundle repo containing current version
saas_operator_dir_base="$temp_dir/saas-operator-dir"
bundle_versions_file="$saas_operator_dir_base/$OPERATOR_NAME/${OPERATOR_NAME}-versions.txt"

if [[ -n "${APP_SRE_BOT_PUSH_TOKEN:-}" ]]; then
    log "Using APP_SRE_BOT_PUSH_TOKEN credentials to authenticate"
    bundle_versions_repo_url="https://app:${APP_SRE_BOT_PUSH_TOKEN}@$BUNDLE_VERSIONS_REPO"
else
    bundle_versions_repo_url="https://$BUNDLE_VERSIONS_REPO"
fi

log "Cloning $BUNDLE_VERSIONS_REPO into $saas_operator_dir_base"
git clone --branch "$BRANCH" "$bundle_versions_repo_url" "$saas_operator_dir_base"

# if the line contains SKIP it will not be included
prev_operator_version=""
prev_good_operator_version=""
skip_versions=()
if [[ -s "$bundle_versions_file" ]]; then
    log "$bundle_versions_file exists. We'll use to determine current version"

    prev_operator_version=$(tail -n 1 "$bundle_versions_file" | awk '{print $1}')

    # we traverse the bundle versions file backwards
    # we cannot use pipes here or we would lose the inner variables changes
    while read -r line; do
        if [[ "$line" == *SKIP* ]]; then
            version=$(echo "$line" | awk '{print $1}')
            skip_versions+=("$version")
        else
            prev_good_operator_version="$line"
            break
        fi
    done < <(sort -r -t . -k 3 -g "$bundle_versions_file")

    if [[ -z "$prev_good_operator_version" ]]; then
        # This means that we have skipped all the available versions. In this case we're going to use the last
        # SKIP version as the prev_good_operator_version to have something to feed replaces in the CSV
        log "No unskipped version in $bundle_versions_file. We'll use the last skipped one: ${skip_versions[0]}"
        prev_good_operator_version="${skip_versions[0]}"
    fi

    log "Previous operator version is $prev_operator_version"
    log "Previous good operator version is $prev_good_operator_version"
    [[ ${#skip_versions[@]} -gt 0 ]] && log "We will be skipping: ${skip_versions[*]}"
else
    log "No $bundle_versions_file exist. This is the first time the operator is built"
    if [[ ! -d "$(dirname "$bundle_versions_file")" ]]; then
        log "Operator directory doesn't exist in versions repository. Exiting"
        exit 1
    fi
fi

if [[ "$OPERATOR_VERSION" == "$prev_operator_version" ]]; then
    log "stopping script as $OPERATOR_VERSION version was already built, so no need to rebuild it"
    exit 0
fi

# image:tag definitions
bundle_image_current_commit="$BUNDLE_IMAGE:$CURRENT_COMMIT"
bundle_image_latest="$BUNDLE_IMAGE:latest"
catalog_image_current_commit="$CATALOG_IMAGE:$CURRENT_COMMIT"
catalog_image_latest="$CATALOG_IMAGE:latest"

# Build bundle
bundle_temp_dir=$(mktemp -d -p "$temp_dir" bundle.XXXX)
generate_csv_template_args=""
[[ -n "$prev_good_operator_version" ]] && generate_csv_template_args="--replaces $prev_good_operator_version"
if [[ ${#skip_versions[@]} -gt 0 ]]; then
    for version in "${skip_versions[@]}"; do
        generate_csv_template_args="$generate_csv_template_args --skip $version"
    done
fi

manifests_temp_dir=$(mktemp -d -p "$bundle_temp_dir" manifests.XXXX)
./hack/generate-operator-bundle-contents.py --name "$OPERATOR_NAME" \
                                            --current-version "$OPERATOR_VERSION" \
                                            --image "$OPERATOR_IMAGE" \
                                            --image-tag "$OPERATOR_IMAGE_TAG" \
                                            --output-dir "$manifests_temp_dir" \
                                            $generate_csv_template_args

# opm won't get anything locally, so we need to push the bundle even in dry run mode
# we will use a different tag to make sure those leftovers are clearly recognized
# TODO: remove this tag if we're in dry-run mode in the cleanup trap script
[[ "$DRY_RUN" == "false" ]] || bundle_image_current_commit="${bundle_image_current_commit}-dryrun"
log "Creating bundle image $bundle_image_current_commit"
pushd "$bundle_temp_dir"
opm alpha bundle build --directory "$manifests_temp_dir" \
                       --channels alpha \
                       --default alpha \
                       --package "$OPERATOR_NAME" \
                       --tag "$bundle_image_current_commit" \
                       --image-builder "$image_builder" \
                       --overwrite
popd

log "Pushing bundle image $bundle_image_current_commit"
$engine_cmd push "$bundle_image_current_commit"

# Make sure this is run after pushing the image
log "Validating bundle $bundle_image_current_commit"
$opm_local_executable alpha bundle validate --tag "$bundle_image_current_commit"  --image-builder "$image_builder"

log "Tagging bundle image $bundle_image_current_commit as $bundle_image_latest"
$CONTAINER_ENGINE tag "$bundle_image_current_commit" "$bundle_image_latest"

from_arg=""
if [[ "$prev_operator_version" ]]; then
    prev_commit=${prev_operator_version#*-}
    from_arg="--from-index $CATALOG_IMAGE:$prev_commit"
fi

log "Creating catalog image $catalog_image_current_commit using opm"
$opm_local_executable index add --bundles "$bundle_image_current_commit" \
                                --tag "$catalog_image_current_commit" \
                                --build-tool "$image_builder" \
                                $from_arg

# Check that catalog works fine
log "Checking that catalog we have built returns the correct version $OPERATOR_VERSION"

free_port=$(python -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()')

log "Running $catalog_image_current_commit and exposing $free_port"
catalog_container_id=$($CONTAINER_ENGINE run -d -p "$free_port:50051" "$catalog_image_current_commit")

log "Getting current version from running catalog"
current_version_from_catalog=$(
    $grpcurl_local_executable -plaintext -d '{"name": "'"$OPERATOR_NAME"'"}' \
        "localhost:$free_port" api.Registry/GetPackage | \
            jq -r '.channels[] | select(.name=="alpha") | .csvName' | \
            sed "s/$OPERATOR_NAME\.//"
)

log "Removing docker container $catalog_container_id"
$CONTAINER_ENGINE rm -f "$catalog_container_id"

if [[ "$current_version_from_catalog" != "v$OPERATOR_VERSION" ]]; then
    log "Version from catalog $current_version_from_catalog != v$OPERATOR_VERSION"
    exit 1
fi

log "Tagging catalog image $catalog_image_current_commit as $catalog_image_latest"
$CONTAINER_ENGINE tag "$catalog_image_current_commit" "$catalog_image_latest"

# create package yaml
log "Storing current state in the $BUNDLE_VERSIONS_REPO repository"

# add, commit & push
log "Adding the current version $OPERATOR_VERSION to the bundle versions file"
echo "$OPERATOR_VERSION" >> "$bundle_versions_file"

cd "$saas_operator_dir_base/$OPERATOR_NAME"
git add .
message="add version $OPERATOR_VERSION"
[[ "$prev_operator_version" ]] && message="$message

replaces $prev_operator_version"

git commit -m "$message"

log "Pushing the repository changes to $BUNDLE_VERSIONS_REPO"
[[ "$DRY_RUN" == "false" ]] && git push origin master
cd -

log "Pushing catalog image $catalog_image_current_commit"
[[ "$DRY_RUN" == "false" ]] && $engine_cmd push "$catalog_image_current_commit"

log "Pushing bundle image $catalog_image_latest"
[[ "$DRY_RUN" == "false" ]] && $engine_cmd push "$catalog_image_latest"

log "Pushing bundle image $bundle_image_latest"
[[ "$DRY_RUN" == "false" ]] && $engine_cmd push "$bundle_image_latest"

exit 0
