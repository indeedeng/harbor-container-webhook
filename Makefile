# set the shell to bash always
SHELL         := /bin/bash

# set make and shell flags to exit on errors
MAKEFLAGS     += --warn-undefined-variables
.SHELLFLAGS   := -euo pipefail -c

ARCH = amd64
BUILD_ARGS ?=

DOCKER_BUILD_PLATFORMS = linux/amd64,linux/arm64
DOCKER_BUILDX_BUILDER ?= "harbor-container-webhook"

# default target is build
.DEFAULT_GOAL := all
.PHONY: all
all: $(addprefix build-,$(ARCH))

# Image registry for build/push image targets
IMAGE_REGISTRY ?= ghcr.io/indeedeng/harbor-container-webhook

HELM_DIR    ?= deploy/charts/harbor-container-webhook

OUTPUT_DIR  ?= bin

RUN_GOLANGCI_LINT := go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8

# check if there are any existing `git tag` values
ifeq ($(shell git tag),)
# no tags found - default to initial tag `v0.0.0`
VERSION ?= $(shell echo "v0.0.0-$$(git rev-list HEAD --count)-g$$(git describe --dirty --always)" | sed 's/-/./2' | sed 's/-/./2')
else
# use tags
VERSION ?= $(shell git describe --dirty --always --tags --exclude 'helm*' | sed 's/-/./2' | sed 's/-/./2')
endif

# ====================================================================================
# Colors

BLUE         := $(shell printf "\033[34m")
YELLOW       := $(shell printf "\033[33m")
RED          := $(shell printf "\033[31m")
GREEN        := $(shell printf "\033[32m")
CNone        := $(shell printf "\033[0m")

# ====================================================================================
# Logger

TIME_LONG	= `date +%Y-%m-%d' '%H:%M:%S`
TIME_SHORT	= `date +%H:%M:%S`
TIME		= $(TIME_SHORT)

INFO	= echo ${TIME} ${BLUE}[ .. ]${CNone}
WARN	= echo ${TIME} ${YELLOW}[WARN]${CNone}
ERR		= echo ${TIME} ${RED}[FAIL]${CNone}
OK		= echo ${TIME} ${GREEN}[ OK ]${CNone}
FAIL	= (echo ${TIME} ${RED}[FAIL]${CNone} && false)

# ====================================================================================
# Conformance

# Ensure a PR is ready for review.
reviewable: generate helm.generate
	@go mod tidy

# Ensure branch is clean.
check-diff: reviewable
	@$(INFO) checking that branch is clean
	@test -z "$$(git status --porcelain)" || (echo "$$(git status --porcelain)" && $(FAIL))
	@$(OK) branch is clean

# ====================================================================================
# Golang

.PHONY: test
test: generate lint ## Run tests
	@$(INFO) go test unit-tests
	go test -race -v ./... -coverprofile cover.out
	@$(OK) go test unit-tests

.PHONY: build
build: $(addprefix build-,$(ARCH))

.PHONY: build-%
build-%: generate ## Build binary for the specified arch
	@$(INFO) go build $*
	@CGO_ENABLED=0 GOOS=linux GOARCH=$* \
		go build -o '$(OUTPUT_DIR)/harbor-container-webhook-$*' ./main.go
	@$(OK) go build $*

.PHONY: lint
lint: ## run golangci-lint
	$(RUN_GOLANGCI_LINT) run

fmt: ## ensure consistent code style
	@go mod tidy
	@go fmt ./...
	$(RUN_GOLANGCI_LINT) run --fix > /dev/null 2>&1 || true
	@$(OK) Ensured consistent code style

generate: ## Generate code
	@go generate ./...

# ====================================================================================
# Helm Chart

helm.docs: ## Generate helm docs
	@cd $(HELM_DIR); \
	docker run --rm -v $(shell pwd)/$(HELM_DIR):/helm-docs -u $(shell id -u) jnorwood/helm-docs:v1.5.0

HELM_VERSION ?= $(shell helm show chart $(HELM_DIR) | grep 'version:' | sed 's/version: //g')

helm.build: ## Build helm chart
	@$(INFO) helm package
	@helm package $(HELM_DIR) --dependency-update --destination $(OUTPUT_DIR)/chart
	@$(OK) helm package

# ====================================================================================
# Build Artifacts

build.all: docker.build helm.build

docker.build: docker.buildx.setup ## Build the docker image
	@$(INFO) docker build
	@docker buildx build --platform $(DOCKER_BUILD_PLATFORMS) -t $(IMAGE_REGISTRY):$(VERSION) $(BUILD_ARGS) --push .
	@$(OK) docker build

docker.buildx.setup:
	@$(INFO) docker buildx setup
	@docker buildx ls 2>/dev/null | grep -vq $(DOCKER_BUILDX_BUILDER) || docker buildx create --name $(DOCKER_BUILDX_BUILDER) --driver docker-container --driver-opt network=host --use
	@$(OK) docker buildx setup

# ====================================================================================
# Local Testing

hack/certs/tls.crt hack/certs/tls.key:
	hack/gencerts.sh

.PHONY: hack
hack: build hack/certs/tls.crt hack/certs/tls.key ## build and run the webhook w/hack config
	bin/harbor-container-webhook-* --config hack/config.yaml --kube-client-qps=5 --kube-client-burst=10

.PHONY: hack-test
hack-test: ## curl the admission and no-op json bodies to the webhook
	curl -X POST 'https://localhost:9443/webhook-v1-pod' --data-binary @hack/test/admission.json -H "Content-Type: application/json" --cert hack/certs/tls.crt --key hack/certs/tls.key --cacert hack/certs/caCert.pem
	curl -X POST 'https://localhost:9443/webhook-v1-pod' --data-binary @hack/test/no-op.json -H "Content-Type: application/json" --cert hack/certs/tls.crt --key hack/certs/tls.key --cacert hack/certs/caCert.pem

# ====================================================================================
# Help

# only comments after make target name are shown as help text
help: ## displays this help message
	@echo -e "$$(grep -hE '^\S+:.*##' $(MAKEFILE_LIST) | sed -e 's/:.*##\s*/:/' -e 's/^\(.\+\):\(.*\)/\\x1b[36m\1\\x1b[m:\2/' | column -c2 -t -s : | sort)"
