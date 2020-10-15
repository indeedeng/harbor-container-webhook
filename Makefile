MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := all

BIN_DIR ?= $(shell go env GOPATH)/bin
export PATH := $(PATH):$(BIN_DIR)

.PHONY: deps
deps: ## download go modules
	go mod download

.PHONY: fmt
fmt: lint/check ## ensure consistent code style
	go run oss.indeed.com/go/go-groups -w .
	golangci-lint run --fix > /dev/null 2>&1 || true
	go mod tidy

.PHONY: lint/check
lint/check:
	@if ! golangci-lint --version > /dev/null 2>&1; then \
		echo -e "\033[0;33mgolangci-lint is not installed: run \`\033[0;32mmake lint-install\033[0m\033[0;33m\` or install it from https://golangci-lint.run\033[0m"; \
		exit 1; \
	fi

.PHONY: lint-install
lint-install: ## installs golangci-lint to the go bin dir
	@if ! golangci-lint --version > /dev/null 2>&1; then \
		echo "Installing golangci-lint"; \
		curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(BIN_DIR) v1.31.0; \
	fi

.PHONY: lint
lint: lint/check ## run golangci-lint
	golangci-lint run
	@if [ -n "$$(go run oss.indeed.com/go/go-groups -l .)" ]; then \
		echo -e "\033[0;33mdetected fmt problems: run \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m"; \
		exit 1; \
	fi

.PHONY: test
test: lint ## run go tests
	go test ./... -race

.PHONY: gen
gen:
	go generate ./...

.PHONY: build
build: ## build harbor-container-webhook binary
	go build -o bin/harbor-container-webhook main.go

docker-build: test ## build the docker image
	docker build . -t ${IMG}

docker-push: ## push the docker image
	docker push ${IMG}

.PHONY: all
all: test gen build

.PHONY: help
help: ## displays this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_\/-]+:.*?## / {printf "\033[34m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | \
		sort | \
		grep -v '#'