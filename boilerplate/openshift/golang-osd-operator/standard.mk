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

### Accommodate docker or podman
#
# The docker/podman creds cache needs to be in a location unique to this
# invocation; otherwise it could collide across jenkins jobs. We'll use
# a .docker folder relative to pwd (the repo root).
CONTAINER_ENGINE_CONFIG_DIR = .docker
# But docker and podman use different options to configure it :eyeroll:
# ==> Podman uses --authfile=PATH *after* the `login` subcommand; but
# also accepts REGISTRY_AUTH_FILE from the env. See
# https://www.mankier.com/1/podman-login#Options---authfile=path
export REGISTRY_AUTH_FILE = ${CONTAINER_ENGINE_CONFIG_DIR}/config.json
# If this configuration file doesn't exist, podman will error out. So
# we'll create it if it doesn't exist.
ifeq (,$(wildcard $(REGISTRY_AUTH_FILE)))
$(shell mkdir -p $(CONTAINER_ENGINE_CONFIG_DIR))
$(shell echo '{}' > $(REGISTRY_AUTH_FILE))
endif
# ==> Docker uses --config=PATH *before* (any) subcommand; so we'll glue
# that to the CONTAINER_ENGINE variable itself. (NOTE: I tried half a
# dozen other ways to do this. This was the least ugly one that actually
# works.)
ifndef CONTAINER_ENGINE
CONTAINER_ENGINE=$(shell command -v podman 2>/dev/null || echo docker --config=$(CONTAINER_ENGINE_CONFIG_DIR))
endif

# Generate version and tag information from inputs
COMMIT_NUMBER=$(shell git rev-list `git rev-list --parents HEAD | grep -E "^[a-f0-9]{40}$$"`..HEAD --count)
CURRENT_COMMIT=$(shell git rev-parse --short=7 HEAD)
OPERATOR_VERSION=$(VERSION_MAJOR).$(VERSION_MINOR).$(COMMIT_NUMBER)-$(CURRENT_COMMIT)

OPERATOR_IMAGE=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME)
OPERATOR_IMAGE_TAG=v$(OPERATOR_VERSION)
IMG?=$(OPERATOR_IMAGE):$(OPERATOR_IMAGE_TAG)
OPERATOR_IMAGE_URI=${IMG}
OPERATOR_IMAGE_URI_LATEST=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME):latest
OPERATOR_DOCKERFILE ?=build/Dockerfile
REGISTRY_IMAGE=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(IMAGE_NAME)-registry

ifeq ($(SUPPLEMENTARY_IMAGE_NAME),)
# We need SUPPLEMENTARY_IMAGE to be defined for csv-generate.mk
SUPPLEMENTARY_IMAGE=""
else
# If the configuration specifies a SUPPLEMENTARY_IMAGE_NAME
# then append the image registry and generate the image URI.
SUPPLEMENTARY_IMAGE=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(SUPPLEMENTARY_IMAGE_NAME)
SUPPLEMENTARY_IMAGE_URI=$(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/$(SUPPLEMENTARY_IMAGE_NAME):${OPERATOR_IMAGE_TAG}
endif

ifeq ($(EnableOLMSkipRange), true)
SKIP_RANGE_ENABLED=true
else
SKIP_RANGE_ENABLED=false
endif

# Consumer can optionally define ADDITIONAL_IMAGE_SPECS like:
#     define ADDITIONAL_IMAGE_SPECS
#     ./path/to/a/Dockerfile $(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/a-image:v1.2.3
#     ./path/to/b/Dockerfile $(IMAGE_REGISTRY)/$(IMAGE_REPOSITORY)/b-image:v4.5.6
#     endef
# Each will be conditionally built and pushed along with the operator image.
define IMAGES_TO_BUILD
$(OPERATOR_DOCKERFILE) $(OPERATOR_IMAGE_URI)
$(ADDITIONAL_IMAGE_SPECS)
endef
export IMAGES_TO_BUILD

OLM_BUNDLE_IMAGE = $(OPERATOR_IMAGE)-bundle
OLM_CATALOG_IMAGE = $(OPERATOR_IMAGE)-catalog
OLM_CHANNEL ?= alpha

REGISTRY_USER ?=
REGISTRY_TOKEN ?=

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GOBIN?=$(shell go env GOBIN)

# Consumers may override GOFLAGS_MOD e.g. to use `-mod=vendor`
unexport GOFLAGS
GOFLAGS_MOD ?=

# In openshift ci (Prow), we need to set $HOME to a writable directory else tests will fail
# because they don't have permissions to create /.local or /.cache directories
# as $HOME is set to "/" by default.
ifeq ($(HOME),/)
export HOME=/tmp/home
endif
PWD=$(shell pwd)

GOENV=GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=1 GOFLAGS="${GOFLAGS_MOD}"
GOBUILDFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

ifeq (${FIPS_ENABLED}, true)
GOFLAGS_MOD+=-tags=fips_enabled
GOFLAGS_MOD:=$(strip ${GOFLAGS_MOD})
$(warning Setting GOEXPERIMENT=strictfipsruntime,boringcrypto - this generally causes builds to fail unless building inside the provided Dockerfile. If building locally consider calling 'go build .')
GOENV+=GOEXPERIMENT=strictfipsruntime,boringcrypto
GOENV:=$(strip ${GOENV})
endif

# GOLANGCI_LINT_CACHE needs to be set to a directory which is writeable
# Relevant issue - https://github.com/golangci/golangci-lint/issues/734
GOLANGCI_LINT_CACHE ?= /tmp/golangci-cache

GOLANGCI_OPTIONAL_CONFIG ?=

ifeq ($(origin TESTTARGETS), undefined)
TESTTARGETS := $(shell ${GOENV} go list -e ./... | grep -E -v "/(vendor)/" | grep -E -v "/(osde2e)/")
endif
# ex, -v
TESTOPTS :=

ALLOW_DIRTY_CHECKOUT?=false

# TODO: Figure out how to discover this dynamically
CONVENTION_DIR := boilerplate/openshift/golang-osd-operator
BOILERPLATE_CONTAINER_MAKE := boilerplate/_lib/container-make

# Set the default goal in a way that works for older & newer versions of `make`:
# Older versions (<=3.8.0) will pay attention to the `default` target.
# Newer versions pay attention to .DEFAULT_GOAL, where unsetting it makes the next defined target the default:
# https://www.gnu.org/software/make/manual/make.html#index-_002eDEFAULT_005fGOAL-_0028define-default-goal_0029
.DEFAULT_GOAL :=
.PHONY: default
default: go-check go-test go-build

.PHONY: clean
clean:
	rm -rf ./build/_output

.PHONY: isclean
isclean:
	@(test "$(ALLOW_DIRTY_CHECKOUT)" != "false" || test 0 -eq $$(git status --porcelain | wc -l)) || (echo "Local git checkout is not clean, commit changes and try again or use ALLOW_DIRTY_CHECKOUT=true to override." >&2 && git --no-pager diff && exit 1)

# TODO: figure out how to docker-login only once across multiple `make` calls
.PHONY: docker-build-push-one
docker-build-push-one: isclean docker-login
	@(if [[ -z "${IMAGE_URI}" ]]; then echo "Must specify IMAGE_URI"; exit 1; fi)
	@(if [[ -z "${DOCKERFILE_PATH}" ]]; then echo "Must specify DOCKERFILE_PATH"; exit 1; fi)
	${CONTAINER_ENGINE} build --pull -f $(DOCKERFILE_PATH) -t $(IMAGE_URI) .
	${CONTAINER_ENGINE} push ${IMAGE_URI}

# TODO: Get rid of docker-build. It's only used by opm-build-push
.PHONY: docker-build
docker-build: isclean
	${CONTAINER_ENGINE} build --pull -f $(OPERATOR_DOCKERFILE) -t $(OPERATOR_IMAGE_URI) .
	${CONTAINER_ENGINE} tag $(OPERATOR_IMAGE_URI) $(OPERATOR_IMAGE_URI_LATEST)

# TODO: Get rid of docker-push. It's only used by opm-build-push
.PHONY: docker-push
docker-push: docker-login docker-build
	${CONTAINER_ENGINE} push ${OPERATOR_IMAGE_URI}
	${CONTAINER_ENGINE} push ${OPERATOR_IMAGE_URI_LATEST}

# TODO: Get rid of push. It's not used.
.PHONY: push
push: docker-push

.PHONY: docker-login
docker-login:
	@test "${REGISTRY_USER}" != "" && test "${REGISTRY_TOKEN}" != "" || (echo "REGISTRY_USER and REGISTRY_TOKEN must be defined" && exit 1)
	mkdir -p ${CONTAINER_ENGINE_CONFIG_DIR}
	@${CONTAINER_ENGINE} login -u="${REGISTRY_USER}" -p="${REGISTRY_TOKEN}" quay.io

.PHONY: go-check
go-check: ## Golang linting and other static analysis
	${CONVENTION_DIR}/ensure.sh golangci-lint
	GOLANGCI_LINT_CACHE=${GOLANGCI_LINT_CACHE} golangci-lint run -c ${CONVENTION_DIR}/golangci.yml ./...
	test "${GOLANGCI_OPTIONAL_CONFIG}" = "" || test ! -e "${GOLANGCI_OPTIONAL_CONFIG}" || GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE}" golangci-lint run -c "${GOLANGCI_OPTIONAL_CONFIG}" ./...

.PHONY: go-generate
go-generate:
	${GOENV} go generate $(TESTTARGETS)
	# Don't forget to commit generated files

# go-get-tool will 'go install' any package $2 and install it to $1.
define go-get-tool
@{ \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(shell dirname $(1)) go install $(2) ;\
echo "Installed in $(1)" ;\
rm -rf $$TMP_DIR ;\
}
endef

CONTROLLER_GEN = controller-gen
OPENAPI_GEN = openapi-gen
KUSTOMIZE = kustomize
YQ = yq

.PHONY: op-generate
## CRD v1beta1 is no longer supported.
op-generate:
	cd ./api; $(CONTROLLER_GEN) crd:crdVersions=v1,generateEmbeddedObjectMeta=true paths=./... output:dir=$(PWD)/deploy/crds
	cd ./api; $(CONTROLLER_GEN) object paths=./...

.PHONY: openapi-generate
openapi-generate:
	find ./api -maxdepth 2 -mindepth 1 -type d | xargs -t -I% \
		$(OPENAPI_GEN) --logtostderr=true \
			-i % \
			-o "" \
			-O zz_generated.openapi \
			-p % \
			-h /dev/null \
			-r "-"

.PHONY: manifests
manifests:
# Only use kustomize to template out manifests if the path config/default exists
ifneq (,$(wildcard config/default))
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(KUSTOMIZE) build config/default | $(YQ) -s '"deploy/" + .metadata.name + "." + .kind + ".yaml"'
else
	$(info Did not find 'config/default' - skipping kustomize manifest generation)
endif

.PHONY: generate
generate: op-generate go-generate openapi-generate manifests

ifeq (${FIPS_ENABLED}, true)
go-build: ensure-fips
endif

.PHONY: go-build
go-build: ## Build binary
	# Force GOOS=linux as we may want to build containers in other *nix-like systems (ie darwin).
	# This is temporary until a better container build method is developed
	${GOENV} GOOS=linux go build ${GOBUILDFLAGS} -o build/_output/bin/$(OPERATOR_NAME) .

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23
SETUP_ENVTEST = setup-envtest

.PHONY: setup-envtest
setup-envtest:
	$(eval KUBEBUILDER_ASSETS := "$(shell $(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) -p path --bin-dir /tmp/envtest/bin)")

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: go-test
go-test: setup-envtest
	KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS) go test $(TESTOPTS) $(TESTTARGETS)

.PHONY: python-venv
python-venv:
	${CONVENTION_DIR}/ensure.sh venv ${CONVENTION_DIR}/py-requirements.txt
	$(eval PYTHON := .venv/bin/python3)

.PHONY: generate-check
generate-check:
	@$(MAKE) -s isclean --no-print-directory
	@$(MAKE) -s generate --no-print-directory
	@$(MAKE) -s isclean --no-print-directory || (echo 'Files after generation are different than committed ones. Please commit updated and unaltered generated files' >&2 && exit 1)
	@echo "All generated files are up-to-date and unaltered"

.PHONY: yaml-validate
yaml-validate: python-venv
	${PYTHON} ${CONVENTION_DIR}/validate-yaml.py $(shell git ls-files | grep -E -v '^(vendor|boilerplate)/' | grep -E '.*\.ya?ml')

.PHONY: olm-deploy-yaml-validate
olm-deploy-yaml-validate: python-venv
	${PYTHON} ${CONVENTION_DIR}/validate-yaml.py $(shell git ls-files 'deploy/*.yaml' 'deploy/*.yml')

.PHONY: prow-config
prow-config:
	${CONVENTION_DIR}/prow-config ${RELEASE_CLONE}


######################
# Targets used by prow
######################

# validate: Ensure code generation has not been forgotten; and ensure
# generated and boilerplate code has not been modified.
.PHONY: validate
validate: boilerplate-freeze-check generate-check

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

#########################
# Targets used by app-sre
#########################

# build-push: Construct, tag, and push the official operator and
# registry container images.
# TODO: Boilerplate this script.
.PHONY: build-push
build-push:
	${CONVENTION_DIR}/app-sre-build-deploy.sh ${REGISTRY_IMAGE} ${CURRENT_COMMIT} "$$IMAGES_TO_BUILD"

.PHONY: opm-build-push
opm-build-push: docker-push
	OLM_BUNDLE_IMAGE="${OLM_BUNDLE_IMAGE}" \
	OLM_CATALOG_IMAGE="${OLM_CATALOG_IMAGE}" \
	CONTAINER_ENGINE="${CONTAINER_ENGINE}" \
	CONTAINER_ENGINE_CONFIG_DIR="${CONTAINER_ENGINE_CONFIG_DIR}" \
	CURRENT_COMMIT="${CURRENT_COMMIT}" \
	OPERATOR_VERSION="${OPERATOR_VERSION}" \
	OPERATOR_NAME="${OPERATOR_NAME}" \
	OPERATOR_IMAGE="${OPERATOR_IMAGE}" \
	OPERATOR_IMAGE_TAG="${OPERATOR_IMAGE_TAG}" \
	OLM_CHANNEL="${OLM_CHANNEL}" \
	${CONVENTION_DIR}/build-opm-catalog.sh

.PHONY: ensure-fips
ensure-fips:
	${CONVENTION_DIR}/configure-fips.sh

# You will need to export the forked/cloned operator repository directory as OLD_SDK_REPO_DIR to make this work.
# Example: export OLD_SDK_REPO_DIR=~/Projects/My-Operator-Fork
.PHONY: migrate-to-osdk1
migrate-to-osdk1:
ifndef OLD_SDK_REPO_DIR
	$(error OLD_SDK_REPO_DIR is not set)
endif
	# Copying files & folders from old repository to current project
	rm -rf config
	rsync -a $(OLD_SDK_REPO_DIR)/deploy . --exclude=crds
	rsync -a $(OLD_SDK_REPO_DIR)/pkg . --exclude={'apis','controller'}
	rsync -a $(OLD_SDK_REPO_DIR)/Makefile .
	rsync -a $(OLD_SDK_REPO_DIR)/.gitignore .
	rsync -a $(OLD_SDK_REPO_DIR)/ . --exclude={'cmd','version','boilerplate','deploy','pkg'} --ignore-existing

# Boilerplate container-make targets.
# Runs 'make' in the boilerplate backing container.
# If the command fails, starts a shell in the container so you can debug.
.PHONY: container-test
container-test:
	${BOILERPLATE_CONTAINER_MAKE} test

.PHONY: container-generate
container-generate:
	${BOILERPLATE_CONTAINER_MAKE} generate

.PHONY: container-lint
container-lint:
	${BOILERPLATE_CONTAINER_MAKE} lint

.PHONY: container-validate
container-validate:
	${BOILERPLATE_CONTAINER_MAKE} validate

.PHONY: container-coverage
container-coverage:
	${BOILERPLATE_CONTAINER_MAKE} coverage

.PHONY: rvmo-bundle
rvmo-bundle:
	OPERATOR_NAME=$(OPERATOR_NAME) \
	OPERATOR_VERSION=$(OPERATOR_VERSION) \
	OPERATOR_OLM_REGISTRY_IMAGE=$(REGISTRY_IMAGE) \
	TEMPLATE_FILE=$(abspath hack/olm-registry/olm-artifacts-template.yaml) \
	bash ${CONVENTION_DIR}/rvmo-bundle.sh
