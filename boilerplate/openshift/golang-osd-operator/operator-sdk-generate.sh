#!/bin/bash
set -eo pipefail

###
# Run operator-sdk generate commands appropriate to the version of
# operator-sdk configured in the consuming repository.
###

REPO_ROOT=$(git rev-parse --show-toplevel)

source $REPO_ROOT/boilerplate/_lib/common.sh

$HERE/ensure.sh operator-sdk

# Symlink to operator-sdk binary set up by `ensure.sh operator-sdk`:
OSDK=$REPO_ROOT/.operator-sdk/bin/operator-sdk

VER=$(osdk_version $OSDK)

# This explicitly lists the versions we know about. We don't support
# anything outside of that.
case $VER in
  'v0.15.1'|'v0.16.0'|'v0.17.0'|'v0.17.1')
      $OSDK generate crds
      $OSDK generate k8s
      ;;
  *) err "Unsupported operator-sdk version $VER" ;;
esac
