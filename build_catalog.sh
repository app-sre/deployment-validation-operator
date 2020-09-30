#!/bin/bash

count=0
for var in BUNDLE_IMAGE_NAME \
           CATALOG_IMAGE_NAME \
           QUAY_USER \
           QUAY_TOKEN
do
  if [ ! "${!var}" ]; then
    echo "$var is not set"
    count=$((count + 1))
  fi
done

[ $count -gt 0 ] && exit 1

num_commits=$(git rev-list $(git rev-list --max-parents=0 HEAD)..HEAD --count)
current_commit=$(git rev-parse --short=7 HEAD)
version="0.1.$num_commits-$current_commit"
opm_version="1.14.0"

# Login to docker
docker_conf="$PWD/.docker"
mkdir -p "$docker_conf"
docker_cmd="docker --config=$docker_conf"

$docker_cmd login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

# Find the CSV version from the previous bundle
echo "pulling laster bundle image"
$docker_cmd pull $BUNDLE_IMAGE_NAME:latest && exists=1 || exists=0

if [ $exists -eq 1 ]; then
  echo "extracting previous version from bundle image"
  docker create --name="tmp_$$" $BUNDLE_IMAGE_NAME:latest sh
  tmp_dir=$(mktemp -d -t sa-XXXXXXXXXX)
  pushd $tmp_dir
    docker export tmp_$$ | tar -xf -
    prev_version=`find . -name *.clusterserviceversion.* | xargs cat - | python3 -c 'import sys,yaml; print(yaml.safe_load(sys.stdin.read())["metadata"]["name"])'`
    echo $prev_version
    if [[ "$prev_version" == "" ]]; then
      exit 1
    fi
  popd
  rm -rf $tmp_dir
  docker rm tmp_$$
fi

# Build/push the new bundle
pushd deploy/bundle
  echo "creating the bundle"
  if [[ $prev_version != "" ]]; then
    export REPLACE_VERSION=$prev_version
  fi
  export IMAGE=$BUNDLE_IMAGE_NAME
  export IMAGE_TAG=$current_commit
  export VERSION=$version
  make bundle

  echo "pushing the bundle to repository"
  docker tag $BUNDLE_IMAGE_NAME:$current_commit $BUNDLE_IMAGE_NAME:latest
  $docker_cmd push $BUNDLE_IMAGE_NAME:$current_commit
  $docker_cmd push $BUNDLE_IMAGE_NAME:latest
popd

# Download opm build
curl -L https://github.com/operator-framework/operator-registry/releases/download/v$opm_version/linux-amd64-opm -o ./opm
chmod u+x ./opm

# Create/push a new catalog via opm
echo "pulling existing latest catalog"
$docker_cmd pull $CATALOG_IMAGE_NAME:latest && exists=1 || exists=0
if [ $exists -eq 1 ]; then
  from_arg="--from-index $CATALOG_IMAGE_NAME:latest"
fi

echo "creating new catalog"
./opm index add --bundles $BUNDLE_IMAGE_NAME:$current_commit $from_arg --tag $CATALOG_IMAGE_NAME:$current_commit --build-tool docker

echo "pushing catalog to repository"
docker tag $CATALOG_IMAGE_NAME:$current_commit $CATALOG_IMAGE_NAME:latest
$docker_cmd push $CATALOG_IMAGE_NAME:$current_commit
$docker_cmd push $CATALOG_IMAGE_NAME:latest
