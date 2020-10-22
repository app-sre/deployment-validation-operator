.EXPORT_ALL_VARIABLES:

BUNDLE_VERSIONS_REPO = gitlab.cee.redhat.com/service/saas-deployment-validation-operator-bundle.git
GOFLAGS_MOD = -mod=vendor
GOLANGCI_LINT_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
QUAY_USER ?=
QUAY_TOKEN ?=
OPM_VERSION = v1.14.0

BUNDLE_DEPLOY_DIR = deploy/bundle
MANIFEST_DIR = $(BUNDLE_DEPLOY_DIR)/manifests
CSV = $(MANIFEST_DIR)/deploymentvalidationoperator.clusterserviceversion.yaml
CONFIG_DIR = .docker

# This include must go below the above definitions
include boilerplate/generated-includes.mk

BUNDLE_IMAGE ?= $(OPERATOR_IMAGE)-bundle
CATALOG_IMAGE ?= $(OPERATOR_IMAGE)-catalog

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
	@./hack/build_channel_catalog.sh
