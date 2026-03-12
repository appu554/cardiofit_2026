# KB-11 Population Health Engine Makefile
# ══════════════════════════════════════════════════════════════════════════════
# "KB-11 answers population-level questions, NOT patient-level decisions."
# ══════════════════════════════════════════════════════════════════════════════

.PHONY: help build run test test-unit test-integration test-coverage lint fmt \
        clean migrate migrate-down docker-build docker-run health deps

# ──────────────────────────────────────────────────────────────────────────────
# Variables
# ──────────────────────────────────────────────────────────────────────────────

BINARY_NAME=kb-11-population-health
MAIN_PATH=./cmd/server
BUILD_DIR=./bin
DOCKER_IMAGE=kb-11-population-health
VERSION?=1.0.0
GO_FILES=$(shell find . -name '*.go' -type f)

# Database settings (defaults for local development)
DATABASE_URL?=postgres://postgres:password@localhost:5432/kb_population_health?sslmode=disable
REDIS_URL?=redis://localhost:6379

# External service URLs
KB18_URL?=http://localhost:8018
KB13_URL?=http://localhost:8013
KB17_URL?=http://localhost:8017

# Server settings
PORT?=8011
LOG_LEVEL?=info

# ──────────────────────────────────────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────────────────────────────────────

help: ## Display this help message
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════════════════╗"
	@echo "║            KB-11 Population Health Engine - Development Commands              ║"
	@echo "╚══════════════════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build & Run:"
	@awk 'BEGIN {FS = ":.*##"; section=""} /^##@/ {section=substr($$0,5)} /^[a-zA-Z_-]+:.*##/ {if(section=="Build & Run") printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Testing:"
	@awk 'BEGIN {FS = ":.*##"; section=""} /^##@/ {section=substr($$0,5)} /^[a-zA-Z_-]+:.*##/ {if(section=="Testing") printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Code Quality:"
	@awk 'BEGIN {FS = ":.*##"; section=""} /^##@/ {section=substr($$0,5)} /^[a-zA-Z_-]+:.*##/ {if(section=="Code Quality") printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Database:"
	@awk 'BEGIN {FS = ":.*##"; section=""} /^##@/ {section=substr($$0,5)} /^[a-zA-Z_-]+:.*##/ {if(section=="Database") printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Docker:"
	@awk 'BEGIN {FS = ":.*##"; section=""} /^##@/ {section=substr($$0,5)} /^[a-zA-Z_-]+:.*##/ {if(section=="Docker") printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

##@ Build & Run

deps: ## Download Go dependencies
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy

build: deps ## Build the binary
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s -X main.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Built $(BUILD_DIR)/$(BINARY_NAME)"

run: build ## Build and run the server
	@echo "🚀 Starting KB-11 Population Health Engine on port $(PORT)..."
	DATABASE_URL="$(DATABASE_URL)" \
	REDIS_URL="$(REDIS_URL)" \
	KB18_URL="$(KB18_URL)" \
	KB13_URL="$(KB13_URL)" \
	KB17_URL="$(KB17_URL)" \
	PORT=$(PORT) \
	LOG_LEVEL=$(LOG_LEVEL) \
	$(BUILD_DIR)/$(BINARY_NAME)

run-dev: ## Run with hot-reload (requires air: go install github.com/air-verse/air@latest)
	@echo "🔄 Starting KB-11 with hot-reload..."
	DATABASE_URL="$(DATABASE_URL)" \
	REDIS_URL="$(REDIS_URL)" \
	KB18_URL="$(KB18_URL)" \
	KB13_URL="$(KB13_URL)" \
	KB17_URL="$(KB17_URL)" \
	PORT=$(PORT) \
	LOG_LEVEL=debug \
	air

##@ Testing

test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	@echo "🧪 Running unit tests..."
	go test -v -race -short ./internal/... ./pkg/...

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	go test -v -race ./tests/integration/...

test-determinism: ## Run determinism verification tests (critical for KB-18 compliance)
	@echo "🔐 Running determinism tests (CRITICAL for governance)..."
	go test -v -race ./tests/integration/... -run "Determinism"

test-coverage: ## Run tests with coverage report
	@echo "📊 Running tests with coverage..."
	@mkdir -p coverage
	go test -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "✅ Coverage report: coverage/coverage.html"
	@go tool cover -func=coverage/coverage.out | tail -1

test-benchmark: ## Run benchmark tests
	@echo "⏱️ Running benchmarks..."
	go test -bench=. -benchmem ./internal/risk/...

##@ Code Quality

lint: ## Run linters
	@echo "🔍 Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./...; \
	else \
		echo "⚠️  golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi

fmt: ## Format code
	@echo "✨ Formatting code..."
	gofmt -s -w $(GO_FILES)
	@echo "✅ Code formatted"

vet: ## Run go vet
	@echo "🔬 Running go vet..."
	go vet ./...

check: lint vet test-unit ## Run all checks (lint, vet, unit tests)
	@echo "✅ All checks passed"

##@ Database

migrate: ## Run database migrations
	@echo "📝 Running migrations..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f"; \
	done
	@echo "✅ Migrations complete"

migrate-down: ## Rollback last migration (requires manual implementation)
	@echo "⚠️  Manual rollback required - check migrations/*.sql for DOWN statements"

seed: ## Seed database with test data
	@echo "🌱 Seeding database..."
	@if [ -f scripts/seed.sql ]; then \
		psql "$(DATABASE_URL)" -f scripts/seed.sql; \
		echo "✅ Seeding complete"; \
	else \
		echo "⚠️  No seed file found at scripts/seed.sql"; \
	fi

##@ Docker

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

docker-run: ## Run Docker container
	@echo "🐳 Starting Docker container..."
	docker run -d \
		--name $(BINARY_NAME) \
		-p $(PORT):$(PORT) \
		-e DATABASE_URL="$(DATABASE_URL)" \
		-e REDIS_URL="$(REDIS_URL)" \
		-e KB18_URL="$(KB18_URL)" \
		-e KB13_URL="$(KB13_URL)" \
		-e KB17_URL="$(KB17_URL)" \
		-e PORT=$(PORT) \
		$(DOCKER_IMAGE):latest

docker-stop: ## Stop Docker container
	@echo "🛑 Stopping Docker container..."
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

docker-logs: ## View Docker container logs
	docker logs -f $(BINARY_NAME)

##@ Utilities

health: ## Check service health
	@echo "❤️ Checking health..."
	@curl -s http://localhost:$(PORT)/health | jq . || echo "Service not running"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf coverage
	go clean -testcache
	@echo "✅ Clean complete"

generate: ## Run go generate
	@echo "⚙️ Running go generate..."
	go generate ./...

# ──────────────────────────────────────────────────────────────────────────────
# Development Workflow
# ──────────────────────────────────────────────────────────────────────────────

dev-setup: deps ## Set up development environment
	@echo "🛠️ Setting up development environment..."
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest || true
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest || true
	@echo "✅ Development environment ready"

# ──────────────────────────────────────────────────────────────────────────────
# CI/CD Targets
# ──────────────────────────────────────────────────────────────────────────────

ci: check test-coverage ## Run CI pipeline (lint, test, coverage)
	@echo "✅ CI pipeline complete"

release: clean build docker-build ## Create release artifacts
	@echo "📦 Release artifacts created"
	@echo "  Binary: $(BUILD_DIR)/$(BINARY_NAME)"
	@echo "  Docker: $(DOCKER_IMAGE):$(VERSION)"
