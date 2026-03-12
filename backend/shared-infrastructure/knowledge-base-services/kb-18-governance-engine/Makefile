# KB-18 Governance Engine Makefile
# Clinical Governance Enforcement Platform

.PHONY: help build run test clean docker docker-build docker-run lint fmt vet \
        up down logs ps restart db-shell redis-shell

# Variables
APP_NAME := kb-18-governance-engine
PORT := 8018
DOCKER_IMAGE := $(APP_NAME):latest
GO := go
DC := docker-compose

# Default target
.DEFAULT_GOAL := help

## help: Show this help message
help:
	@echo "╔═══════════════════════════════════════════════════════════════════╗"
	@echo "║           KB-18 Governance Engine - Makefile                      ║"
	@echo "║         Clinical Governance Enforcement Platform                  ║"
	@echo "╚═══════════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^##/ {comment = substr($$0, 4)} /^[a-zA-Z_-]+:/ && comment {split($$1, a, ":"); printf "  %-15s %s\n", a[1], comment; comment=""}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make build       # Build the binary"
	@echo "  make run         # Run locally"
	@echo "  make test        # Run all tests"
	@echo "  make docker-run  # Run in Docker"

## build: Build the binary
build:
	@echo "🔨 Building $(APP_NAME)..."
	$(GO) build -o bin/$(APP_NAME) ./cmd/server/main.go
	@echo "✅ Build complete: bin/$(APP_NAME)"

## run: Run the service locally
run: build
	@echo "🚀 Starting $(APP_NAME) on port $(PORT)..."
	PORT=$(PORT) ENVIRONMENT=development LOG_LEVEL=debug ./bin/$(APP_NAME)

## test: Run all tests
test:
	@echo "🧪 Running tests..."
	$(GO) test -v ./tests/... -cover
	@echo "✅ Tests complete"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "🧪 Running tests with coverage..."
	$(GO) test -v ./tests/... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

## lint: Run linter
lint:
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...
	@echo "✅ Lint complete"

## fmt: Format code
fmt:
	@echo "📝 Formatting code..."
	$(GO) fmt ./...
	@echo "✅ Format complete"

## vet: Run go vet
vet:
	@echo "🔎 Running go vet..."
	$(GO) vet ./...
	@echo "✅ Vet complete"

## clean: Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

## deps: Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✅ Dependencies downloaded"

## docker-build: Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

## docker-run: Run in Docker container
docker-run: docker-build
	@echo "🐳 Running $(APP_NAME) in Docker..."
	docker run -d \
		--name $(APP_NAME) \
		-p $(PORT):$(PORT) \
		-e PORT=$(PORT) \
		-e ENVIRONMENT=production \
		-e LOG_LEVEL=info \
		$(DOCKER_IMAGE)
	@echo "✅ Container started: $(APP_NAME)"
	@echo "📍 Access at: http://localhost:$(PORT)"

## docker-stop: Stop Docker container
docker-stop:
	@echo "🛑 Stopping $(APP_NAME) container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true
	@echo "✅ Container stopped"

## docker-logs: Show Docker logs
docker-logs:
	docker logs -f $(APP_NAME)

## health: Check service health
health:
	@echo "🏥 Checking health..."
	@curl -s http://localhost:$(PORT)/health | jq . || echo "Service not running"

## ready: Check service readiness
ready:
	@echo "🔍 Checking readiness..."
	@curl -s http://localhost:$(PORT)/ready | jq . || echo "Service not running"

## stats: Get engine statistics
stats:
	@echo "📊 Getting engine statistics..."
	@curl -s http://localhost:$(PORT)/api/v1/stats | jq . || echo "Service not running"

## programs: List governance programs
programs:
	@echo "📋 Listing governance programs..."
	@curl -s http://localhost:$(PORT)/api/v1/programs | jq . || echo "Service not running"

## test-evaluate: Test medication evaluation
test-evaluate:
	@echo "🧪 Testing medication evaluation..."
	@curl -s -X POST http://localhost:$(PORT)/api/v1/evaluate/medication \
		-H "Content-Type: application/json" \
		-d '{"patient_id":"P001","patient_context":{"patient_id":"P001","age":28,"sex":"F","is_pregnant":true,"gestational_age":12},"medication_order":{"medication_code":"MTX","medication_name":"Methotrexate","drug_class":"METHOTREXATE","dose":15,"dose_unit":"mg"},"requestor_id":"DR001"}' | jq .

## all: Build, test, and lint
all: deps fmt vet lint test build
	@echo "✅ All checks passed"

# =============================================================================
# DOCKER COMPOSE COMMANDS
# =============================================================================

## up: Start all services with Docker Compose
up:
	@echo "🚀 Starting KB-18 Governance Engine stack..."
	$(DC) up -d --build
	@echo "✅ Stack started successfully"
	@echo ""
	@echo "📍 Services:"
	@echo "   KB-18 API:      http://localhost:$(PORT)"
	@echo "   PostgreSQL:     localhost:5418"
	@echo "   Redis:          localhost:6318"
	@echo ""
	@echo "Run 'make logs' to view logs"

## up-dev: Start all services including dev tools (Adminer)
up-dev:
	@echo "🚀 Starting KB-18 Governance Engine stack with dev tools..."
	$(DC) --profile dev up -d --build
	@echo "✅ Stack started successfully"
	@echo ""
	@echo "📍 Services:"
	@echo "   KB-18 API:      http://localhost:$(PORT)"
	@echo "   PostgreSQL:     localhost:5418"
	@echo "   Redis:          localhost:6318"
	@echo "   Adminer (DB UI): http://localhost:8118"

## down: Stop all services
down:
	@echo "🛑 Stopping KB-18 stack..."
	$(DC) down
	@echo "✅ Stack stopped"

## down-clean: Stop all services and remove volumes
down-clean:
	@echo "🧹 Stopping KB-18 stack and cleaning volumes..."
	$(DC) down -v --remove-orphans
	@echo "✅ Stack stopped and volumes removed"

## logs: Show logs from all services
logs:
	$(DC) logs -f

## logs-kb18: Show logs from KB-18 service only
logs-kb18:
	$(DC) logs -f kb-18-governance-engine

## ps: Show running containers
ps:
	$(DC) ps

## restart: Restart all services
restart: down up

## restart-kb18: Restart only KB-18 service
restart-kb18:
	@echo "🔄 Restarting KB-18 service..."
	$(DC) restart kb-18-governance-engine
	@echo "✅ KB-18 restarted"

## rebuild: Rebuild and restart KB-18 service
rebuild:
	@echo "🔨 Rebuilding KB-18 service..."
	$(DC) up -d --build kb-18-governance-engine
	@echo "✅ KB-18 rebuilt and restarted"

## db-shell: Open PostgreSQL shell
db-shell:
	@echo "🐘 Connecting to PostgreSQL..."
	docker exec -it kb18-postgres psql -U kb18user -d kb_governance

## redis-shell: Open Redis CLI
redis-shell:
	@echo "📦 Connecting to Redis..."
	docker exec -it kb18-redis redis-cli

## db-migrate: Run database migrations
db-migrate:
	@echo "📊 Running database migrations..."
	docker exec -i kb18-postgres psql -U kb18user -d kb_governance < migrations/001_initial_schema.sql
	@echo "✅ Migrations complete"

## db-reset: Reset database (drop and recreate)
db-reset:
	@echo "⚠️  Resetting database..."
	docker exec -it kb18-postgres psql -U kb18user -d postgres -c "DROP DATABASE IF EXISTS kb_governance;"
	docker exec -it kb18-postgres psql -U kb18user -d postgres -c "CREATE DATABASE kb_governance;"
	$(MAKE) db-migrate
	@echo "✅ Database reset complete"

# =============================================================================
# TESTING COMMANDS
# =============================================================================

## test-api: Test API endpoints
test-api: health
	@echo ""
	@echo "📋 Testing API endpoints..."
	@echo ""
	@echo "1. List Programs:"
	@curl -s http://localhost:$(PORT)/api/v1/programs | jq -r '.total, .programs[0:2]' || echo "Failed"
	@echo ""
	@echo "2. Get Stats:"
	@curl -s http://localhost:$(PORT)/api/v1/stats | jq '.engine' || echo "Failed"
	@echo ""
	@echo "✅ API tests complete"

## test-maternal: Test maternal safety evaluation
test-maternal:
	@echo "🧪 Testing Maternal Safety - Methotrexate in Pregnancy..."
	@curl -s -X POST http://localhost:$(PORT)/api/v1/evaluate/medication \
		-H "Content-Type: application/json" \
		-d '{ \
			"patient_id": "P001", \
			"patient_context": { \
				"patient_id": "P001", \
				"age": 28, \
				"sex": "F", \
				"is_pregnant": true, \
				"gestational_age": 12, \
				"registry_memberships": [{"registry_code": "PREGNANCY", "status": "ACTIVE"}] \
			}, \
			"medication_order": { \
				"medication_code": "MTX", \
				"medication_name": "Methotrexate", \
				"drug_class": "METHOTREXATE", \
				"dose": 15, \
				"dose_unit": "mg", \
				"frequency": "weekly" \
			}, \
			"requestor_id": "DR001", \
			"requestor_role": "PHYSICIAN" \
		}' | jq '{ outcome, is_approved, violations: [.violations[]? | {rule_code, severity, enforcement_level, message}] }'

## test-opioid: Test opioid stewardship evaluation
test-opioid:
	@echo "🧪 Testing Opioid Stewardship - ER Opioid to Naive Patient..."
	@curl -s -X POST http://localhost:$(PORT)/api/v1/evaluate/medication \
		-H "Content-Type: application/json" \
		-d '{ \
			"patient_id": "P002", \
			"patient_context": { \
				"patient_id": "P002", \
				"age": 55, \
				"sex": "F", \
				"is_opioid_naive": true, \
				"registry_memberships": [{"registry_code": "OPIOID_NAIVE", "status": "ACTIVE"}] \
			}, \
			"medication_order": { \
				"medication_code": "OXY-ER", \
				"medication_name": "OxyContin Extended Release", \
				"drug_class": "OPIOID_ER", \
				"is_er": true, \
				"dose": 20, \
				"dose_unit": "mg", \
				"frequency": "q12h" \
			}, \
			"requestor_id": "DR002", \
			"requestor_role": "PHYSICIAN" \
		}' | jq '{ outcome, is_approved, violations: [.violations[]? | {rule_code, severity, enforcement_level, message}] }'
