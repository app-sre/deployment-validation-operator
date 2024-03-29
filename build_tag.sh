#!/bin/bash

make \
OPERATOR_IMAGE_TAG=$(echo $GIT_BRANCH|cut -d"/" -f3) \
IMAGE_REPOSITORY="deployment-validation-operator" \
IMAGE_NAME="dv-operator" \
REGISTRY_USER=$(echo $QUAY_USER) \
REGISTRY_TOKEN=$(echo $QUAY_TOKEN) \
docker-push
