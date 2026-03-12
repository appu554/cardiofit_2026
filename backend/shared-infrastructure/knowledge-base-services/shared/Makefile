# =============================================================================
# Knowledge Base Services - Shared Infrastructure Makefile
# Purpose: Manage Phase 1 database stack and data ingestion
# =============================================================================

.PHONY: help up down logs migrate health ingest backup restore clean test

# Default target
help:
	@echo "╔═══════════════════════════════════════════════════════════════╗"
	@echo "║     Knowledge Base - Phase 1 Database Stack Commands          ║"
	@echo "╚═══════════════════════════════════════════════════════════════╝"
	@echo ""
	@echo "Database Stack:"
	@echo "  make up         - Start PostgreSQL, Redis, and Adminer"
	@echo "  make down       - Stop all services"
	@echo "  make logs       - View service logs"
	@echo "  make health     - Check health of all services"
	@echo "  make clean      - Stop services and remove volumes (DESTRUCTIVE)"
	@echo ""
	@echo "Migrations:"
	@echo "  make migrate    - Run database migrations"
	@echo "  make migrate-docker - Run migrations via Docker"
	@echo ""
	@echo "Data Ingestion:"
	@echo "  make ingest     - Load all Phase 1 data (dry-run)"
	@echo "  make ingest-live - Load all Phase 1 data (live)"
	@echo ""
	@echo "Backup & Restore:"
	@echo "  make backup     - Create Golden State backup"
	@echo "  make restore    - Restore from Golden State backup"
	@echo ""
	@echo "Utilities:"
	@echo "  make psql       - Open psql shell to database"
	@echo "  make redis-cli  - Open redis-cli shell"
	@echo "  make test       - Run data verification queries"
	@echo ""

# =============================================================================
# Database Stack Management
# =============================================================================

# Start all services
up:
	@echo "Starting Phase 1 database stack..."
	docker-compose -f docker-compose.phase1.yml up -d
	@echo ""
	@echo "Services starting..."
	@echo "  PostgreSQL: localhost:5433"
	@echo "  Redis:      localhost:6380"
	@echo "  Adminer:    http://localhost:8082"
	@echo ""
	@echo "Run 'make migrate' to apply database schema"

# Stop all services
down:
	@echo "Stopping Phase 1 database stack..."
	docker-compose -f docker-compose.phase1.yml down

# View logs
logs:
	docker-compose -f docker-compose.phase1.yml logs -f

# View specific service logs
logs-postgres:
	docker-compose -f docker-compose.phase1.yml logs -f kb-postgres

logs-redis:
	docker-compose -f docker-compose.phase1.yml logs -f kb-redis

# Health check
health:
	@echo "Checking service health..."
	@echo ""
	@echo "PostgreSQL:"
	@docker exec kb-fact-store pg_isready -U kb_admin -d canonical_facts && echo "  ✅ Healthy" || echo "  ❌ Unhealthy"
	@echo ""
	@echo "Redis:"
	@docker exec kb-cache redis-cli ping && echo "  ✅ Healthy" || echo "  ❌ Unhealthy"
	@echo ""

# Clean everything (DESTRUCTIVE)
clean:
	@echo "⚠️  WARNING: This will delete all data!"
	@read -p "Are you sure? (y/N) " confirm && [ "$$confirm" = "y" ]
	docker-compose -f docker-compose.phase1.yml down -v
	@echo "All services stopped and volumes removed."

# =============================================================================
# Migrations
# =============================================================================

# Run migrations (local psql)
migrate:
	@chmod +x scripts/run-migrations.sh
	./scripts/run-migrations.sh --local

# Run migrations via Docker
migrate-docker:
	docker-compose -f docker-compose.phase1.yml --profile migrate up kb-migrate

# =============================================================================
# Data Ingestion
# =============================================================================

# Dry-run ingestion (validate only)
ingest:
	@echo "Running Phase 1 data ingestion (dry-run)..."
	go run ./cmd/phase1-ingest/main.go --all --dry-run

# Live ingestion
ingest-live:
	@echo "Running Phase 1 data ingestion (LIVE)..."
	go run ./cmd/phase1-ingest/main.go --all

# Individual data source ingestion
ingest-onc:
	go run ./cmd/phase1-ingest/main.go --onc

ingest-formulary:
	go run ./cmd/phase1-ingest/main.go --formulary

ingest-loinc:
	go run ./cmd/phase1-ingest/main.go --loinc

# =============================================================================
# Backup & Restore
# =============================================================================

# Create Golden State backup
backup:
	@echo "Creating Golden State backup..."
	@mkdir -p ./cmd/phase1-ingest/backup
	@docker exec kb-fact-store pg_dump -U kb_admin -d canonical_facts \
		--no-owner --no-acl \
		-t interaction_matrix \
		-t formulary_coverage \
		-t lab_reference_ranges \
		-t ingestion_metadata \
		> ./cmd/phase1-ingest/backup/golden_state_$$(date +%Y%m%d_%H%M%S).sql
	@echo "Backup created: ./cmd/phase1-ingest/backup/golden_state_$$(date +%Y%m%d_%H%M%S).sql"

# Restore from Golden State
restore:
	@chmod +x ./cmd/phase1-ingest/backup/restore_golden_state.sh
	./cmd/phase1-ingest/backup/restore_golden_state.sh

# =============================================================================
# Database Utilities
# =============================================================================

# Open psql shell
psql:
	docker exec -it kb-fact-store psql -U kb_admin -d canonical_facts

# Open redis-cli shell
redis-cli:
	docker exec -it kb-cache redis-cli

# =============================================================================
# Testing & Verification
# =============================================================================

# Run verification queries
test:
	@echo "Running data verification queries..."
	@echo ""
	@echo "=== Drug Interactions (KB-5) ==="
	@docker exec kb-fact-store psql -U kb_admin -d canonical_facts -c \
		"SELECT COUNT(*) as total, severity, source_dataset FROM interaction_matrix GROUP BY severity, source_dataset ORDER BY severity;"
	@echo ""
	@echo "=== Formulary Coverage (KB-6) ==="
	@docker exec kb-fact-store psql -U kb_admin -d canonical_facts -c \
		"SELECT COUNT(*) as total, tier_level_code, COUNT(CASE WHEN prior_auth THEN 1 END) as with_pa FROM formulary_coverage GROUP BY tier_level_code ORDER BY tier_level_code;"
	@echo ""
	@echo "=== Lab Reference Ranges (KB-16) ==="
	@docker exec kb-fact-store psql -U kb_admin -d canonical_facts -c \
		"SELECT COUNT(*) as total, clinical_category FROM lab_reference_ranges GROUP BY clinical_category ORDER BY clinical_category;"
	@echo ""
	@echo "=== Ingestion Metadata ==="
	@docker exec kb-fact-store psql -U kb_admin -d canonical_facts -c \
		"SELECT source_name, source_version, records_loaded, load_timestamp FROM ingestion_metadata ORDER BY load_timestamp DESC LIMIT 5;"

# Quick count verification
count:
	@echo "Record counts:"
	@docker exec kb-fact-store psql -U kb_admin -d canonical_facts -tAc \
		"SELECT 'interaction_matrix: ' || COUNT(*) FROM interaction_matrix UNION ALL \
		 SELECT 'formulary_coverage: ' || COUNT(*) FROM formulary_coverage UNION ALL \
		 SELECT 'lab_reference_ranges: ' || COUNT(*) FROM lab_reference_ranges;"

# =============================================================================
# Development Utilities
# =============================================================================

# Format Go code
fmt:
	go fmt ./...

# Run Go tests
go-test:
	go test -v ./...

# Build CLI tool
build:
	go build -o bin/phase1-ingest ./cmd/phase1-ingest/main.go

# Install dependencies
deps:
	go mod download
	go mod tidy
