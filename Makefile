GOFLAGS_MOD = -mod=vendor
GOLANGCI_LINT_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
QUAY_USER ?=
QUAY_TOKEN ?=

# This include must go below the above definitions
include boilerplate/generated-includes.mk

OPERATOR_IMAGE_URI_TEST=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):test

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: docker-test
docker-test:
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE).test -t $(OPERATOR_IMAGE_URI_TEST)
	${CONTAINER_ENGINE} run -t $(OPERATOR_IMAGE_URI_TEST)

.PHONY: docker-login-and-push
docker-login-and-push: docker-build
	@CONFIG_DIR=`mktemp -d`; \
	${CONTAINER_ENGINE} --config="$${CONFIG_DIR}" login -u="${QUAY_USER}" -p="${QUAY_TOKEN}" quay.io; \
	${CONTAINER_ENGINE} --config="$${CONFIG_DIR}" push $(OPERATOR_IMAGE_URI); \
	${CONTAINER_ENGINE} --config="$${CONFIG_DIR}" push $(OPERATOR_IMAGE_URI_LATEST)
