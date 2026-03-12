# KB-14 Care Navigator & Tasking Engine
# Makefile for development and operations

.PHONY: all build run test clean docker-build docker-up docker-down \
        migrate-up migrate-down lint fmt vet deps help \
        health integration-test coverage db-start db-stop \
        redis-start redis-stop dev dev-down

# Default target
all: build

# Variables
APP_NAME := kb-14-care-navigator
MAIN_PATH := ./cmd/server
BINARY := bin/$(APP_NAME)
DOCKER_IMAGE := kb-14-care-navigator:latest
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
export PORT ?= 8091
export DATABASE_URL ?= postgres://kb14user:kb14password@localhost:5438/kb_care_navigator?sslmode=disable
export REDIS_URL ?= redis://localhost:6386/0
export KB3_TEMPORAL_URL ?= http://localhost:8087
export KB3_ENABLED ?= true
export KB9_CARE_GAPS_URL ?= http://localhost:8089
export KB9_ENABLED ?= true
export KB12_ORDER_SETS_URL ?= http://localhost:8090
export KB12_ENABLED ?= true
export WORKERS_ENABLED ?= true
export ESCALATION_CHECK_INTERVAL ?= 60
export SYNC_INTERVAL_MINUTES ?= 5
export LOG_LEVEL ?= debug
export ENVIRONMENT ?= development

#==============================================================================
# BUILD
#==============================================================================

build: ## Build the application binary
	@echo "Building $(APP_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) $(MAIN_PATH)
	@echo "Binary built: $(BINARY)"

build-linux: ## Build for Linux (for Docker)
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY)-linux $(MAIN_PATH)
	@echo "Binary built: $(BINARY)-linux"

#==============================================================================
# RUN
#==============================================================================

run: ## Run the application locally
	@echo "Starting $(APP_NAME) on port $(PORT)..."
	$(GORUN) $(MAIN_PATH)

run-binary: build ## Run the compiled binary
	@echo "Starting $(APP_NAME) binary..."
	./$(BINARY)

#==============================================================================
# DOCKER
#==============================================================================

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "Docker image built: $(DOCKER_IMAGE)"

docker-up: ## Start all services with Docker Compose
	@echo "Starting Docker services..."
	docker-compose -f $(COMPOSE_FILE) up -d
	@echo "Services started. Use 'make docker-logs' to view logs."

docker-up-dev: ## Start all services including dev tools (Adminer)
	@echo "Starting Docker services with dev tools..."
	docker-compose -f $(COMPOSE_FILE) --profile dev up -d
	@echo "Services started with Adminer at http://localhost:8191"

docker-down: ## Stop Docker Compose services
	@echo "Stopping Docker services..."
	docker-compose -f $(COMPOSE_FILE) down
	@echo "Services stopped."

docker-logs: ## View Docker Compose logs
	docker-compose -f $(COMPOSE_FILE) logs -f

docker-logs-app: ## View only application logs
	docker-compose -f $(COMPOSE_FILE) logs -f kb-14-care-navigator

docker-clean: ## Remove Docker containers, images, and volumes
	@echo "Cleaning Docker resources..."
	docker-compose -f $(COMPOSE_FILE) down -v --rmi local
	@echo "Docker resources cleaned."

docker-rebuild: docker-clean docker-build docker-up ## Rebuild and restart Docker services

docker-status: ## Show status of Docker services
	docker-compose -f $(COMPOSE_FILE) ps

#==============================================================================
# DATABASE
#==============================================================================

db-start: ## Start PostgreSQL container only
	@echo "Starting PostgreSQL on port 5438..."
	docker-compose -f $(COMPOSE_FILE) up -d kb-14-postgres
	@sleep 3
	@echo "PostgreSQL is ready."

db-stop: ## Stop PostgreSQL container
	docker-compose -f $(COMPOSE_FILE) stop kb-14-postgres

migrate-up: ## Run database migrations
	@echo "Running migrations..."
	@for f in ./migrations/*.sql; do \
		echo "Applying $$f..."; \
		PGPASSWORD=kb14password psql -h localhost -p 5438 -U kb14user -d kb_care_navigator -f "$$f"; \
	done
	@echo "Migrations complete."

migrate-status: ## Check migration status
	@echo "Checking tables..."
	@PGPASSWORD=kb14password psql -h localhost -p 5438 -U kb14user -d kb_care_navigator -c "\dt"

db-shell: ## Open PostgreSQL shell
	@PGPASSWORD=kb14password psql -h localhost -p 5438 -U kb14user -d kb_care_navigator

db-reset: ## Reset database (drop and recreate)
	@echo "Resetting database..."
	@PGPASSWORD=kb14password psql -h localhost -p 5438 -U kb14user -d postgres -c "DROP DATABASE IF EXISTS kb_care_navigator;"
	@PGPASSWORD=kb14password psql -h localhost -p 5438 -U kb14user -d postgres -c "CREATE DATABASE kb_care_navigator;"
	@echo "Database reset. Run 'make migrate-up' to apply migrations."

#==============================================================================
# CACHE
#==============================================================================

redis-start: ## Start Redis container only
	@echo "Starting Redis on port 6386..."
	docker-compose -f $(COMPOSE_FILE) up -d kb-14-redis
	@echo "Redis is ready."

redis-stop: ## Stop Redis container
	docker-compose -f $(COMPOSE_FILE) stop kb-14-redis

redis-cli: ## Open Redis CLI
	docker exec -it kb-14-redis redis-cli

redis-flush: ## Flush Redis cache
	docker exec -it kb-14-redis redis-cli FLUSHALL
	@echo "Redis cache flushed."

redis-info: ## Show Redis info
	docker exec -it kb-14-redis redis-cli INFO

#==============================================================================
# TEST
#==============================================================================

test: ## Run unit tests
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

test-short: ## Run short tests only
	$(GOTEST) -v -short ./...

coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

integration-test: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration ./test/...

benchmark: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

#==============================================================================
# QUALITY
#==============================================================================

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Code formatted."

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

check: fmt vet test ## Run all checks

#==============================================================================
# DEPENDENCIES
#==============================================================================

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded."

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) get -u ./...
	$(GOMOD) tidy
	@echo "Dependencies updated."

#==============================================================================
# UTILITIES
#==============================================================================

health: ## Check service health
	@echo "Checking health..."
	@curl -s http://localhost:$(PORT)/health | jq . || echo "Service not responding"

ready: ## Check service readiness
	@echo "Checking readiness..."
	@curl -s http://localhost:$(PORT)/ready | jq . || echo "Service not ready"

sync-all: ## Trigger manual sync from all KB sources
	@echo "Triggering sync from all sources..."
	@curl -s -X POST http://localhost:$(PORT)/api/v1/sync/all | jq .

sync-kb3: ## Trigger sync from KB-3 Temporal
	@curl -s -X POST http://localhost:$(PORT)/api/v1/sync/kb3 | jq .

sync-kb9: ## Trigger sync from KB-9 Care Gaps
	@curl -s -X POST http://localhost:$(PORT)/api/v1/sync/kb9 | jq .

sync-kb12: ## Trigger sync from KB-12 Order Sets
	@curl -s -X POST http://localhost:$(PORT)/api/v1/sync/kb12 | jq .

dashboard: ## Get analytics dashboard
	@curl -s http://localhost:$(PORT)/api/v1/analytics/dashboard | jq .

worklist-overdue: ## Get overdue tasks
	@curl -s http://localhost:$(PORT)/api/v1/worklist/overdue | jq .

worklist-urgent: ## Get urgent tasks
	@curl -s http://localhost:$(PORT)/api/v1/worklist/urgent | jq .

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	$(GOCMD) clean
	@echo "Cleaned."

#==============================================================================
# DEVELOPMENT
#==============================================================================

dev: deps db-start redis-start ## Set up development environment
	@echo "Development environment ready!"
	@echo "   PostgreSQL: localhost:5438"
	@echo "   Redis: localhost:6386"
	@echo "   Run 'make run' to start the service."

dev-down: db-stop redis-stop ## Stop development environment

watch: ## Run with hot reload (requires air)
	air

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/cosmtrek/air@latest
	@echo "Tools installed."

#==============================================================================
# HELP
#==============================================================================

help: ## Show this help message
	@echo "KB-14 Care Navigator & Tasking Engine"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment Variables:"
	@echo "  PORT                     Service port (default: 8091)"
	@echo "  DATABASE_URL             PostgreSQL connection string"
	@echo "  REDIS_URL                Redis connection string"
	@echo "  KB3_TEMPORAL_URL         KB-3 Temporal Service URL"
	@echo "  KB9_CARE_GAPS_URL        KB-9 Care Gaps Service URL"
	@echo "  KB12_ORDER_SETS_URL      KB-12 Order Sets Service URL"
	@echo "  WORKERS_ENABLED          Enable background workers"
	@echo "  LOG_LEVEL                Log level (debug, info, warn, error)"
	@echo ""
	@echo "Quick Start:"
	@echo "  1. make dev              # Start PostgreSQL and Redis"
	@echo "  2. make migrate-up       # Apply database migrations"
	@echo "  3. make run              # Start the service"
	@echo "  4. make health           # Verify service is running"
