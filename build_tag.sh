#!/bin/bash

## This script is the entry point for the Jenkins job: deployment-validation-operator build tag
## It builds a new image and pushes it to the dv-operator repository on quay.io.
make \
OPERATOR_IMAGE_TAG=$(echo $GIT_BRANCH|cut -d"/" -f3) \
IMAGE_REPOSITORY="deployment-validation-operator" \
IMAGE_NAME="dv-operator" \
REGISTRY_USER=$(echo $QUAY_USER) \
REGISTRY_TOKEN=$(echo $QUAY_TOKEN) \
docker-publish
