.PHONY: test test-coverage lint build clean vet generate dev-up dev-down help

## Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run all tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo "---"
	@echo "HTML report: go tool cover -html=coverage.out -o coverage.html"

lint: ## Run golangci-lint
	golangci-lint run ./...

build: ## Build all examples
	go build ./...

vet: ## Run go vet
	go vet ./...

clean: ## Clean build artifacts
	rm -f coverage.out coverage.html

generate: ## Run code generation
	go generate ./...

dev-up: ## Start development environment
	docker compose -f .devenv/docker-compose.yml up -d

dev-down: ## Stop development environment
	docker compose -f .devenv/docker-compose.yml down
