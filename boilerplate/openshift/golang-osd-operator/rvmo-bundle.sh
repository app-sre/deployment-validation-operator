#!/usr/bin/env bash

set -e

REPOSITORY=${REPOSITORY:-"git@github.com:openshift/managed-release-bundle.git"}
BRANCH=${BRANCH:-main}
DELETE_TEMP_DIR=${DELETE_TEMP_DIR:-true}
TMPD=$(mktemp -d --suffix -rvmo-bundle)
[[ "${DELETE_TEMP_DIR}" == "true" ]] && trap 'rm -rf ${TMPD}' EXIT

cd "${TMPD}"
git clone --single-branch -b "${BRANCH}" "${REPOSITORY}" .
bash hack/update-operator-release.sh
