.PHONY: help build run test clean docker lint fmt deps dev prod logs

APP_NAME := eth-validator-api
MAIN_PATH := ./cmd/api
BINARY_NAME := api
DOCKER_IMAGE := $(APP_NAME):latest
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags="-w -s -X main.version=$$(git describe --tags --always --dirty) -X main.commit=$$(git rev-parse HEAD) -X main.date=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Run application in development mode with hot reload
	@echo "Starting development server..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Installing air..."; \
		go install github.com/air-verse/air@latest; \
		$(shell go env GOPATH)/bin/air; \
	fi

run: ## Run the application
	@echo "Running application..."
	$(GO) run $(GOFLAGS) $(MAIN_PATH)

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

build-linux: ## Build for Linux
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME)-linux $(MAIN_PATH)

build-all: build build-linux ## Build for all platforms

test: ## Run tests
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	$(GO) test -race $(GOFLAGS) ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -tags=integration $(GOFLAGS) ./test/...

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if command -v goimports > /dev/null; then \
		goimports -w .; \
	else \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		goimports -w .; \
	fi

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GO) mod verify

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

docker-compose-up: ## Start services with docker-compose
	@echo "Starting services..."
	docker-compose up -d

docker-compose-down: ## Stop services
	@echo "Stopping services..."
	docker-compose down

docker-compose-logs: ## View docker-compose logs
	docker-compose logs -f

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE)

logs: ## Tail application logs
	@if [ -f "app.log" ]; then \
		tail -f app.log; \
	else \
		echo "No log file found. Run the application first."; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux
	rm -f coverage.out coverage.html
	rm -rf dist/
	$(GO) clean -cache

clean-docker: ## Clean Docker artifacts
	@echo "Cleaning Docker artifacts..."
	docker-compose down -v
	docker rmi $(DOCKER_IMAGE) || true

install: build ## Install the application
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(GOFLAGS) $(LDFLAGS) $(MAIN_PATH)

uninstall: ## Uninstall the application
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $$(go env GOPATH)/bin/$(BINARY_NAME)

prod: ## Build and run for production
	@echo "Building for production..."
	$(MAKE) build
	@echo "Starting production server..."
	./$(BINARY_NAME)

version: ## Show version information
	@echo "Version: $$(git describe --tags --always --dirty)"
	@echo "Commit: $$(git rev-parse HEAD)"
	@echo "Date: $$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	@echo "Go version: $$(go version)"

env-example: ## Create .env from .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created from .env.example"; \
	else \
		echo ".env file already exists"; \
	fi

check: lint vet test ## Run all checks (lint, vet, test)
	@echo "All checks passed!"

ci: deps check build ## Run CI pipeline

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

profile-cpu: ## Generate CPU profile
	@echo "Generating CPU profile..."
	$(GO) test -cpuprofile=cpu.prof -bench=. ./...
	$(GO) tool pprof cpu.prof

profile-mem: ## Generate memory profile
	@echo "Generating memory profile..."
	$(GO) test -memprofile=mem.prof -bench=. ./...
	$(GO) tool pprof mem.prof

up: docker-compose-up ## Alias for docker-compose-up
down: docker-compose-down ## Alias for docker-compose-down
dc-logs: docker-compose-logs ## Alias for docker-compose-logs