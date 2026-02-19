.PHONY: help build test run clean install dev

# Variables
BINARY_NAME=github-code-agent
GO_FILES=$(shell find . -name '*.go' -type f)
MAIN_FILE=cmd/agent/main.go

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "✓ Build complete: ./$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

run: ## Run the application (requires .env file)
	@echo "Starting $(BINARY_NAME)..."
	go run $(MAIN_FILE)

dev: ## Run with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
	@command -v air >/dev/null 2>&1 || { echo "air not found. Install with: go install github.com/cosmtrek/air@latest"; exit 1; }
	air

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

lint: ## Run linter with auto-fix
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install from: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run --fix

lint-check: ## Run linter without fixes (for CI)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install from: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Vet complete"

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy
	@echo "✓ Tidy complete"

check: fmt vet tidy test lint ## Run formatting, vet, tidy, tests, and linting

setup-env: ## Create .env file from template
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✓ Created .env file. Please edit it with your configuration."; \
	else \
		echo "✗ .env file already exists"; \
	fi

docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

.DEFAULT_GOAL := help
