IMAGE_REGISTRY ?= quay.io
IMAGE_REPOSITORY ?= app-sre
REGISTRY_USER ?= $(QUAY_USER)
REGISTRY_TOKEN ?= $(QUAY_TOKEN)

CONTAINER_ENGINE = $(shell command -v podman 2>/dev/null || echo "docker")
CONTAINER_ENGINE_CONFIG_DIR = .docker

OPERATOR_NAME = deployment-validation-operator
#OPERATOR_IMAGE_TAG ?= copy the catalog image hash
# OPERATOR_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):${OPERATOR_IMAGE_TAG}

# Temporary hardcode for testing, DO NOT MERGE to master
OPERATOR_IMAGE_URI = quay.io/rh_ee_ijimeno/dvo
OPERATOR_IMAGE_TAG ?= dev

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
	@${CONTAINER_ENGINE} login -u="${REGISTRY_USER}" -p="${REGISTRY_TOKEN}" quay.io

.PHONY: docker-build
docker-build:
	@echo "## Building the container image..."
	${CONTAINER_ENGINE} build --pull -f build/Dockerfile -t ${OPERATOR_IMAGE_URI}:${OPERATOR_IMAGE_TAG} .
	${CONTAINER_ENGINE} tag $(OPERATOR_IMAGE_URI):${OPERATOR_IMAGE_TAG} $(OPERATOR_IMAGE_URI):latest

.PHONY: docker-push
docker-push:
	@echo "## Pushing the container image..."
	${CONTAINER_ENGINE} push ${OPERATOR_IMAGE_URI}:${OPERATOR_IMAGE_TAG}
	${CONTAINER_ENGINE} push ${OPERATOR_IMAGE_URI}:latest

## This target is run by build_tag.sh script, triggered by a Jenkins job
.PHONY: docker-publish
docker-publish: quay-login docker-build docker-push
