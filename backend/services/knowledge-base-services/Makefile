# Knowledge Base Services Makefile

.PHONY: help build test clean run-dev run-prod docker-build docker-run stop logs

# Default target
help:
	@echo "Available targets:"
	@echo "  build         - Build all services"
	@echo "  test          - Run all tests"
	@echo "  test-coverage - Run tests with coverage"
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
	@echo "  health        - Check service health"
	@echo "  lint          - Run linters"
	@echo "  format        - Format code"

# Build targets
build:
	@echo "Building KB-Drug-Rules service..."
	cd kb-drug-rules && go build -o bin/kb-drug-rules ./cmd/server
	@echo "Build complete!"

build-all:
	@echo "Building all services..."
	$(MAKE) build
	@echo "All services built!"

# Test targets
test:
	@echo "Running tests for KB-Drug-Rules..."
	cd kb-drug-rules && go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	cd kb-drug-rules && go test -v -coverprofile=coverage.out ./...
	cd kb-drug-rules && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: kb-drug-rules/coverage.html"

test-integration:
	@echo "Running integration tests..."
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
	@echo "Checking service health..."
	@curl -f http://localhost:8081/health || echo "KB-Drug-Rules: UNHEALTHY"
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
	@echo "  KB-Drug-Rules: http://localhost:8081"
	@echo "  Health Check:  http://localhost:8081/health"
	@echo "  Metrics:       http://localhost:8081/metrics"
	@echo "  Prometheus:    http://localhost:9090"
	@echo "  Grafana:       http://localhost:3000"
	@echo "  Adminer:       http://localhost:8080"
