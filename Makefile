OPERATOR_NAME = deployment-validation-operator
# Image repository vars
## Overwritten for testing REGISTRY_USER ?= ${QUAY_USER}
ALT_REGISTRY_USER = rh_ee_ijimeno+dvojenkins01
## Overwritten for testing REGISTRY_TOKEN ?= ${QUAY_TOKEN}
ALT_REGISTRY_TOKEN = 61BOGU7XW2AKL15TI3UR56YPX7BG73TUGYYBLPQ55POR70J0L5KR4J15SEH108DG
IMAGE_REGISTRY ?= quay.io
## Overwritten for testing IMAGE_REPOSITORY ?= app-sre
IMAGE_REPOSITORY ?= rh_ee_ijimeno
## Overwritten for testing IMAGE_NAME ?= ${OPERATOR_NAME}
IMAGE_NAME ?= dvo
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

# This include must go below the above definitions
# include boilerplate/generated-includes.mk
include build/golang.mk

OPERATOR_IMAGE_URI_TEST = $(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):test

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: docker-test
docker-test:
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE).test -t $(OPERATOR_IMAGE_URI_TEST)
	${CONTAINER_ENGINE} run -t $(OPERATOR_IMAGE_URI_TEST)

.PHONY: e2e-test
e2e-test:
	ginkgo run --tags e2e test/e2e/

# We are early adopters of the OPM build/push process. Remove this
# override once boilerplate uses that path by default.
build-push: opm-build-push ;

.PHONY: quay-login
quay-login:
	@echo "## Login to quay.io..."
	mkdir -p ${CONTAINER_ENGINE_CONFIG_DIR}
	export REGISTRY_AUTH_FILE=${CONTAINER_ENGINE_CONFIG_DIR}/config.json
	@${CONTAINER_ENGINE} login -u="${ALT_REGISTRY_USER}" -p="${ALT_REGISTRY_TOKEN}" quay.io

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

# tbd : quay-login -> docker-publish
.PHONY: test_opm
test_opm: docker-publish
	CONTAINER_ENGINE="${CONTAINER_ENGINE}" \
	CONTAINER_ENGINE_CONFIG_DIR="${CONTAINER_ENGINE_CONFIG_DIR}" \
	CURRENT_COMMIT="${CURRENT_COMMIT}" \
	OLM_BUNDLE_IMAGE="${OLM_BUNDLE_IMAGE}" \
	OLM_CATALOG_IMAGE="${OLM_CATALOG_IMAGE}" \
	OLM_CHANNEL="${OLM_CHANNEL}" \
	OPERATOR_NAME="${OPERATOR_NAME}" \
	OPERATOR_VERSION="${OPERATOR_VERSION}" \
	OPERATOR_IMAGE="${OPERATOR_IMAGE}" \
	OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG}" \
		build/build_opm_catalog.sh