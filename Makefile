OPERATOR_NAME = deployment-validation-operator
# Image repository vars
REGISTRY_USER ?= ${QUAY_USER}
REGISTRY_TOKEN ?= ${QUAY_TOKEN}
IMAGE_REGISTRY ?= quay.io
IMAGE_REPOSITORY ?= app-sre
IMAGE_NAME ?= ${OPERATOR_NAME}
OPERATOR_IMAGE = ${IMAGE_REGISTRY}/${IMAGE_REPOSITORY}/${IMAGE_NAME}

OLM_CHANNEL ?= alpha
OLM_BUNDLE_IMAGE = ${OPERATOR_IMAGE}-bundle
OLM_CATALOG_IMAGE = ${OPERATOR_IMAGE}-catalog

VERSION_MAJOR ?= 0
VERSION_MINOR ?= 1
COMMIT_COUNT=$(shell git rev-list --count HEAD)
CURRENT_COMMIT=$(shell git rev-parse --short=7 HEAD)
OPERATOR_VERSION=${VERSION_MAJOR}.${VERSION_MINOR}.${COMMIT_COUNT}-g${CURRENT_COMMIT}
OPERATOR_IMAGE_TAG ?= ${OPERATOR_VERSION}

CONTAINER_ENGINE_CONFIG_DIR = .docker
CONTAINER_ENGINE = $(shell command -v podman 2>/dev/null || echo docker --config=$(CONTAINER_ENGINE_CONFIG_DIR))

ifdef FIPS_ENABLED
FIPSENV=GOEXPERIMENT=strictfipsruntime GOFLAGS="-tags=strictfipsruntime"
endif

.PHONY: go-mod-update
go-mod-update:
	go mod vendor

GOOS ?= linux
GOENV=GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 ${FIPSENV}
GOBUILDFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"
.PHONY: go-build
go-build: go-mod-update
	@echo "## Building the binary..."
	${GOENV} go build ${GOBUILDFLAGS} -o build/_output/bin/$(OPERATOR_NAME) .

## Used by CI pipeline ci/prow/lint
GOLANGCI_OPTIONAL_CONFIG = .golangci.yml
GOLANGCI_LINT_CACHE =/tmp/golangci-cache
.PHONY: lint
lint: go-mod-update
	@echo "## Running the golangci-lint tool..."
	GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE} golangci-lint run -c ${GOLANGCI_OPTIONAL_CONFIG} ./...

## Used by CI pipeline ci/prow/test
TEST_TARGETS = $(shell ${GOENV} go list -e ./... | grep -E -v "/(vendor)/")
.PHONY: test
test: go-mod-update
	@echo "## Running the code unit tests..."
	${GOENV} go test ${TEST_TARGETS}

## These targets: coverage and test-coverage; are used by the CI pipeline ci/prow/coverage
.PHONY: coverage
coverage:
	@echo "## Running code coverage..."
	ci/codecov.sh

TESTOPTS :=
.PHONY: test-coverage
test-coverage: go-mod-update
	@echo "## Running the code unit tests with coverage..."
	${GOENV} go test ${TESTOPTS} ${TEST_TARGETS}

## Used by CI pipeline ci/prow/validate
.PHONY: validate
validate:
	@echo "## Perform validation that the folder does not contain extra artifacts..."
	test 0 -eq $$(git status --porcelain | wc -l) || (echo "Base folder contains unknown artifacts" >&2 && git --no-pager diff && exit 1)

.PHONY: quay-login
quay-login:
	@echo "## Login to quay.io..."
	mkdir -p ${CONTAINER_ENGINE_CONFIG_DIR}
	export REGISTRY_AUTH_FILE=${CONTAINER_ENGINE_CONFIG_DIR}/config.json
	@${CONTAINER_ENGINE} login -u="${REGISTRY_USER}" -p="${REGISTRY_TOKEN}" quay.io

.PHONY: docker-build
docker-build:
	@echo "## Building the container image..."
	${CONTAINER_ENGINE} build --pull -f build/Dockerfile -t ${OPERATOR_IMAGE}:${OPERATOR_IMAGE_TAG} .
	${CONTAINER_ENGINE} tag ${OPERATOR_IMAGE}:${OPERATOR_IMAGE_TAG} ${OPERATOR_IMAGE}:latest

.PHONY: docker-push
docker-push:
	@echo "## Pushing the container image..."
	${CONTAINER_ENGINE} push ${OPERATOR_IMAGE}:${OPERATOR_IMAGE_TAG}
	${CONTAINER_ENGINE} push ${OPERATOR_IMAGE}:latest

## This target is run by build_tag.sh script, triggered by a Jenkins job
.PHONY: docker-publish
docker-publish: quay-login docker-build docker-push

## This target is run by the master branch Jenkins Job
.PHONY: build-push
build-push: docker-publish
	@echo "## Building bundle and catalog images..."
	@(CONTAINER_ENGINE="${CONTAINER_ENGINE}" \
	CONTAINER_ENGINE_CONFIG_DIR="${CONTAINER_ENGINE_CONFIG_DIR}" \
	CURRENT_COMMIT="${CURRENT_COMMIT}" \
	OLM_BUNDLE_IMAGE="${OLM_BUNDLE_IMAGE}" \
	OLM_CATALOG_IMAGE="${OLM_CATALOG_IMAGE}" \
	OLM_CHANNEL="${OLM_CHANNEL}" \
	OPERATOR_NAME="${OPERATOR_NAME}" \
	OPERATOR_VERSION="${OPERATOR_VERSION}" \
	OPERATOR_IMAGE="${OPERATOR_IMAGE}" \
	OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG}" \
	IMAGE_REGISTRY=${IMAGE_REGISTRY} \
	REGISTRY_USER="${REGISTRY_USER}" \
	REGISTRY_TOKEN="${REGISTRY_TOKEN}" \
		build/build_opm_catalog.sh)