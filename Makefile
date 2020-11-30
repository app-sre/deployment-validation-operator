OLM_BUNDLE_VERSIONS_REPO = gitlab.cee.redhat.com/service/saas-operator-versions.git
GOFLAGS_MOD = -mod=vendor
GOLANGCI_OPTIONAL_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
QUAY_USER ?=
QUAY_TOKEN ?=
CONTAINER_ENGINE_CONFIG_DIR = .docker
OLM_CHANNEL ?= alpha

# This include must go below the above definitions
include boilerplate/generated-includes.mk

OLM_BUNDLE_IMAGE = $(OPERATOR_IMAGE)-bundle
OLM_CATALOG_IMAGE = $(OPERATOR_IMAGE)-catalog

OPERATOR_IMAGE_URI_TEST = $(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):test

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: docker-test
docker-test:
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE).test -t $(OPERATOR_IMAGE_URI_TEST)
	${CONTAINER_ENGINE} run -t $(OPERATOR_IMAGE_URI_TEST)

.PHONY: docker-login-and-push
docker-login-and-push: docker-login docker-build
	${CONTAINER_ENGINE} --config=$(CONTAINER_ENGINE_CONFIG_DIR) push $(OPERATOR_IMAGE_URI)
	${CONTAINER_ENGINE} --config=$(CONTAINER_ENGINE_CONFIG_DIR) push $(OPERATOR_IMAGE_URI_LATEST)

.PHONY: docker-login
docker-login:
	@mkdir -p $(CONTAINER_ENGINE_CONFIG_DIR)
	@${CONTAINER_ENGINE} --config=$(CONTAINER_ENGINE_CONFIG_DIR) login -u="${QUAY_USER}" -p="${QUAY_TOKEN}" quay.io

.PHONY: channel-catalog
channel-catalog: docker-login
	OLM_BUNDLE_IMAGE="${OLM_BUNDLE_IMAGE}" \
	OLM_CATALOG_IMAGE="${OLM_CATALOG_IMAGE}" \
	CONTAINER_ENGINE="${CONTAINER_ENGINE}" \
	CONTAINER_ENGINE_CONFIG_DIR="${CONTAINER_ENGINE_CONFIG_DIR}" \
	CURRENT_COMMIT="${CURRENT_COMMIT}" \
	COMMIT_NUMBER="${COMMIT_NUMBER}" \
	OPERATOR_VERSION="${OPERATOR_VERSION}" \
	OPERATOR_NAME="${OPERATOR_NAME}" \
	OPERATOR_IMAGE="${OPERATOR_IMAGE}" \
	OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG}" \
	OLM_BUNDLE_VERSIONS_REPO="${OLM_BUNDLE_VERSIONS_REPO}" \
	OLM_CHANNEL="${OLM_CHANNEL}" \
	./hack/build_channel_catalog.sh
