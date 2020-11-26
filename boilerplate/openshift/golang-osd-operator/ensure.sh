#!/bin/bash
set -x
set -eo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source $REPO_ROOT/boilerplate/_lib/common.sh

GOLANGCI_LINT_VERSION="1.30.0"
OPM_VERSION="v1.15.2"
GRPCURL_VERSION="1.7.0"
DEPENDENCY=${1:-}
GOOS=$(go env GOOS)

case "${DEPENDENCY}" in

golangci-lint)
    GOPATH=$(go env GOPATH)
    if which golangci-lint ; then
        exit
    else
        mkdir -p "${GOPATH}/bin"
        echo "${PATH}" | grep -q "${GOPATH}/bin"
        IN_PATH=$?
        if [ $IN_PATH != 0 ]; then
            echo "${GOPATH}/bin not in $$PATH"
            exit 1
        fi
        DOWNLOAD_URL="https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-${GOOS}-amd64.tar.gz"
        curl -sfL "${DOWNLOAD_URL}" | tar -C "${GOPATH}/bin" -zx --strip-components=1 "golangci-lint-${GOLANGCI_LINT_VERSION}-${GOOS}-amd64/golangci-lint"
    fi
    ;;

operator-sdk)
    #########################################################
    # Ensure operator-sdk is installed at the desired version
    # When done, ./.operator-sdk/bin/operator-sdk will be a
    # symlink to the appropriate executable.
    #########################################################
    # First discover the desired version from go.mod
    # The following properly takes `replace` directives into account.
    wantver=$(go list -json -m github.com/operator-framework/operator-sdk | jq -r 'if .Replace != null then .Replace.Version else .Version end')
    echo "go.mod says you want operator-sdk $wantver"
    # Where we'll put our (binary and) symlink
    mkdir -p .operator-sdk/bin
    cd .operator-sdk/bin
    # Discover existing, giving preference to one already installed in
    # this path, because that has a higher probability of being the
    # right one.
    if [[ -x ./operator-sdk ]] && [[ "$(osdk_version ./operator-sdk)" == "$wantver" ]]; then
        echo "operator-sdk $wantver already installed"
        exit 0
    fi
    # Is there one in $PATH?
    if which operator-sdk && [[ $(osdk_version $(which operator-sdk)) == "$wantver" ]]; then
        osdk=$(realpath $(which operator-sdk))
        echo "Found at $osdk"
    else
        case "$(uname -s)" in
            Linux*)
                binary="operator-sdk-${wantver}-x86_64-linux-gnu"
                ;;
            Darwin*)
                binary="operator-sdk-${wantver}-x86_64-apple-darwin"
                ;;
            *)
                echo "OS unsupported"
                exit 1
                ;;
        esac
        # The boilerplate backing image sets up binaries with the full
        # name in /usr/local/bin, so look for the right one of those
        if which $binary; then
            osdk=$(realpath $(which $binary))
        else
            echo "Downloading $binary"
            curl -OJL https://github.com/operator-framework/operator-sdk/releases/download/${wantver}/${binary}
            chmod +x ${binary}
            osdk=${binary}
        fi
    fi
    # Create (or overwrite) the symlink to the binary we discovered or
    # downloaded above.
    ln -sf $osdk operator-sdk
    ;;

opm)
    mkdir -p .opm/bin
    cd .opm/bin

    if [[ -x ./opm  && "$(opm_version ./opm)" == "$OPM_VERSION" ]]; then
        exit 0
    fi

    if which opm && [[ "$(opm_version $(which opm))" == "$OPM_VERSION" ]]; then
        opm=$(realpath $(which opm))
    else
        opm="opm-$OPM_VERSION-$GOOS-amd64"
        opm_download_url="https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/$GOOS-amd64-opm"
        curl -sfL "${opm_download_url}" -o "$opm"
        chmod +x "$opm"
    fi

    ln -fs "$opm" opm
    ;;

grpcurl)
    mkdir -p .grpcurl/bin
    cd .grpcurl/bin

    if [[ -x ./grpcurl  && "$(grpcurl_version ./grpcurl)" == "$GRPCURL_VERSION" ]]; then
        exit 0
    fi

    if which grpcurl && [[ "$(grpcurl_version $(which grpcurl))" == "$GRPCURL_VERSION" ]]; then
        grpcurl=$(realpath $(which grpcurl))
    else
        # mapping from https://github.com/fullstorydev/grpcurl/blob/master/.goreleaser.yml
        [[ "$GOOS" == "darwin" ]] && os=osx || os="$GOOS"
        grpcurl="grpcurl-$GRPCURL_VERSION-$os-x86_64"
        grpcurl_download_url="https://github.com/fullstorydev/grpcurl/releases/download/v$GRPCURL_VERSION/grpcurl_${GRPCURL_VERSION}_${os}_x86_64.tar.gz"
        curl -sfL "$grpcurl_download_url" | tar -xzf - -O grpcurl > "$grpcurl"
        chmod +x "$grpcurl"
    fi

    ln -fs "$grpcurl" grpcurl
    ;;

venv)
    # Set up a python virtual environment
    python3 -m venv .venv
    # Install required libs, if a requirements file was given
    if [[ -n "$2" ]]; then
        .venv/bin/python3 -m pip install -r "$2"
    fi
    ;;

*)
    echo "Unknown dependency: ${DEPENDENCY}"
    exit 1
    ;;
esac
