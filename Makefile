include boilerplate/generated-includes.mk

GOLANGCI_LINT_CONFIG = .golangci.yml
IMAGE_REGISTRY ?= quay.io
IMAGE_REPOSITORY ?= deployment-validation-operator
IMAGE_NAME ?= ${OPERATOR_NAME}

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
