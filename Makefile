include boilerplate/generated-includes.mk

GOFLAGS_MOD = -mod=vendor
GOLANGCI_OPTIONAL_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
REGISTRY_USER = $(QUAY_USER)
REGISTRY_TOKEN = $(QUAY_TOKEN)
OPERATOR_IMAGE_URI_TEST = $(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):test

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: docker-test
docker-test:
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE).test -t $(OPERATOR_IMAGE_URI_TEST)
	${CONTAINER_ENGINE} run -t $(OPERATOR_IMAGE_URI_TEST)
