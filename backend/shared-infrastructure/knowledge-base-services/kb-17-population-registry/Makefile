.PHONY: help build run test clean docker-build docker-run docker-stop lint tidy

# Default environment variables
export PORT ?= 8017
export ENVIRONMENT ?= development
export LOG_LEVEL ?= debug
export DATABASE_URL ?= postgres://kb17user:kb17password@localhost:5439/kb_population_registry?sslmode=disable
export REDIS_URL ?= redis://localhost:6389/0
export KAFKA_BROKERS ?= localhost:9092
export KAFKA_ENABLED ?= false

# Application
APP_NAME = kb-17-population-registry
BINARY = bin/$(APP_NAME)
MAIN = ./cmd/server

help: ## Show this help
	@echo "KB-17 Population Registry Service"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p bin
	go build -o $(BINARY) $(MAIN)
	@echo "Built: $(BINARY)"

run: build ## Run the application locally
	@echo "Starting $(APP_NAME)..."
	./$(BINARY)

run-dev: ## Run with hot reload (requires air)
	@command -v air > /dev/null 2>&1 || { echo "Installing air..."; go install github.com/cosmtrek/air@latest; }
	air

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	@command -v golangci-lint > /dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	golangci-lint run ./...

tidy: ## Tidy go modules
	go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Cleaned"

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-run: docker-stop ## Run with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker-compose up -d
	@echo "Services started. API available at http://localhost:8017"

docker-run-kafka: docker-stop ## Run with Docker Compose including Kafka
	@echo "Starting services with Kafka..."
	docker-compose --profile kafka up -d
	@echo "Services started with Kafka. API available at http://localhost:8017"

docker-stop: ## Stop Docker Compose services
	@echo "Stopping services..."
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

docker-clean: docker-stop ## Clean Docker resources
	docker-compose down -v --rmi local
	docker system prune -f

# Database targets
db-migrate: ## Run database migrations
	@echo "Running migrations..."
	go run $(MAIN) migrate

db-seed: ## Seed database with default registries
	@echo "Seeding database..."
	go run $(MAIN) seed

db-reset: ## Reset database
	@echo "Resetting database..."
	docker-compose exec postgres psql -U kb17user -d kb_population_registry -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "Database reset. Run 'make db-migrate' to recreate schema."

# Health check
health: ## Check service health
	@curl -s http://localhost:$(PORT)/health | jq .

ready: ## Check service readiness
	@curl -s http://localhost:$(PORT)/ready | jq .

# API tests
api-test-registries: ## List all registries
	@curl -s http://localhost:$(PORT)/api/v1/registries | jq .

api-test-stats: ## Get all registry stats
	@curl -s http://localhost:$(PORT)/api/v1/stats | jq .

api-test-high-risk: ## Get high-risk patients
	@curl -s http://localhost:$(PORT)/api/v1/high-risk | jq .

# Development helpers
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed"

gen-mocks: ## Generate mocks for testing
	@command -v mockgen > /dev/null 2>&1 || { echo "Installing mockgen..."; go install github.com/golang/mock/mockgen@latest; }
	go generate ./...
