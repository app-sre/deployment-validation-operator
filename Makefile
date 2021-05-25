GOFLAGS_MOD = -mod=vendor
GOLANGCI_OPTIONAL_CONFIG = .golangci.yml
IMAGE_REPOSITORY ?= app-sre
REGISTRY_USER = $(QUAY_USER)
REGISTRY_TOKEN = $(QUAY_TOKEN)

# This include must go below the above definitions
include boilerplate/generated-includes.mk

OPERATOR_IMAGE_URI_TEST = $(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):test

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update

.PHONY: docker-test
docker-test:
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE).test -t $(OPERATOR_IMAGE_URI_TEST)
	${CONTAINER_ENGINE} run -t $(OPERATOR_IMAGE_URI_TEST)

# We are early adopters of the OPM build/push process. Remove this
# override once boilerplate uses that path by default.
build-push: opm-build-push ;
