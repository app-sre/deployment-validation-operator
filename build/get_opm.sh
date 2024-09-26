#!/usr/bin/env bash
set -x
set -eo pipefail

GOOS=$(go env GOOS)
OPM_VERSION="v1.23.2"

if which opm ; then
    exit
fi

mkdir -p .opm/bin
cd .opm/bin

DOWNLOAD_URL="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/$GOOS-amd64-opm"
curl -sfL "${DOWNLOAD_URL}" -o opm
chmod +x opm
