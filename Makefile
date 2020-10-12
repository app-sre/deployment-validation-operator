.EXPORT_ALL_VARIABLES:

GOLANGCI_LINT_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
QUAY_USER ?=
QUAY_TOKEN ?=
OPM_VERSION = v1.14.0

BUNDLE_DEPLOY_DIR=deploy/bundle
MANIFEST_DIR=$(BUNDLE_DEPLOY_DIR)/manifests
CSV=$(MANIFEST_DIR)/deploymentvalidationoperator.clusterserviceversion.yaml
CONFIG_DIR=.docker

# This include must go below the above definitions
include boilerplate/generated-includes.mk

BUNDLE_IMAGE?=$(OPERATOR_IMAGE)-bundle
CATALOG_IMAGE?=$(OPERATOR_IMAGE)-catalog

OPERATOR_IMAGE_URI_TEST=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):test

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: docker-test
docker-test:
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE).test -t $(OPERATOR_IMAGE_URI_TEST)
	${CONTAINER_ENGINE} run -t $(OPERATOR_IMAGE_URI_TEST)

.PHONY: docker-login-and-push
docker-login-and-push: docker-login docker-build
	${CONTAINER_ENGINE} --config=$(CONFIG_DIR) push $(OPERATOR_IMAGE_URI); \
	${CONTAINER_ENGINE} --config=$(CONFIG_DIR) push $(OPERATOR_IMAGE_URI_LATEST)

.PHONY: docker-login
docker-login:
	mkdir -p $(CONFIG_DIR)
	${CONTAINER_ENGINE} --config=$(CONFIG_DIR) login -u="${QUAY_USER}" -p="${QUAY_TOKEN}" quay.io;

.PHONY: manifest
manifest:
	@mkdir -p $(MANIFEST_DIR) ; \
	TEMPLATE=`mktemp` ; \
	./$(BUNDLE_DEPLOY_DIR)/generate-csv-template.py > $${TEMPLATE} ; \
	oc process --local -o yaml --raw=true IMAGE=$(OPERATOR_IMAGE) IMAGE_TAG=$(OPERATOR_IMAGE_TAG) VERSION=$(OPERATOR_VERSION) REPLACE_VERSION=$(REPLACE_VERSION) -f $${TEMPLATE} > $(CSV)
	@if [ "${REPLACE_VERSION}" == "" ]; then \
	  sed -i.bak "/replaces/d" $(CSV); \
	  rm -f $(CSV).bak; \
	fi

.PHONY: bundle
bundle: manifest
	docker build -t ${BUNDLE_IMAGE}:$(CURRENT_COMMIT) $(BUNDLE_DEPLOY_DIR)

.PHONY: catalog
catalog: docker-login bundle
	./create_catalog.sh

.PHONY: clean
clean:
	rm -rf $(MANIFEST_DIR) $(CONFIG_DIR)
