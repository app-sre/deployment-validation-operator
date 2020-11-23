BUNDLE_VERSIONS_REPO = gitlab.cee.redhat.com/service/saas-deployment-validation-operator-bundle.git
GOFLAGS_MOD = -mod=vendor
GOLANGCI_OPTIONAL_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
QUAY_USER ?=
QUAY_TOKEN ?=
OPM_VERSION = v1.15.2
GRPCURL_VERSION = 1.7.0
CONFIG_DIR = .docker

# This include must go below the above definitions
include boilerplate/generated-includes.mk

BUNDLE_IMAGE = $(OPERATOR_IMAGE)-bundle
CATALOG_IMAGE = $(OPERATOR_IMAGE)-catalog

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
	${CONTAINER_ENGINE} --config=$(CONFIG_DIR) push $(OPERATOR_IMAGE_URI)
	${CONTAINER_ENGINE} --config=$(CONFIG_DIR) push $(OPERATOR_IMAGE_URI_LATEST)

.PHONY: docker-login
docker-login:
	@mkdir -p $(CONFIG_DIR)
	@${CONTAINER_ENGINE} --config=$(CONFIG_DIR) login -u="${QUAY_USER}" -p="${QUAY_TOKEN}" quay.io

.PHONY: catalog
channel-catalog: docker-login
	BUNDLE_IMAGE="${BUNDLE_IMAGE}" \
	CATALOG_IMAGE="${CATALOG_IMAGE}" \
	CONTAINER_ENGINE="${CONTAINER_ENGINE}" \
	CONFIG_DIR="${CONFIG_DIR}" \
	CURRENT_COMMIT="${CURRENT_COMMIT}" \
	COMMIT_NUMBER="${COMMIT_NUMBER}" \
	OPERATOR_VERSION="${OPERATOR_VERSION}" \
	OPERATOR_NAME="${OPERATOR_NAME}" \
	OPERATOR_IMAGE="${OPERATOR_IMAGE}" \
	OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG}" \
	GOOS="${GOOS}" \
	GOARCH="${GOARCH}" \
	OPM_VERSION="${OPM_VERSION}" \
	GRPCURL_VERSION="${GRPCURL_VERSION}" \
	BUNDLE_VERSIONS_REPO="${BUNDLE_VERSIONS_REPO}" \
	./hack/build_channel_catalog.sh
