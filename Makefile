# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# kubetrail.io/influxdb-operator-bundle:$VERSION and kubetrail.io/influxdb-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= kubetrail.io/influxdb-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

# ===============================================================================================
# Code below has been added for custom builds using podman
# formatting color values
RD="$(shell tput setaf 1)"
YE="$(shell tput setaf 3)"
NC="$(shell tput sgr0)"

# Image URL to use all building/pushing image targets to
# Google artifact registry
NAME=influxdb-operator
CATEGORY=services
TAG=0.0.1-dev-1
REPO=us-central1-docker.pkg.dev/${PROJECT}
IMG_TMP=${REPO}/tmp/${CATEGORY}/${NAME}:${TAG}
IMG_BASE=${REPO}/artifacts/${CATEGORY}/${NAME}

# sanity check
.PHONY: _sanity
_sanity:
	@if [[ -z "${PROJECT}" ]]; then \
		echo "please set PROJECT env. var for your Google cloud project"; \
		exit 1; \
	fi
	@for cmd in podman kubectl helm go goimports; do \
		if [[ -z $$(command -v $${cmd}) ]]; then \
			echo "$${cmd} not found. pl. install."; \
			exit 1; \
		fi; \
	done

.PHONY: goimports ## Run goimports against code
goimports:
	goimports -w -l ./main.go
	goimports -w -l ./api
	goimports -w -l ./controllers

.PHONY: vendor ## Run go vendor against code
vendor:
	@echo need golang version 1.17.x or higher
	go version
	rm -rf vendor
	go mod vendor

# podman-build-push builds amd64 and arm64 containers and links them
# to a manifest file that is compatible with both arch
# This will only work with podman on Linux OS!
.PHONY: podman-build-push
podman-build-push: _podman-build-push generate
	@podman manifest inspect ${IMG_BASE}:${TAG} | jq '.'
	@echo -e ${YE}▶ container images${NC}
	@podman images | grep ${IMG_BASE}

.PHONY: _podman-build-push
_podman-build-push: _sanity goimports vendor
	@echo -e ${YE}▶ building and pushing tmp container${NC}
	@podman build -t ${IMG_TMP} -f ./Dockerfile-podman
	@podman push ${IMG_TMP}
	@echo -e ${YE}▶ building and pushing amd64 container${NC}
	@cat Dockerfile-podman.tmpl | \
		PROJECT=${PROJECT} \
		CATEGORY=${CATEGORY} \
		NAME=${NAME} \
		TAG=${TAG} \
		ARCH=amd64 \
		envsubst | \
		podman build --arch=amd64 -t ${IMG_BASE}:${TAG}.amd64 -f -
	@podman push ${IMG_BASE}:${TAG}.amd64
	@echo -e ${YE}▶ building and pushing arm64 container${NC}
	@cat Dockerfile-podman.tmpl | \
		PROJECT=${PROJECT} \
		CATEGORY=${CATEGORY} \
		NAME=${NAME} \
		TAG=${TAG} \
		ARCH=arm64 \
		envsubst | \
		podman build --arch=arm64 -t ${IMG_BASE}:${TAG}.arm64 -f -
	@podman push ${IMG_BASE}:${TAG}.arm64
	@echo -e ${YE}▶ creating or modifying manifest${NC}
	@podman manifest create ${IMG_BASE}:${TAG} || \
		for digest in $$(podman manifest inspect ${IMG_BASE}:${TAG} | jq -r '.manifests[].digest'); do \
			podman manifest remove ${IMG_BASE}:${TAG} $${digest}; \
		done
	@echo -e ${YE}▶ adding amd64 container to manifest${NC}
	@podman manifest add ${IMG_BASE}:${TAG} ${IMG_BASE}:${TAG}.amd64
	@echo -e ${YE}▶ adding arm64 container to manifest${NC}
	@podman manifest add ${IMG_BASE}:${TAG} ${IMG_BASE}:${TAG}.arm64
	@echo -e ${YE}▶ pushing manifest${NC}
	@podman push ${IMG_BASE}:${TAG}

# deploy-manifests creates manifests with custom updates of image name
.PHONY: deploy-manifests
deploy-manifests: manifests kustomize ## Deploy-manifests generates k8s manifests without deploying
	cd config/manager && $(KUSTOMIZE) edit set image controller=controller:latest
	$(KUSTOMIZE) build config/default --output config/extra/manifests.yaml
	sed -i -e "s/image: controller:latest/image: $$(echo -n ${IMG_BASE}:${TAG} | sed -e 's/\//\\\//g')/g" config/extra/manifests.yaml
	$(KUSTOMIZE) build config/extra > config/samples/manifests.yaml
	@echo "==============="
	@echo kubectl apply -f config/samples/manifests.yaml

# logs monitors logs from the controller
.PHONY: logs
logs:
	@kubectl --namespace=${NAME}-system logs -f deployments.apps/${NAME}-controller-manager -c manager

# reboot scales down and then up the controller deployment
.PHONY: reboot
reboot:
	@kubectl --namespace=${NAME}-system scale deployment --replicas=0 ${NAME}-controller-manager
	@kubectl --namespace=${NAME}-system scale deployment --replicas=1 ${NAME}-controller-manager

# watch watches a few resources
.PHONY: watch
watch:
	@watch kubectl --namespace=${NAME}-system get pods,svc,configmaps,secrets,servicemonitors
