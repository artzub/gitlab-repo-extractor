##@ General

default: help ## Default target, displays help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

reviewable: setup fmt lint test ## Run before committing.

fmt: install-gofumpt ## Format code
	@gofumpt -l -w ./

lint: install-golangci-lint ## Run linter
	@golangci-lint run

.PHONY: setup
setup: install-golangci-lint install-gofumpt ## Setup your local environment
	go mod tidy

install-golangci-lint:
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

install-gofumpt:
	@go install mvdan.cc/gofumpt@latest

test: ## Run tests
	go test ./... -race
