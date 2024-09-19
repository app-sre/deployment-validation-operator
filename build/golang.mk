GOLANGCI_OPTIONAL_CONFIG = .golangci.yml
GOLANGCI_LINT_CACHE =/tmp/golangci-cache
.PHONY: go-lint
go-lint:
	@echo "## Running the golangci-lint tool..."
	jenkins/get_dependencies.sh
	GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE} golangci-lint run -c ${GOLANGCI_OPTIONAL_CONFIG} ./...

TEST_TARGETS = $(shell ${GOENV} go list -e ./... | grep -E -v "/(vendor)/")
.PHONY: go-test
go-test:
	@echo "## Running the code unit tests..."
	${GOENV} go test ${TEST_TARGETS}

GOOS ?= linux
GOENV=GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=1
GOBUILDFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"
.PHONY: go-build
go-build:
	@echo "## Building the binary..."
	go mod vendor
	${GOENV} go build ${GOBUILDFLAGS} -o build/_output/bin/$(OPERATOR_NAME) .
