#!/usr/bin/env bash
set -x
set -eo pipefail

GOOS=$(go env GOOS)
OPM_VERSION="v1.23.2"
GRPCURL_VERSION="1.7.0"

if ! which opm ; then
    mkdir -p .opm/bin
    cd .opm/bin

    DOWNLOAD_URL="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/$GOOS-amd64-opm"
    curl -sfL "${DOWNLOAD_URL}" -o opm
    chmod +x opm

    cd -
    ln -s .opm/bin/opm
fi

if ! which grpcurl; then
    mkdir -p .grpcurl/bin
    cd .grpcurl/bin

    DOWNLOAD_URL="https://github.com/fullstorydev/grpcurl/releases/download/v$GRPCURL_VERSION/grpcurl_${GRPCURL_VERSION}_${GOOS}_x86_64.tar.gz"
    curl -sfL "$DOWNLOAD_URL" | tar -xzf - -O grpcurl > grpcurl
    chmod +x grpcurl

    cd -
    ln -s .grpcurl/bin/grpcurl
fi

## TODO: This will probably require rework with the path to the executables, locally required some workaround. Revisit after running in the Job