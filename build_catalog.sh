#!/bin/bash

function log() {
    echo "$(date "+%Y-%m-%d %H:%M:%S") -- $1"
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
           CSV \
           GOOS \
           GOARCH \
           OPM_VERSION \
           BUNDLE_VERSIONS_REPO \
           BRANCH_CHANNEL
do
    if [ ! "${!var}" ]; then
      log "$var is not set"
      count=$((count + 1))
    fi
done

[[ $count -gt 0 ]] && exit 1

# We shouln't need set -u as we have checked for the vars before
# but let's play safe
set -euo pipefail

# General vars to modify behaviour of the script
DRY_RUN=${DRY_RUN:-true}
DELETE_TEMP_DIR=${DELETE_TEMP_DIR:-true}
REMOVE_UNDEPLOYED=${REMOVE_UNDEPLOYED:-false}

temp_dir=$(mktemp -d)
[[ "$DELETE_TEMP_DIR" == "true" ]] && trap 'rm -rf $temp_dir' EXIT

engine_cmd="$CONTAINER_ENGINE --config=$CONFIG_DIR"

# clone bundle repo containing current version
saas_operator_dir_base="$temp_dir/saas-operator-dir"
bundle_versions_file="$saas_operator_dir_base/$OPERATOR_NAME/${OPERATOR_NAME}-versions.txt"

log "Cloning $BUNDLE_VERSIONS_REPO"
if [[ -n "${APP_SRE_BOT_PUSH_USER:-}" && -n "${APP_SRE_BOT_PUSH_TOKEN:-}" ]]; then
    bundle_versions_repo_url="https://${APP_SRE_BOT_PUSH_USER}:${APP_SRE_BOT_PUSH_TOKEN}@$BUNDLE_VERSIONS_REPO"
else
    bundle_versions_repo_url="https://$BUNDLE_VERSIONS_REPO"
fi

git clone --branch "$BRANCH_CHANNEL" "$bundle_versions_repo_url" "$saas_operator_dir_base"

prev_operator_version=""
removed_versions=""
if [[ -s "$bundle_versions_file" ]]; then
    log "$bundle_versions_file exists. We'll use to determine current version"
    if [[ "$REMOVE_UNDEPLOYED" == "true" ]]; then
        log "Checking if we have to remove any versions more recent than deployed hash"
        # TODO: Move this out of here
        deployed_hash=$(
            curl -s "https://gitlab.cee.redhat.com/service/app-interface/-/raw/master/data/services/deployment-validation-operator/cicd/saas.yaml" | \
                docker run --rm -i quay.io/app-sre/yq -r '.resourceTemplates[]|select(.name="deployment-validation-operator").targets[]|select(.namespace["$ref"]=="/openshift/app-sre-stage-01/namespaces/app-sre-dvo-per-cluster.yml")|.ref'
        )

        log "Current deployed hash is $deployed_hash"

        new_bundle_versions_file=$(mktemp -p "$temp_dir")
        delete=false
        # Sort based on commit number
        for version in $(sort -t . -k 3 -g "$bundle_versions_file"); do
            if [[ "$delete" == false ]]; then
                echo "$version" >> "$new_bundle_versions_file"

                short_hash=$(echo "$version" | cut -d- -f2)

                if [[ "$deployed_hash" == "${short_hash}"* ]]; then
                    log "found deployed hash in the bundle versions repository"
                    delete=true
                fi
            else
                log "Adding $version to removed versions"
                removed_versions="$version $removed_versions"
            fi
        done
        [[ -n "$removed_versions" ]] && log "The following versions will be removed: $removed_versions"
        cp "$new_bundle_versions_file" "$bundle_versions_file"
    fi
    prev_operator_version=$(tail -n 1 "$bundle_versions_file")
else
    log "No $bundle_versions_file exist. This is the first time the operator is built"
fi

if [[ "$OPERATOR_VERSION" == "$prev_operator_version" ]]; then
    log "stopping script as $OPERATOR_VERSION version was already built, so no need to rebuild it"
    exit 0
fi

log "Creating bundle image $BUNDLE_IMAGE:$CURRENT_COMMIT"

mkdir -p "$MANIFEST_DIR"
template=$(mktemp -p "$temp_dir")
./"$BUNDLE_DEPLOY_DIR"/generate-csv-template.py > "$template"
oc process --local -o yaml --raw=true \
    IMAGE="$OPERATOR_IMAGE" \
    IMAGE_TAG="$OPERATOR_IMAGE_TAG" \
    VERSION="$OPERATOR_VERSION" \
    REPLACE_VERSION="$prev_operator_version" \
    -f "$template" > "$CSV"

if [[ "$prev_operator_version" == "" ]]; then \
    sed -i.bak "/replaces/d" "$CSV"
    rm -f "$CSV.bak"
fi

$CONTAINER_ENGINE build -t "$BUNDLE_IMAGE:$CURRENT_COMMIT" "$BUNDLE_DEPLOY_DIR"
$CONTAINER_ENGINE tag "$BUNDLE_IMAGE:$CURRENT_COMMIT" "$BUNDLE_IMAGE:latest"

# opm won't get anything locally, so we need to push the bundle
log "Pushing $BUNDLE_IMAGE:$CURRENT_COMMIT"
$engine_cmd push "$BUNDLE_IMAGE:$CURRENT_COMMIT"

# We need an up to date version of opm executable
opm_local_executable=$(which opm || true)
if [[ "$opm_local_executable" ]]; then
    opm_local_version=$(opm version | sed 's/.*OpmVersion:"//;s/".*//')
fi

if [[ -n "$opm_local_executable" && "$opm_local_version" == "$OPM_VERSION" ]]; then
    log "Using local opm version $opm_local_executable"
else
    log "Downloading opm version $OPM_VERSION"
    curl -s -L "https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/${GOOS}-${GOARCH}-opm" -o "$temp_dir/opm"
    chmod u+x "$temp_dir/opm"
    opm_local_executable="$temp_dir/opm"
fi

from_arg=""
if [[ "$prev_operator_version" ]]; then
    prev_commit=${prev_operator_version#*-}
    from_arg="--from-index $CATALOG_IMAGE:$prev_commit"
fi

log "Creating catalog using opm"
$opm_local_executable index add --bundles "$BUNDLE_IMAGE:$CURRENT_COMMIT" \
                                --tag "$CATALOG_IMAGE:$CURRENT_COMMIT" \
                                --build-tool "$(basename $CONTAINER_ENGINE)" \
                                $from_arg
$CONTAINER_ENGINE tag "$CATALOG_IMAGE:$CURRENT_COMMIT" "$CATALOG_IMAGE:latest"

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

[[ "$removed_versions" ]] && message="$message

removed versions: $removed_versions"

git commit -m "$message"
log "Pushing the repository changes to $BRANCH_CHANNEL in $BUNDLE_VERSIONS_REPO"
[[ "$DRY_RUN" == "false" ]] && git push origin "$BRANCH_CHANNEL"
cd -

log "Pushing $CATALOG_IMAGE:$CURRENT_COMMIT"
[[ "$DRY_RUN" == "false" ]] && $engine_cmd push "$CATALOG_IMAGE:$CURRENT_COMMIT"

log "Pushing $CATALOG_IMAGE:latest"
[[ "$DRY_RUN" == "false" ]] && $engine_cmd push "$CATALOG_IMAGE:latest"

log "Pushing $BUNDLE_IMAGE:latest"
[[ "$DRY_RUN" == "false" ]] && $engine_cmd push "$BUNDLE_IMAGE:latest"
