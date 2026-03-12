# Global Outbox Service Go - Makefile

BINARY_NAME=global-outbox-service-go
GO_VERSION=1.21
PROTOC_VERSION=3.21.12

# Go build flags
LDFLAGS=-ldflags="-s -w"
BUILD_FLAGS=-trimpath $(LDFLAGS)

# Docker settings
DOCKER_IMAGE=global-outbox-service-go
DOCKER_TAG=latest

# Directories
BIN_DIR=bin
PROTO_DIR=pkg/proto
INTERNAL_DIR=internal
CMD_DIR=cmd/server

.PHONY: help
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup: ## Install dependencies and tools
	@echo "Setting up development environment..."
	go mod tidy
	go mod download
	@echo "Installing protobuf compiler..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: proto
proto: ## Generate protobuf code
	@echo "Generating protobuf code..."
	mkdir -p $(PROTO_DIR)
	protoc --go_out=$(PROTO_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/outbox.proto

.PHONY: build
build: proto ## Build the application
	@echo "Building $(BINARY_NAME)..."
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go

.PHONY: build-linux
build-linux: proto ## Build for Linux
	@echo "Building $(BINARY_NAME) for Linux..."
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME)-linux $(CMD_DIR)/main.go

.PHONY: run
run: build ## Run the application locally
	@echo "Running $(BINARY_NAME)..."
	./$(BIN_DIR)/$(BINARY_NAME)

.PHONY: run-dev
run-dev: proto ## Run in development mode
	@echo "Running $(BINARY_NAME) in development mode..."
	go run $(CMD_DIR)/main.go

.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	go mod tidy
	go mod verify

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -rf coverage.out coverage.html
	go clean -cache -testcache

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8042:8042 -p 50052:50052 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-push
docker-push: docker-build ## Push Docker image
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: migrations-create
migrations-create: ## Create a new migration (usage: make migrations-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrations-create NAME=migration_name"; exit 1; fi
	@echo "Creating migration: $(NAME)"
	mkdir -p migrations
	@timestamp=$$(date +%Y%m%d%H%M%S); \
	echo "-- Migration: $(NAME)" > migrations/$${timestamp}_$(NAME).up.sql; \
	echo "-- Add your up migration here" >> migrations/$${timestamp}_$(NAME).up.sql; \
	echo "-- Migration: $(NAME)" > migrations/$${timestamp}_$(NAME).down.sql; \
	echo "-- Add your down migration here" >> migrations/$${timestamp}_$(NAME).down.sql; \
	echo "Created migrations/$${timestamp}_$(NAME).up.sql"; \
	echo "Created migrations/$${timestamp}_$(NAME).down.sql"

.PHONY: health-check
health-check: ## Check service health
	@echo "Checking service health..."
	curl -f http://localhost:8042/health || echo "Service is not healthy"

.PHONY: stats
stats: ## Get service statistics
	@echo "Getting service statistics..."
	curl -s http://localhost:8042/stats | jq '.' || echo "Failed to get stats"

.PHONY: circuit-breaker-status
circuit-breaker-status: ## Get circuit breaker status
	@echo "Getting circuit breaker status..."
	curl -s http://localhost:8042/circuit-breaker | jq '.' || echo "Failed to get circuit breaker status"

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

.PHONY: security-scan
security-scan: ## Run security scan
	@echo "Running security scan..."
	govulncheck ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

.PHONY: check
check: fmt vet lint test ## Run all checks (format, vet, lint, test)

.PHONY: ci
ci: mod-tidy check build ## Run CI pipeline

.PHONY: dev-setup
dev-setup: setup install-tools proto ## Complete development setup

# Environment-specific targets
.PHONY: env-dev
env-dev: ## Copy development environment file
	cp .env.example .env.dev

.PHONY: env-prod
env-prod: ## Copy production environment file
	cp .env.example .env.prod

# Database related targets
.PHONY: db-test
db-test: ## Test database connection
	@echo "Testing database connection..."
	go run tools/db-test.go

.PHONY: create-tables
create-tables: ## Create database tables for all services
	@echo "Creating database tables..."
	go run tools/create-tables.go

# Default target
.DEFAULT_GOAL := help