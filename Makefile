MAKEFILE_DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Image URL to use all building/pushing image targets
IMG = $(CONTROLLER_IMAGE_NAME)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: clean
clean: ## Clean dev tools and artifacts
	find bin -type f -maxdepth 1 -delete
	rm -rf dist

.PHONY: manifests
manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: docs
docs: ## Generate CRD documents.
	$(HACK)/make-docs.sh $(CRD_DOCS_DIR)

.PHONY: generate
generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST))" go test -v $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# CertManager is installed by default; skip with:
# - CERT_MANAGER_INSTALL_SKIP=true
.PHONY: test-e2e
test-e2e: manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	@command -v kind >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
ifeq ($(CI),true)
	@kind get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
else
	@echo "To avoid errors caused by pre-existing resources in the kind cluster before make test-e2e, we will recreate the cluster."
	kind delete cluster || true
	kind create cluster --image $(KIND_IMAGE_NAME)
endif
	go test ./test/e2e/ -v -ginkgo.v

.PHONY: lint
lint: lint-plugins lint-licenses ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: lint-plugins ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: lint-plugins ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

.PHONY: lint-plugins
lint-plugins:
	$(HACK)/setup-golangci-lint.sh

.PHONY: lint-licenses
lint-licenses:
	$(HACK)/verify-licenses.sh

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
ifeq ($(CI),true)
docker-build: ## Rename controller image for test-e2e.
	$(CONTAINER_TOOL) pull $(IMAGE_REPO):$(IMAGE_TAG)
	$(CONTAINER_TOOL) tag $(IMAGE_REPO):$(IMAGE_TAG) ${IMG}
else
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} --build-arg GO_VERSION=$(GO_VERSION) --build-arg KUBECTL_VERSION=$(KUBECTL_VERSION) .
endif

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	($(KUBECTL) apply -k config/crd) || ($(KUBECTL) replace -k config/crd --force)

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUBECTL) delete -k config/crd --ignore-not-found=$(ignore-not-found)

.PHONY: deploy
deploy: manifests delete-controller ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	($(KUBECTL) apply -k config/default) || ($(KUBECTL) replace -k config/default --force)

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUBECTL) delete -k config/default --ignore-not-found=$(ignore-not-found)

.PHONY: sample
sample: ## Deploy sample manifests
	$(KUBECTL) apply -k config/samples

.PHONY: delete-controller
delete-controller:
	$(KUBECTL) -n k8s-job-wrapper-system delete deploy k8s-job-wrapper-controller-manager --ignore-not-found=true

.PHONY: deploy-local
deploy-local: ## Install CRDs and deploy controller into the kind cluster.
	$(HACK)/deploy-local.sh $(KIND_IMAGE_NAME) $(CONTROLLER_IMAGE_NAME)

##@ Dependencies

## Tool Binaries
HACK = $(MAKEFILE_DIR)/hack
TOOLS = $(HACK)/tools.sh
KUBECTL ?= $(HACK)/kubectl
KUSTOMIZE ?= $(TOOLS) kustomize
CONTROLLER_GEN ?= $(TOOLS) controller-gen
ENVTEST ?= $(TOOLS) setup-envtest
GOLANGCI_LINT = $(TOOLS) golangci-lint
