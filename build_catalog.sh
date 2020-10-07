#!/bin/bash
set -e

function log ()
{
  echo "######## $1 ########"
}

count=0
for var in BUNDLE_IMAGE \
           CATALOG_IMAGE \
           QUAY_USER \
           QUAY_TOKEN \
           CONTAINER_ENGINE \
           CONFIG_DIR
do
  if [ ! "${!var}" ]; then
    log "$var is not set"
    count=$((count + 1))
  fi
done

[ $count -gt 0 ] && exit 1

engine_cmd="${CONTAINER_ENGINE} --config=${CONFIG_DIR}"

# Find the CSV version from the previous bundle
log "Pulling latest tag for bundle $BUNDLE_IMAGE"
$engine_cmd pull $BUNDLE_IMAGE:latest && exists=1 || exists=0

if [ $exists -eq 1 ]; then
  log "Extracting previous version from bundle image"
  ${CONTAINER_ENGINE} create --name="tmp_$$" $BUNDLE_IMAGE:latest sh
  tmp_dir=$(mktemp -d -t sa-XXXXXXXXXX)
  pushd $tmp_dir
    ${CONTAINER_ENGINE} export tmp_$$ | tar -xf -
    prev_version=`find . -name *.clusterserviceversion.* | xargs cat - | python3 -c 'import sys,yaml; print(yaml.safe_load(sys.stdin.read())["metadata"]["name"])'`
    if [[ "$prev_version" == "" ]]; then
      log "Unable to find previous bundle version"
      exit 1
    fi
    log "Found previous bundle version $prev_version"
  popd
  rm -rf $tmp_dir
  ${CONTAINER_ENGINE} rm tmp_$$
fi

# Build/push the new bundle
pushd deploy/bundle
  log "Creating bundle $BUNDLE_IMAGE:$CURRENT_COMMIT"
  if [[ $prev_version != "" ]]; then
    export REPLACE_VERSION=$prev_version
  fi
  export BUNDLE_IMAGE_TAG=$CURRENT_COMMIT
#  export OPERATOR_IMAGE_TAG=v$version
  export VERSION=$OPERATOR_VERSION
  make bundle
  ${CONTAINER_ENGINE} tag $BUNDLE_IMAGE:$CURRENT_COMMIT $BUNDLE_IMAGE:latest

  log "Pushing the bundle $BUNDLE_IMAGE:$CURRENT_COMMIT to repository"
  $engine_cmd push $BUNDLE_IMAGE:$CURRENT_COMMIT
  # Do not push the latest tag here.  If there is a problem creating the catalog then
  # pushing the latest tag here will mean subsequent runs will be extracting a bundle
  # version that isn't referenced in the catalog.  This will result in all future
  # catalog creation failing to be created.
popd

# Download opm build
curl -L https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/${GOOS}-${GOARCH}-opm -o ./opm
chmod u+x ./opm

# Create/push a new catalog via opm
log "Pulling latest tag for catalog $CATALOG_IMAGE"
$engine_cmd pull $CATALOG_IMAGE:latest && exists=1 || exists=0
if [ $exists -eq 1 ]; then
  from_arg="--from-index $CATALOG_IMAGE:latest"
fi

if [[ "$from_arg" == "" ]]; then
  log "Creating new catalog $CATALOG_IMAGE"
else
  log "Updating existing catalog $CATALOG_IMAGE"
fi

./opm index add --bundles $BUNDLE_IMAGE:$CURRENT_COMMIT $from_arg --tag $CATALOG_IMAGE:$CURRENT_COMMIT --build-tool ${CONTAINER_ENGINE}
if [ $? -ne 0 ]; then
  exit 1
fi
${CONTAINER_ENGINE} tag $CATALOG_IMAGE:$CURRENT_COMMIT $CATALOG_IMAGE:latest

log "Pushing catalog $CATALOG_IMAGE:$CURRENT_COMMIT to repository"
$engine_cmd push $CATALOG_IMAGE:$CURRENT_COMMIT

# Only put the latest tags once everything else has succeeded
log "Pushing latest tags for $BUNDLE_IMAGE and $CATALOG_IMAGE"
$engine_cmd push $CATALOG_IMAGE:latest
$engine_cmd push $BUNDLE_IMAGE:latest
