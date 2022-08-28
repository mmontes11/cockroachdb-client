# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: help

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: lint
lint: golangci-lint ## Lint.
	$(GOLANGCI_LINT) run

.PHONY: run
run: lint ## Run a controller from your host.
	@go run ./main.go

.PHONY: test
test: ## Run tests.
	@go test ./... -coverprofile cover.out

.PHONY: cover
cover: test ## Run tests and generate coverage.
	@go tool cover -html=cover.out -o=cover.html

##@ Build

.PHONY: build
build: ## Build manager binary.
	go build .

##@ Tooling

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
GOLANGCI_LINT_VERSION := v1.46.2

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
