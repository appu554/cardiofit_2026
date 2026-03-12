# KB-1 Drug Dosing Rules Service - Makefile
# Provides convenient commands for development and deployment

.PHONY: help build run test docker-build docker-up docker-down docker-logs health clean

# Default target
.DEFAULT_GOAL := help

# Variables
SERVICE_NAME := kb1-drug-rules
DOCKER_COMPOSE := docker-compose -f docker-compose.kb1.yml
GO := go
PORT := 8081

# ============================================================================
# HELP
# ============================================================================
help: ## Show this help message
	@echo "KB-1 Drug Dosing Rules Service - Available Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# ============================================================================
# LOCAL DEVELOPMENT
# ============================================================================
build: ## Build the Go binary locally
	@echo "📦 Building KB-1 service..."
	$(GO) build -o bin/kb-drug-rules ./cmd/server
	@echo "✅ Build complete: bin/kb-drug-rules"

run: ## Run the service locally (requires DATABASE_URL and REDIS_URL)
	@echo "🚀 Starting KB-1 service on port $(PORT)..."
	$(GO) run ./cmd/server/main.go

run-local: docker-infra ## Run service locally with Docker infrastructure
	@echo "🚀 Starting KB-1 service with Docker infra..."
	DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5481/kb_drug_rules?sslmode=disable \
	REDIS_URL=redis://localhost:6382/0 \
	$(GO) run ./cmd/server/main.go

test: ## Run all tests
	@echo "🧪 Running tests..."
	$(GO) test ./... -v

test-dosing: ## Run dosing package tests only
	@echo "🧪 Running dosing tests..."
	$(GO) test ./internal/dosing/... -v

test-clinical: ## Run clinical scenario tests
	@echo "🧪 Running clinical scenario tests..."
	$(GO) test ./internal/dosing/... -run "Scenario|PrintAll" -v

test-coverage: ## Run tests with coverage report
	@echo "📊 Running tests with coverage..."
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# ============================================================================
# DOCKER OPERATIONS
# ============================================================================
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(SERVICE_NAME):latest .
	@echo "✅ Image built: $(SERVICE_NAME):latest"

docker-up: ## Start all services with Docker Compose
	@echo "🐳 Starting KB-1 services..."
	$(DOCKER_COMPOSE) up -d
	@echo "✅ Services started"
	@make docker-status

docker-up-dev: ## Start all services including dev tools (Adminer)
	@echo "🐳 Starting KB-1 services with dev tools..."
	$(DOCKER_COMPOSE) --profile dev up -d
	@echo "✅ Services started with Adminer at http://localhost:8181"

docker-infra: ## Start only infrastructure (PostgreSQL, Redis) without the app
	@echo "🐳 Starting infrastructure services..."
	$(DOCKER_COMPOSE) up -d kb1-postgres kb1-redis
	@echo "⏳ Waiting for services to be healthy..."
	@sleep 5
	@make docker-status

docker-down: ## Stop all services
	@echo "🛑 Stopping KB-1 services..."
	$(DOCKER_COMPOSE) down
	@echo "✅ Services stopped"

docker-down-clean: ## Stop all services and remove volumes
	@echo "🧹 Stopping services and removing volumes..."
	$(DOCKER_COMPOSE) down -v
	@echo "✅ Services stopped and volumes removed"

docker-logs: ## Show logs for all services
	$(DOCKER_COMPOSE) logs -f

docker-logs-app: ## Show logs for the KB-1 app only
	$(DOCKER_COMPOSE) logs -f kb-drug-rules

docker-status: ## Show status of all services
	@echo "📊 Service Status:"
	@$(DOCKER_COMPOSE) ps

docker-restart: docker-down docker-up ## Restart all services

# ============================================================================
# DATABASE OPERATIONS
# ============================================================================
db-connect: ## Connect to PostgreSQL using psql
	@echo "🔌 Connecting to PostgreSQL..."
	PGPASSWORD=kb_password psql -h localhost -p 5481 -U kb_drug_rules_user -d kb_drug_rules

db-reset: ## Reset database (drop and recreate)
	@echo "🔄 Resetting database..."
	$(DOCKER_COMPOSE) stop kb1-postgres
	$(DOCKER_COMPOSE) rm -f kb1-postgres
	docker volume rm kb1_postgres_data 2>/dev/null || true
	$(DOCKER_COMPOSE) up -d kb1-postgres
	@echo "⏳ Waiting for database to initialize..."
	@sleep 5
	@echo "✅ Database reset complete"

db-migrate: ## Run database migrations manually
	@echo "📝 Running migrations..."
	$(GO) run ./cmd/migrate/main.go up

# ============================================================================
# HEALTH & MONITORING
# ============================================================================
health: ## Check health of all services
	@echo "🏥 Checking service health..."
	@echo ""
	@echo "KB-1 Service:"
	@curl -s http://localhost:$(PORT)/health | jq . 2>/dev/null || echo "  ❌ Not responding"
	@echo ""
	@echo "PostgreSQL:"
	@$(DOCKER_COMPOSE) exec -T kb1-postgres pg_isready -U kb_drug_rules_user -d kb_drug_rules && echo "  ✅ Healthy" || echo "  ❌ Not healthy"
	@echo ""
	@echo "Redis:"
	@$(DOCKER_COMPOSE) exec -T kb1-redis redis-cli ping && echo "  ✅ Healthy" || echo "  ❌ Not healthy"

metrics: ## Show Prometheus metrics
	@curl -s http://localhost:$(PORT)/metrics

ready: ## Check readiness endpoint
	@curl -s http://localhost:$(PORT)/ready | jq .

# ============================================================================
# API TESTING
# ============================================================================
test-api-health: ## Test health endpoint
	@echo "Testing /health endpoint..."
	@curl -s http://localhost:$(PORT)/health | jq .

test-api-rules: ## List all drug rules
	@echo "Testing /v1/rules endpoint..."
	@curl -s http://localhost:$(PORT)/v1/rules | jq .

test-api-calculate: ## Test dose calculation (Lisinopril example)
	@echo "Testing /v1/calculate endpoint..."
	@curl -s -X POST http://localhost:$(PORT)/v1/calculate \
		-H "Content-Type: application/json" \
		-d '{"rxnorm_code": "8610", "patient": {"age": 65, "gender": "M", "weight_kg": 75, "height_cm": 175, "serum_creatinine": 1.2}}' \
		| jq .

test-api-bsa: ## Test BSA calculation
	@echo "Testing /v1/patient/bsa endpoint..."
	@curl -s -X POST http://localhost:$(PORT)/v1/patient/bsa \
		-H "Content-Type: application/json" \
		-d '{"height_cm": 175, "weight_kg": 75}' \
		| jq .

# ============================================================================
# CLEANUP
# ============================================================================
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

clean-all: clean docker-down-clean ## Clean everything including Docker volumes
	@echo "✅ Full cleanup complete"

# ============================================================================
# DEVELOPMENT WORKFLOW
# ============================================================================
dev: docker-infra run-local ## Start infrastructure and run app locally

dev-full: docker-up-dev ## Start all services including dev tools

quick-test: test-dosing test-api-health ## Quick test: dosing tests + health check
