# KB-12 Order Sets & Care Plans Service
# Makefile for development and operations

.PHONY: all build run test clean docker-build docker-up docker-down \
        migrate-up migrate-down seed lint fmt vet deps help \
        health integration-test coverage

# Default target
all: build

# Variables
APP_NAME := kb-12-ordersets
MAIN_PATH := ./cmd/server
BINARY := bin/$(APP_NAME)
DOCKER_IMAGE := kb-12-ordersets:latest
COMPOSE_FILE := docker-compose.yml

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOVET := $(GOCMD) vet
GOFMT := gofmt

# Build settings
LDFLAGS := -ldflags="-w -s -X main.Version=$$(git describe --tags --always --dirty 2>/dev/null || echo 'dev') -X main.BuildTime=$$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Environment variables for local development
export PORT ?= 8092
export DATABASE_URL ?= postgres://kb12user:kb12password@localhost:5437/kb_ordersets?sslmode=disable
export REDIS_URL ?= redis://localhost:6385/0
export KB1_URL ?= http://localhost:8081
export KB3_URL ?= http://localhost:8083
export KB6_URL ?= http://localhost:8086
export KB7_URL ?= http://localhost:8087
export LOG_LEVEL ?= debug
export ENVIRONMENT ?= development

#==============================================================================
# BUILD
#==============================================================================

build: ## Build the application binary
	@echo "🔨 Building $(APP_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) $(MAIN_PATH)
	@echo "✅ Binary built: $(BINARY)"

build-linux: ## Build for Linux (for Docker)
	@echo "🔨 Building $(APP_NAME) for Linux..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY)-linux $(MAIN_PATH)
	@echo "✅ Binary built: $(BINARY)-linux"

#==============================================================================
# RUN
#==============================================================================

run: ## Run the application locally
	@echo "🚀 Starting $(APP_NAME) on port $(PORT)..."
	$(GORUN) $(MAIN_PATH)

run-binary: build ## Run the compiled binary
	@echo "🚀 Starting $(APP_NAME) binary..."
	./$(BINARY)

#==============================================================================
# DOCKER
#==============================================================================

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

docker-up: ## Start all services with Docker Compose
	@echo "🐳 Starting Docker services..."
	docker-compose -f $(COMPOSE_FILE) up -d
	@echo "✅ Services started. Use 'make docker-logs' to view logs."

docker-down: ## Stop Docker Compose services
	@echo "🐳 Stopping Docker services..."
	docker-compose -f $(COMPOSE_FILE) down
	@echo "✅ Services stopped."

docker-logs: ## View Docker Compose logs
	docker-compose -f $(COMPOSE_FILE) logs -f

docker-clean: ## Remove Docker containers, images, and volumes
	@echo "🧹 Cleaning Docker resources..."
	docker-compose -f $(COMPOSE_FILE) down -v --rmi local
	@echo "✅ Docker resources cleaned."

docker-rebuild: docker-clean docker-build docker-up ## Rebuild and restart Docker services

#==============================================================================
# DATABASE
#==============================================================================

db-start: ## Start PostgreSQL container only
	@echo "🐘 Starting PostgreSQL on port 5437..."
	docker-compose -f $(COMPOSE_FILE) up -d kb-12-postgres
	@sleep 3
	@echo "✅ PostgreSQL is ready."

db-stop: ## Stop PostgreSQL container
	docker-compose -f $(COMPOSE_FILE) stop kb-12-postgres

migrate-up: ## Run database migrations
	@echo "📦 Running migrations..."
	@PGPASSWORD=kb12password psql -h localhost -p 5437 -U kb12user -d kb_ordersets -f ./migrations/init.sql
	@echo "✅ Migrations complete."

seed: ## Seed database with template data
	@echo "🌱 Seeding database..."
	$(GORUN) ./cmd/seed/main.go
	@echo "✅ Database seeded."

db-shell: ## Open PostgreSQL shell
	@PGPASSWORD=kb12password psql -h localhost -p 5437 -U kb12user -d kb_ordersets

#==============================================================================
# CACHE
#==============================================================================

redis-start: ## Start Redis container only
	@echo "📕 Starting Redis on port 6385..."
	docker-compose -f $(COMPOSE_FILE) up -d kb-12-redis
	@echo "✅ Redis is ready."

redis-stop: ## Stop Redis container
	docker-compose -f $(COMPOSE_FILE) stop kb-12-redis

redis-cli: ## Open Redis CLI
	docker exec -it kb-12-redis redis-cli

redis-flush: ## Flush Redis cache
	docker exec -it kb-12-redis redis-cli FLUSHALL
	@echo "✅ Redis cache flushed."

#==============================================================================
# TEST
#==============================================================================

test: ## Run unit tests
	@echo "🧪 Running tests..."
	$(GOTEST) -v -race ./...

test-short: ## Run short tests only
	$(GOTEST) -v -short ./...

coverage: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

integration-test: ## Run integration tests
	@echo "🧪 Running integration tests..."
	$(GOTEST) -v -tags=integration ./tests/...

benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

#==============================================================================
# QUALITY
#==============================================================================

lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "✨ Formatting code..."
	$(GOFMT) -s -w .
	@echo "✅ Code formatted."

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	$(GOVET) ./...

check: fmt vet lint test ## Run all checks

#==============================================================================
# DEPENDENCIES
#==============================================================================

deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies downloaded."

deps-update: ## Update dependencies
	@echo "📦 Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy
	@echo "✅ Dependencies updated."

#==============================================================================
# UTILITIES
#==============================================================================

health: ## Check service health
	@echo "❤️ Checking health..."
	@curl -s http://localhost:$(PORT)/health | jq . || echo "Service not responding"

api-docs: ## Generate API documentation
	@echo "📚 Generating API docs..."
	swag init -g cmd/server/main.go -o docs
	@echo "✅ API docs generated in docs/"

templates-count: ## Count templates
	@echo "📊 Template counts:"
	@echo "Order Sets: $$(find pkg/ordersets -name '*.go' | xargs grep -l 'func.*OrderSet' | wc -l)"
	@echo "Care Plans: $$(find pkg/careplans -name '*.go' | xargs grep -l 'func.*CarePlan' | wc -l)"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GOCMD) clean
	@echo "✅ Cleaned."

#==============================================================================
# DEVELOPMENT
#==============================================================================

dev: deps db-start redis-start ## Set up development environment
	@echo "✅ Development environment ready!"
	@echo "   PostgreSQL: localhost:5437"
	@echo "   Redis: localhost:6385"
	@echo "   Run 'make run' to start the service."

dev-down: db-stop redis-stop ## Stop development environment

watch: ## Run with hot reload (requires air)
	air

install-tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/cosmtrek/air@latest
	@echo "✅ Tools installed."

#==============================================================================
# HELP
#==============================================================================

help: ## Show this help message
	@echo "KB-12 Order Sets & Care Plans Service"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment Variables:"
	@echo "  PORT            Service port (default: 8092)"
	@echo "  DATABASE_URL    PostgreSQL connection string"
	@echo "  REDIS_URL       Redis connection string"
	@echo "  KB1_URL         KB-1 Drug Rules service URL"
	@echo "  LOG_LEVEL       Log level (debug, info, warn, error)"
