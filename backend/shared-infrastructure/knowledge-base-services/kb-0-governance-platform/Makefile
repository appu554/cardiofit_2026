# =============================================================================
# KB-0 Governance Platform Makefile
# =============================================================================

.PHONY: help build run stop logs health test clean

# Default target
help:
	@echo "KB-0 Governance Platform - Docker Commands"
	@echo "==========================================="
	@echo ""
	@echo "Docker Operations:"
	@echo "  make build         - Build the Docker image"
	@echo "  make run           - Start KB-0 with PostgreSQL and Redis"
	@echo "  make stop          - Stop all services"
	@echo "  make logs          - View KB-0 service logs"
	@echo "  make health        - Check service health"
	@echo "  make clean         - Remove containers and volumes"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Run locally with go run"
	@echo "  make test          - Run unit tests"
	@echo "  make test-kb1      - Test KB-1 integration endpoints"
	@echo ""
	@echo "Database:"
	@echo "  make db-shell      - Connect to PostgreSQL"
	@echo "  make db-migrate    - Run database migrations"
	@echo ""

# =============================================================================
# Docker Operations
# =============================================================================

# Build the Docker image
build:
	docker-compose build --no-cache kb0-governance

# Start all services (KB-0, PostgreSQL, Redis)
run:
	@echo "Starting KB-0 Governance Platform..."
	docker-compose up -d
	@echo ""
	@echo "Waiting for services to be ready..."
	@sleep 5
	@make health

# Stop all services
stop:
	docker-compose down

# View logs
logs:
	docker-compose logs -f kb0-governance

# Health check
health:
	@echo "Checking KB-0 health..."
	@curl -s http://localhost:8080/health | jq . || echo "KB-0 not responding"
	@echo ""
	@echo "Checking PostgreSQL..."
	@docker-compose exec -T kb0-db pg_isready -U kb0_user -d kb0_governance && echo "PostgreSQL: healthy" || echo "PostgreSQL: not ready"
	@echo ""
	@echo "Checking Redis..."
	@docker-compose exec -T kb0-redis redis-cli ping && echo "Redis: healthy" || echo "Redis: not ready"

# Clean up everything
clean:
	docker-compose down -v
	docker rmi kb-0-governance-platform-kb0-governance 2>/dev/null || true

# =============================================================================
# Development
# =============================================================================

# Run locally (requires PostgreSQL and Redis running)
dev:
	KB0_PORT=8080 \
	KB0_DATABASE_HOST=localhost \
	KB0_DATABASE_PORT=5500 \
	KB0_DATABASE_NAME=kb0_governance \
	KB0_DATABASE_USER=kb0_user \
	KB0_DATABASE_PASSWORD=kb0_secure_password \
	KB1_URL=http://localhost:8081 \
	go run ./cmd/server

# Run unit tests
test:
	go test -v ./...

# Test KB-1 integration endpoints
test-kb1:
	@echo "=== Testing KB-0 → KB-1 Integration ==="
	@echo ""
	@echo "1. Checking KB-0 health..."
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "2. Getting pending drugs from KB-1 via KB-0..."
	@curl -s http://localhost:8080/api/v1/kb1/pending | jq .
	@echo ""
	@echo "=== Integration test complete ==="

# =============================================================================
# Database Operations
# =============================================================================

# Connect to PostgreSQL shell
db-shell:
	docker-compose exec kb0-db psql -U kb0_user -d kb0_governance

# Run migrations (already run on container start)
db-migrate:
	docker-compose exec kb0-db psql -U kb0_user -d kb0_governance -f /docker-entrypoint-initdb.d/001_create_schema.sql

# Check database tables
db-tables:
	@docker-compose exec -T kb0-db psql -U kb0_user -d kb0_governance -c "\dt"

# =============================================================================
# KB-1 Testing Workflows
# =============================================================================

# Full governance workflow test: DRAFT → REVIEWED → ACTIVE
test-workflow:
	@echo "=== KB-0 → KB-1 Governance Workflow Test ==="
	@echo ""
	@echo "Step 1: Get pending drugs from KB-1..."
	@PENDING=$$(curl -s http://localhost:8080/api/v1/kb1/pending); \
	echo $$PENDING | jq '.items[0]'; \
	RULE_ID=$$(echo $$PENDING | jq -r '.items[0].ID'); \
	echo ""; \
	echo "Step 2: Submit pharmacist review for $$RULE_ID..."; \
	curl -s -X POST http://localhost:8080/api/v1/kb1/drugs/$$RULE_ID/review \
		-H "Content-Type: application/json" \
		-d '{"reviewed_by":"Dr. Clinical Pharmacist", "review_notes":"All dosing verified", "checklist":{"dosing_verified":true,"renal_verified":true,"hepatic_verified":true,"interactions_verified":true,"safety_verified":true}}' | jq .; \
	echo ""; \
	echo "Step 3: Submit CMO approval for $$RULE_ID..."; \
	curl -s -X POST http://localhost:8080/api/v1/kb1/drugs/$$RULE_ID/approve \
		-H "Content-Type: application/json" \
		-d '{"approved_by":"Dr. CMO","review_notes":"Approved for clinical use","is_high_risk":true}' | jq .
	@echo ""
	@echo "=== Workflow test complete ==="
