# Workflow Engine Service - Go Implementation Makefile

# Application name and version
APP_NAME := workflow-engine
VERSION := 1.0.0
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build parameters
BINARY_DIR := bin
BINARY_NAME := $(BINARY_DIR)/$(APP_NAME)
MAIN_FILE := cmd/server/main.go

# Docker parameters
DOCKER_IMAGE := workflow-engine:$(VERSION)
DOCKER_REGISTRY := localhost:5000
COMPOSE_FILE := docker-compose.yml

# Database parameters
DB_NAME := workflow_engine
DB_USER := workflow_user
DB_PASSWORD := workflow_password
DB_HOST := localhost
DB_PORT := 5432
DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Test parameters
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Build flags
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)
BUILD_FLAGS := -ldflags "$(LDFLAGS)" -trimpath

.PHONY: help
help: ## Display this help message
	@echo "Workflow Engine Service - Go Implementation"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-20s\033[0m %s\n", "Command", "Description"} /^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: setup
setup: ## Set up development environment
	@echo "Setting up development environment..."
	$(GOMOD) download
	$(GOMOD) tidy
	@mkdir -p $(BINARY_DIR)
	@mkdir -p logs
	@echo "Development environment setup complete"

.PHONY: build
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BINARY_NAME)"

.PHONY: build-linux
build-linux: ## Build for Linux
	@echo "Building $(APP_NAME) for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME)-linux $(MAIN_FILE)
	@echo "Linux build complete: $(BINARY_NAME)-linux"

.PHONY: build-windows
build-windows: ## Build for Windows
	@echo "Building $(APP_NAME) for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME)-windows.exe $(MAIN_FILE)
	@echo "Windows build complete: $(BINARY_NAME)-windows.exe"

.PHONY: run
run: build ## Build and run the application
	@echo "Running $(APP_NAME)..."
	./$(BINARY_NAME)

.PHONY: run-dev
run-dev: ## Run in development mode with hot reload
	@echo "Running $(APP_NAME) in development mode..."
	DEBUG=true LOG_LEVEL=debug air

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BINARY_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "Clean complete"

##@ Testing

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	$(GOTEST) -race -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -tags=integration -v ./tests/integration/...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

##@ Database

.PHONY: db-start
db-start: ## Start database with Docker Compose
	@echo "Starting database services..."
	docker-compose up -d postgres redis
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Database services started"

.PHONY: db-stop
db-stop: ## Stop database services
	@echo "Stopping database services..."
	docker-compose stop postgres redis
	@echo "Database services stopped"

.PHONY: db-reset
db-reset: ## Reset database (warning: destroys all data)
	@echo "Resetting database (this will destroy all data)..."
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v postgres; \
		docker-compose up -d postgres; \
		sleep 10; \
		echo "Database reset complete"; \
	else \
		echo "Database reset cancelled"; \
	fi

.PHONY: db-migrate
db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	@echo "Migrations are automatically applied on container startup"

.PHONY: db-shell
db-shell: ## Connect to database shell
	@echo "Connecting to database..."
	docker-compose exec postgres psql -U $(DB_USER) -d $(DB_NAME)

.PHONY: db-logs
db-logs: ## View database logs
	docker-compose logs -f postgres

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

.PHONY: docker-run
docker-run: ## Run with Docker Compose
	@echo "Starting all services with Docker Compose..."
	docker-compose up -d
	@echo "All services started. Access:"
	@echo "  - Workflow Engine: http://localhost:8017"
	@echo "  - Grafana: http://localhost:3000 (admin:admin123)"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Jaeger: http://localhost:16686"
	@echo "  - Adminer: http://localhost:8080"

.PHONY: docker-stop
docker-stop: ## Stop Docker services
	@echo "Stopping Docker services..."
	docker-compose stop

.PHONY: docker-down
docker-down: ## Stop and remove Docker services
	@echo "Stopping and removing Docker services..."
	docker-compose down

.PHONY: docker-logs
docker-logs: ## View Docker logs
	docker-compose logs -f workflow-engine

.PHONY: docker-shell
docker-shell: ## Access Docker container shell
	docker-compose exec workflow-engine sh

##@ Quality

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting Go code..."
	$(GOCMD) fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run ./...

.PHONY: security
security: ## Run security scan
	@echo "Running security scan..."
	gosec ./...

.PHONY: quality
quality: fmt vet lint security ## Run all quality checks

##@ Monitoring

.PHONY: metrics
metrics: ## View metrics endpoint
	@echo "Fetching metrics..."
	curl -s http://localhost:8017/metrics | head -20

.PHONY: health
health: ## Check service health
	@echo "Checking service health..."
	curl -s http://localhost:8017/health | jq .

.PHONY: monitor-start
monitor-start: ## Start monitoring stack
	@echo "Starting monitoring stack..."
	docker-compose up -d prometheus grafana jaeger
	@echo "Monitoring services started:"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Grafana: http://localhost:3000"
	@echo "  - Jaeger: http://localhost:16686"

##@ Performance

.PHONY: load-test
load-test: ## Run load tests
	@echo "Running load tests..."
	@echo "Make sure the service is running first with: make docker-run"
	@echo "Installing hey if not available..."
	@which hey > /dev/null || go install github.com/rakyll/hey@latest
	hey -n 1000 -c 10 -H "Content-Type: application/json" \
		-d '{"patient_id":"test-123","correlation_id":"load-test","medication_request":{},"clinical_intent":{},"provider_context":{}}' \
		http://localhost:8017/api/v1/orchestration/medication

.PHONY: profile
profile: ## Generate CPU profile
	@echo "Generating CPU profile..."
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	$(GOCMD) tool pprof cpu.prof

##@ Release

.PHONY: release
release: quality test docker-build ## Prepare release (run all checks and build)
	@echo "Release preparation complete"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

.PHONY: tag
tag: ## Create git tag
	@echo "Creating git tag v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release version $(VERSION)"
	git push origin v$(VERSION)

##@ Utilities

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	$(GOGET) -u github.com/cosmtrek/air@latest
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	$(GOGET) -u github.com/rakyll/hey@latest

.PHONY: version
version: ## Show version information
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

.PHONY: env
env: ## Show environment configuration
	@echo "Environment Configuration:"
	@echo "  DB_URL: $(DB_URL)"
	@echo "  Binary: $(BINARY_NAME)"
	@echo "  Docker Image: $(DOCKER_IMAGE)"

# Default target
.DEFAULT_GOAL := help