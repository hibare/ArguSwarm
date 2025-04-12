SHELL=/bin/bash

MAKEFLAGS += -s

# Bold
BCYAN=\033[1;36m
BBLUE=\033[1;34m

# No color (Reset)
NC=\033[0m

.DEFAULT_GOAL := help

.PHONY: init
init: ## Initialize the project
	$(MAKE) install-golangci-lint
	$(MAKE) install-pre-commit

.PHONY: install-golangci-lint
install-golangci-lint: ## Install golangci-lint
ifeq (, $(shell which golangci-lint))
	@echo "Installing golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin
endif

.PHONY: install-pre-commit
install-pre-commit: ## Install pre-commit
	pre-commit install

.PHONY: dev
dev: ## Run the development server
	docker compose -f docker-compose.dev.yml up

.PHONY: test
test: ## Run the tests
	go test -v ./... -cover

.PHONY: build
build: ## Build the project
	docker build -t arguswarm . --build-arg GOLANG_VERSION=1.24.2 --build-arg VERSION=0.1.0 --progress=plain --no-cache

.PHONY: clean
clean: ## Clean up the development server
	docker compose -f docker-compose.dev.yml down

.PHONY: help
help: ## Disply this help
		@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(BCYAN)%-25s$(NC)%s\n", $$1, $$2}'
