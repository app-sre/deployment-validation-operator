# Validate variables in project.mk exist
ifndef IMAGE_REGISTRY
$(error IMAGE_REGISTRY is not set; check project.mk file)
endif
ifndef IMAGE_REPOSITORY
$(error IMAGE_REPOSITORY is not set; check project.mk file)
endif
ifndef IMAGE_NAME
$(error IMAGE_NAME is not set; check project.mk file)
endif
ifndef VERSION_MAJOR
$(error VERSION_MAJOR is not set; check project.mk file)
endif
ifndef VERSION_MINOR
$(error VERSION_MINOR is not set; check project.mk file)
endif

# Accommodate docker or podman
CONTAINER_ENGINE=$(shell command -v podman 2>/dev/null || command -v docker 2>/dev/null)

# Generate version and tag information from inputs
COMMIT_NUMBER=$(shell git rev-list `git rev-list --parents HEAD | egrep "^[a-f0-9]{40}$$"`..HEAD --count)
# Standard/App-SRE sha length is 7 instead of 8
CURRENT_COMMIT=$(shell git rev-parse --short=7 HEAD)
OPERATOR_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$(COMMIT_NUMBER)-$(CURRENT_COMMIT)

# upstream-diff - set IMG from two sub-components which allows more commonality in use
OPERATOR_IMAGE=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME)
OPERATOR_IMAGE_TAG=v$(OPERATOR_VERSION)
IMG?=$(OPERATOR_IMAGE):$(OPERATOR_IMAGE_TAG)

OPERATOR_IMAGE_URI=${IMG}
OPERATOR_IMAGE_URI_LATEST=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):latest
OPERATOR_DOCKERFILE ?=build/Dockerfile

BINFILE=build/_output/bin/$(OPERATOR_NAME)
MAINPACKAGE=./cmd/manager

# Containers may default GOFLAGS=-mod=vendor which would break us since
# we're using modules.
unexport GOFLAGS
# upstream-diff - don't set this to Linux, there are Mac OS X in the world
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
# upstream-diff - we want to use vendor
GOENV=GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 GOFLAGS=-mod=vendor

GOBUILDFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

# GOLANGCI_LINT_CACHE needs to be set to a directory which is writeable
# Relevant issue - https://github.com/golangci/golangci-lint/issues/734
GOLANGCI_LINT_CACHE ?= /tmp/golangci-cache

GOLANGCI_LINT_CONFIG ?= ${CONVENTION_DIR}/golangci.yml

TESTTARGETS := $(shell ${GOENV} go list -e ./... | egrep -v "/(vendor)/")
# ex, -v
TESTOPTS :=

ALLOW_DIRTY_CHECKOUT?=false

# TODO: Figure out how to discover this dynamically
CONVENTION_DIR := boilerplate/openshift/golang-osd-operator

default: go-build

.PHONY: clean
clean:
	rm -rf ./build/_output

.PHONY: isclean
isclean:
	@(test "$(ALLOW_DIRTY_CHECKOUT)" != "false" || test 0 -eq $$(git status --porcelain | wc -l)) || (echo "Local git checkout is not clean, commit changes and try again." >&2 && exit 1)

.PHONY: docker-build
docker-build: isclean
	${CONTAINER_ENGINE} build . -f $(OPERATOR_DOCKERFILE) -t $(OPERATOR_IMAGE_URI)
	${CONTAINER_ENGINE} tag $(OPERATOR_IMAGE_URI) $(OPERATOR_IMAGE_URI_LATEST)

.PHONY: docker-push
docker-push: docker-build
	${CONTAINER_ENGINE} push $(OPERATOR_IMAGE_URI)
	${CONTAINER_ENGINE} push $(OPERATOR_IMAGE_URI_LATEST)

.PHONY: push
push: docker-push

.PHONY: go-check
go-check: # golanci-lint and other static analysis
	${CONVENTION_DIR}/ensure.sh golangci-lint
	GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE} golangci-lint run -c ${GOLANGCI_LINT_CONFIG} ./...

.PHONY: go-generate
go-generate:
	${GOENV} go generate $(TESTTARGETS)
	# Don't forget to commit generated files

.PHONY: op-generate
op-generate:
	${CONVENTION_DIR}/operator-sdk-generate.sh
	# Don't forget to commit generated files

.PHONY: generate
generate: op-generate go-generate

# upstream-diff: Force GOOS=linux as we may want to build containers in darwin
.PHONY: go-build
go-build: go-check go-test ## Build binary
	${GOENV} GOOS=linux go build ${GOBUILDFLAGS} -o ${BINFILE} ${MAINPACKAGE}

.PHONY: go-test
go-test:
	${GOENV} go test $(TESTOPTS) $(TESTTARGETS)

.PHONY: python-venv
python-venv:
	${CONVENTION_DIR}/ensure.sh venv ${CONVENTION_DIR}/py-requirements.txt
	$(eval PYTHON := .venv/bin/python3)

.PHONY: yaml-validate
yaml-validate: python-venv
	${PYTHON} ${CONVENTION_DIR}/validate-yaml.py $(shell git ls-files | egrep -v '^(vendor|boilerplate)/' | egrep '.*\.ya?ml')

.PHONY: olm-deploy-yaml-validate
olm-deploy-yaml-validate: python-venv
	${PYTHON} ${CONVENTION_DIR}/validate-yaml.py $(shell git ls-files 'deploy/*.yaml' 'deploy/*.yml')

######################
# Targets used by prow
######################

# validate: Ensure code generation has not been forgotten; and ensure
# generated and boilerplate code has not been modified.
# TODO:
# - isclean; generate; isclean
# - boilerplate/_lib/freeze-check
.PHONY: validate
validate: ;

# lint: Perform static analysis.
.PHONY: lint
lint: olm-deploy-yaml-validate go-check

# test: "Local" unit and functional testing.
.PHONY: test
test: go-test

# coverage: Code coverage analysis and reporting.
.PHONY: coverage
coverage:
	${CONVENTION_DIR}/codecov.sh

# build: Code compilation and bundle generation. This should do as much
# of what app-sre does as possible, so that there are no surprises after
# a PR is merged.
# TODO: Include generating (but not pushing) the bundle
.PHONY: build
build: docker-build

#########################
# Targets used by app-sre
#########################

# build-push: Construct, tag, and push the official operator and
# registry container images.
# TODO: Boilerplate this script.
.PHONY: build-push
build-push:
	hack/app_sre_build_deploy.sh
