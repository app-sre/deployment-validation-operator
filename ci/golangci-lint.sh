#!/bin/bash

set -ex

if [ -z "${GOPATH:-}" ]; then
    eval "$(go env | grep GOPATH)"
fi

export BINARY=bin/golangci-lint
if [ ! -f "$BINARY" ]; then
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.64.7
fi
