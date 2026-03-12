# Knowledge Base Services Makefile

.PHONY: help build test clean run-dev run-prod docker-build docker-run stop logs

# Default target
help:
	@echo "KB Services Available:"
	@echo "  KB-1: Drug Rules Engine (TOML-based clinical rules)"
	@echo "  KB-2: Clinical Context (MongoDB-based patient phenotypes)"
	@echo "  KB-3: Clinical Guidelines (Evidence-based protocols)"
	@echo "  KB-4: Patient Safety Monitor (TimescaleDB-based alerts)"
	@echo "  KB-5: Drug Interactions (Drug-drug interaction detection)"
	@echo "  KB-6: Formulary Management (PostgreSQL + Elasticsearch)"
	@echo "  KB-7: Clinical Terminology (SNOMED CT, ICD-10, RxNorm, LOINC)"
	@echo "  Evidence Envelope: Audit trail and governance tracking"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build all services"
	@echo "  build-kb-X    - Build specific KB service (X = 1,2,3,4,5,6,7)"
	@echo "  test          - Run all tests"
	@echo "  test-kb-X     - Test specific KB service (X = 1,2,3,4,5,6,7)"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-integration - Run comprehensive integration tests (Python)"
	@echo "  test-integration-bash - Run integration tests (Bash)"
	@echo "  clean         - Clean build artifacts"
	@echo "  run-kb-docker - Start KB service with Docker PostgreSQL (RECOMMENDED)"
	@echo "  stop-kb       - Stop KB Docker services"
	@echo "  logs-kb       - Show KB service logs"
	@echo "  setup-local   - Setup local PostgreSQL database"
	@echo "  run-local     - Run KB service with local PostgreSQL"
	@echo "  run-dev       - Run development environment (Docker)"
	@echo "  run-prod      - Run production environment"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-run    - Run services in Docker"
	@echo "  stop          - Stop all services"
	@echo "  logs          - Show service logs"
	@echo "  migrate       - Run database migrations"
	@echo "  seed          - Seed database with sample data"
	@echo "  health        - Check all KB services health"
	@echo "  lint          - Run linters"
	@echo "  format        - Format code"

# Build targets
build:
	@echo "Building KB-Drug-Rules service..."
	cd kb-drug-rules && go build -o bin/kb-drug-rules ./cmd/server
	@echo "Build complete!"

build-all:
	@echo "Building all services..."
	$(MAKE) build-kb-1
	$(MAKE) build-kb-2
	$(MAKE) build-kb-3
	$(MAKE) build-kb-4
	$(MAKE) build-kb-5
	$(MAKE) build-kb-6
	$(MAKE) build-kb-7
	$(MAKE) build-kb-9
	$(MAKE) build-evidence-envelope
	@echo "All services built!"

build-kb-1:
	@echo "Building KB-1 Drug Rules service..."
	cd kb-drug-rules && go build -o bin/kb-drug-rules ./cmd/server

build-kb-3:
	@echo "Building KB-3 Guidelines service..."
	cd kb-3-guidelines && go build -o bin/kb-guidelines .

build-kb-4:
	@echo "Building KB-4 Patient Safety service..."
	cd kb-4-patient-safety && go build -o bin/kb-patient-safety .

build-kb-5:
	@echo "Building KB-5 Drug Interactions service..."
	cd kb-5-drug-interactions && go build -o bin/kb-drug-interactions .

build-kb-2:
	@echo "Building KB-2 Clinical Context service..."
	cd kb-2-clinical-context && go build -o bin/kb-clinical-context .

build-kb-6:
	@echo "Building KB-6 Formulary Management service..."
	cd kb-6-formulary && go build -o bin/kb-formulary .

build-kb-7:
	@echo "Building KB-7 Terminology service..."
	cd kb-7-terminology && go build -o bin/kb-terminology .

build-evidence-envelope:
	@echo "Building Evidence Envelope service..."
	cd evidence-envelope && go build -o bin/evidence-envelope .

build-kb-9:
	@echo "Building KB-9 Care Gaps service..."
	cd kb-9-care-gaps && go build -o bin/kb-9-care-gaps ./cmd/server

# Test targets
test:
	@echo "Running tests for all KB services..."
	$(MAKE) test-kb-1
	$(MAKE) test-kb-2
	$(MAKE) test-kb-3
	$(MAKE) test-kb-4
	$(MAKE) test-kb-5
	$(MAKE) test-kb-6
	$(MAKE) test-kb-7
	$(MAKE) test-kb-9
	$(MAKE) test-evidence-envelope

test-kb-1:
	@echo "Running tests for KB-1 Drug Rules..."
	cd kb-drug-rules && go test -v ./...

test-kb-3:
	@echo "Running tests for KB-3 Guidelines..."
	cd kb-3-guidelines && go test -v ./...

test-kb-4:
	@echo "Running tests for KB-4 Patient Safety..."
	cd kb-4-patient-safety && go test -v ./...

test-kb-5:
	@echo "Running tests for KB-5 Drug Interactions..."
	cd kb-5-drug-interactions && go test -v ./...

test-kb-2:
	@echo "Running tests for KB-2 Clinical Context..."
	cd kb-2-clinical-context && go test -v ./...

test-kb-6:
	@echo "Running tests for KB-6 Formulary Management..."
	cd kb-6-formulary && go test -v ./...

test-kb-7:
	@echo "Running tests for KB-7 Terminology..."
	cd kb-7-terminology && go test -v ./...

test-kb-9:
	@echo "Running tests for KB-9 Care Gaps..."
	cd kb-9-care-gaps && go test -v ./...

test-evidence-envelope:
	@echo "Running tests for Evidence Envelope..."
	cd evidence-envelope && go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	cd kb-drug-rules && go test -v -coverprofile=coverage.out ./...
	cd kb-drug-rules && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: kb-drug-rules/coverage.html"

test-integration:
	@echo "Running integration tests for all KB services..."
	python3 scripts/test_kb_integration.py

test-integration-bash:
	@echo "Running bash integration tests..."
	chmod +x scripts/test-all-kb-services.sh
	./scripts/test-all-kb-services.sh

test-integration-individual:
	@echo "Running individual service integration tests..."
	cd kb-drug-rules && go test -v -tags=integration ./tests/integration/...

# KB Docker targets (RECOMMENDED)
run-kb-docker:
	@echo "Starting KB-Drug-Rules with Docker PostgreSQL..."
	@echo "This uses isolated containers (PostgreSQL on port 5433, Redis on port 6380)"
	@if command -v docker >/dev/null 2>&1; then \
		chmod +x scripts/start-kb-docker.sh; \
		./scripts/start-kb-docker.sh; \
	else \
		echo "Docker not found. Please install Docker Desktop."; \
		echo "Download from: https://www.docker.com/products/docker-desktop"; \
	fi

stop-kb:
	@echo "Stopping KB Docker services..."
	docker-compose -f docker-compose.kb-only.yml down

logs-kb:
	@echo "Showing KB service logs..."
	docker logs kb-drug-rules -f

test-kb-docker:
	@echo "Testing KB Docker setup..."
	@if command -v docker >/dev/null 2>&1; then \
		chmod +x scripts/test-kb-docker.sh 2>/dev/null || true; \
		if [ -f scripts/test-kb-docker.sh ]; then \
			./scripts/test-kb-docker.sh; \
		else \
			echo "Running basic tests..."; \
			curl -s http://localhost:8081/health && echo "✅ Health check passed"; \
			curl -s http://localhost:8081/v1/items/metformin >/dev/null && echo "✅ Drug rules working"; \
		fi \
	else \
		echo "Docker not found. Please install Docker Desktop."; \
	fi

# Local development targets
setup-local:
	@echo "Setting up local PostgreSQL database..."
	@echo "Choose your platform:"
	@echo "  Linux/macOS: ./scripts/setup-local-postgres.sh"
	@echo "  Windows:     scripts\\setup-local-postgres.bat"
	@echo ""
	@echo "Or run manually:"
	@echo "  psql -U postgres -f scripts/setup-local-postgres.sql"

run-local:
	@echo "Starting KB-Drug-Rules with local PostgreSQL..."
	@if [ ! -f .env.local ]; then \
		echo "Creating default .env.local file..."; \
		echo "DATABASE_URL=postgresql://kb_drug_rules_user:kb_password@localhost:5432/kb_drug_rules" > .env.local; \
		echo "REDIS_URL=redis://localhost:6379/0" >> .env.local; \
		echo "PORT=8081" >> .env.local; \
		echo "DEBUG=true" >> .env.local; \
	fi
	@echo "Starting service..."
	cd kb-drug-rules && \
	export $$(cat ../.env.local | xargs) && \
	go run cmd/server/main.go

# Docker development targets
run-dev:
	@echo "Starting development environment with Docker..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Development environment started!"
	@echo "Services available at:"
	@echo "  KB-Drug-Rules: http://localhost:8081"
	@echo "  PostgreSQL:    localhost:5432 (postgres/password)"
	@echo "  Redis:         localhost:6379"
	@echo "  Prometheus:    http://localhost:9090"
	@echo "  Grafana:       http://localhost:3000 (admin/admin)"
	@echo "  Adminer:       http://localhost:8080"

run-prod:
	@echo "Starting production environment..."
	docker-compose -f docker-compose.prod.yml up -d
	@echo "Production environment started!"

# Docker targets
docker-build:
	@echo "Building Docker images..."
	docker-compose -f docker-compose.dev.yml build
	@echo "Docker images built!"

docker-run: docker-build
	@echo "Running services in Docker..."
	docker-compose -f docker-compose.dev.yml up -d
	@echo "Services are running!"

# Management targets
stop:
	@echo "Stopping all services..."
	docker-compose -f docker-compose.dev.yml down
	docker-compose -f docker-compose.prod.yml down
	@echo "All services stopped!"

logs:
	@echo "Showing service logs..."
	docker-compose -f docker-compose.dev.yml logs -f

logs-kb:
	@echo "Showing KB-Drug-Rules logs..."
	docker-compose -f docker-compose.dev.yml logs -f kb-drug-rules

# Database targets
migrate:
	@echo "Running database migrations..."
	docker-compose -f docker-compose.dev.yml exec db psql -U postgres -d kb_drug_rules -f /docker-entrypoint-initdb.d/init-db.sql
	@echo "Migrations complete!"

migrate-supabase:
	@echo "Setting up Supabase database..."
	@if [ ! -f .env.supabase ]; then \
		echo "Error: .env.supabase file not found!"; \
		echo "Please copy .env.supabase.example to .env.supabase and configure it."; \
		exit 1; \
	fi
	@echo "Please run the following SQL script in your Supabase SQL Editor:"
	@echo "File: scripts/setup-supabase.sql"
	@echo "Or copy and paste the contents from the file into your Supabase dashboard."

seed:
	@echo "Seeding database with sample data..."
	cd kb-drug-rules && go run ./cmd/seed
	@echo "Database seeded!"

# Supabase-specific targets
setup-supabase:
	@echo "Setting up Supabase configuration..."
	@if [ ! -f .env.supabase ]; then \
		cp .env.supabase.example .env.supabase; \
		echo "Created .env.supabase from template."; \
		echo "Please edit .env.supabase with your Supabase credentials."; \
	else \
		echo ".env.supabase already exists."; \
	fi
	@echo ""
	@echo "Next steps:"
	@echo "1. Edit .env.supabase with your Supabase project details"
	@echo "2. Run the SQL script in scripts/setup-supabase.sql in your Supabase SQL Editor"
	@echo "3. Run: make run-dev-supabase"

test-supabase:
	@echo "Testing Supabase connection..."
	@if [ ! -f .env.supabase ]; then \
		echo "Error: .env.supabase file not found!"; \
		exit 1; \
	fi
	@source .env.supabase && curl -f "$$SUPABASE_URL/rest/v1/" \
		-H "apikey: $$SUPABASE_API_KEY" \
		-H "Authorization: Bearer $$SUPABASE_API_KEY" || \
		echo "Supabase connection test failed. Check your credentials."

# Health check targets
health:
	@echo "Checking all KB services health..."
	@curl -f http://localhost:8081/health || echo "KB-1 Drug Rules: UNHEALTHY"
	@curl -f http://localhost:8082/health || echo "KB-2 Clinical Context: UNHEALTHY"
	@curl -f http://localhost:8083/health || echo "KB-3 Guidelines: UNHEALTHY"
	@curl -f http://localhost:8084/health || echo "KB-4 Patient Safety: UNHEALTHY"
	@curl -f http://localhost:8085/health || echo "KB-5 Drug Interactions: UNHEALTHY"
	@curl -f http://localhost:8086/health || echo "KB-6 Formulary Management: UNHEALTHY"
	@curl -f http://localhost:8087/health || echo "KB-7 Terminology: UNHEALTHY"
	@curl -f http://localhost:8088/health || echo "Evidence Envelope: UNHEALTHY"
	@curl -f http://localhost:8089/health || echo "KB-9 Care Gaps: UNHEALTHY"
	@curl -f http://localhost:9090/-/healthy || echo "Prometheus: UNHEALTHY"
	@echo "Health check complete!"

health-detailed:
	@echo "Detailed health check..."
	@echo "KB-Drug-Rules:"
	@curl -s http://localhost:8081/health | jq '.' || echo "  Service unavailable"
	@echo ""
	@echo "Database:"
	@docker-compose -f docker-compose.dev.yml exec -T db pg_isready -U postgres || echo "  Database unavailable"
	@echo ""
	@echo "Redis:"
	@docker-compose -f docker-compose.dev.yml exec -T redis redis-cli ping || echo "  Redis unavailable"

# Code quality targets
lint:
	@echo "Running linters..."
	cd kb-drug-rules && golangci-lint run
	@echo "Linting complete!"

format:
	@echo "Formatting code..."
	cd kb-drug-rules && go fmt ./...
	cd kb-drug-rules && goimports -w .
	@echo "Code formatted!"

# Utility targets
clean:
	@echo "Cleaning build artifacts..."
	cd kb-drug-rules && rm -rf bin/ coverage.out coverage.html
	docker system prune -f
	@echo "Clean complete!"

reset: stop clean
	@echo "Resetting environment..."
	docker-compose -f docker-compose.dev.yml down -v
	docker system prune -af --volumes
	@echo "Environment reset complete!"

# API testing targets
test-api:
	@echo "Testing API endpoints..."
	@echo "Health check:"
	@curl -s http://localhost:8081/health | jq '.'
	@echo ""
	@echo "Get drug rules (metformin):"
	@curl -s "http://localhost:8081/v1/items/metformin" | jq '.'
	@echo ""
	@echo "Validate rules:"
	@curl -s -X POST http://localhost:8081/v1/validate \
		-H "Content-Type: application/json" \
		-d '{"content":"[meta]\ndrug_name=\"Test\"\ntherapeutic_class=[\"Test\"]\n[dose_calculation]\nbase_formula=\"100mg\"\nmax_daily_dose=200.0\nmin_daily_dose=50.0\n[safety_verification]\ncontraindications=[]\nwarnings=[]\nprecautions=[]\ninteraction_checks=[]\nlab_monitoring=[]\nmonitoring_requirements=[]\nregional_variations={}","regions":["US"]}' | jq '.'

test-local-db:
	@echo "Running comprehensive local database tests..."
	python3 scripts/test-local-db.py

test-kb-service:
	@echo "Running KB service test suite..."
	chmod +x scripts/test-kb-service.sh
	./scripts/test-kb-service.sh

# Performance testing
test-performance:
	@echo "Running performance tests..."
	@echo "Load testing KB-Drug-Rules..."
	@ab -n 1000 -c 10 http://localhost:8081/health
	@echo "Performance test complete!"

# Monitoring targets
metrics:
	@echo "Fetching metrics..."
	@curl -s http://localhost:8081/metrics | grep kb_

dashboard:
	@echo "Opening monitoring dashboard..."
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@echo "Installing Go dependencies..."
	cd kb-drug-rules && go mod download
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Development setup complete!"

# Backup and restore
backup:
	@echo "Creating database backup..."
	docker-compose -f docker-compose.dev.yml exec -T db pg_dump -U postgres kb_drug_rules > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Backup created!"

restore:
	@echo "Restoring database from backup..."
	@read -p "Enter backup file name: " backup_file; \
	docker-compose -f docker-compose.dev.yml exec -T db psql -U postgres -d kb_drug_rules < $$backup_file
	@echo "Database restored!"

# Documentation
docs:
	@echo "Generating API documentation..."
	cd kb-drug-rules && swag init -g cmd/server/main.go
	@echo "Documentation generated!"

# Security scanning
security-scan:
	@echo "Running security scan..."
	cd kb-drug-rules && gosec ./...
	@echo "Security scan complete!"

# Environment info
info:
	@echo "Environment Information:"
	@echo "======================="
	@echo "Go version: $(shell go version)"
	@echo "Docker version: $(shell docker --version)"
	@echo "Docker Compose version: $(shell docker-compose --version)"
	@echo ""
	@echo "Running containers:"
	@docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "Service URLs:"
	@echo "  KB-1 Drug Rules:        http://localhost:8081"
	@echo "  KB-2 Clinical Context:  http://localhost:8082"
	@echo "  KB-3 Guidelines:        http://localhost:8083"
	@echo "  KB-4 Patient Safety:    http://localhost:8084"
	@echo "  KB-5 Drug Interactions: http://localhost:8085"
	@echo "  KB-6 Formulary Mgmt:    http://localhost:8086"
	@echo "  KB-7 Terminology:       http://localhost:8087"
	@echo "  Evidence Envelope:      http://localhost:8088"
	@echo "  Prometheus:             http://localhost:9090"
	@echo "  Grafana:                http://localhost:3000"
	@echo "  Adminer:                http://localhost:8080"
