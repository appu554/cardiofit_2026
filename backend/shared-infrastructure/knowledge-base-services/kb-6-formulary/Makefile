# KB-6 Formulary Management Service Makefile

.PHONY: help build up down logs clean test health status restart seed seed-build migrate build-local run-local

# Default target
help: ## Show this help message
	@echo "KB-6 Formulary Management Service"
	@echo "================================="
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# Docker operations
build: ## Build all Docker images
	@echo "Building KB-6 Formulary Docker images..."
	@docker-compose build --no-cache
	@echo "✅ Build completed"

up: ## Start all services
	@echo "Starting KB-6 Formulary services..."
	@docker-compose up -d
	@echo "✅ Services started"
	@echo "🌐 Endpoints:"
	@echo "  - gRPC API:     http://localhost:8086"
	@echo "  - REST API:     http://localhost:8087"
	@echo "  - Elasticsearch: http://localhost:9200 (elastic/changeme)"
	@echo "  - Kibana:       http://localhost:5601"
	@echo "  - Prometheus:   http://localhost:9090"
	@echo "  - Grafana:      http://localhost:3000 (admin/admin)"

down: ## Stop all services
	@echo "Stopping KB-6 Formulary services..."
	@docker-compose down
	@echo "✅ Services stopped"

restart: ## Restart all services
	@echo "Restarting KB-6 Formulary services..."
	@docker-compose down
	@docker-compose up -d
	@echo "✅ Services restarted"

logs: ## Show logs for all services
	@docker-compose logs -f

logs-kb6: ## Show logs for KB-6 service only
	@docker-compose logs -f kb6-formulary

logs-es: ## Show Elasticsearch logs
	@docker-compose logs -f elasticsearch

logs-db: ## Show database logs
	@docker-compose logs -f postgres

# Health and status
health: ## Check health of all services
	@echo "Checking KB-6 Formulary service health..."
	@echo ""
	@echo "🔍 Service Health Checks:"
	@curl -s http://localhost:8087/health | jq '.' || echo "❌ KB-6 service not responding"
	@echo ""
	@echo "🔍 Infrastructure Health Checks:"
	@docker-compose ps
	@echo ""

status: ## Show service status
	@echo "KB-6 Formulary Service Status"
	@echo "============================="
	@docker-compose ps
	@echo ""
	@echo "Docker Images:"
	@docker images | grep kb6 || echo "No KB-6 images found"
	@echo ""
	@echo "Docker Volumes:"
	@docker volume ls | grep kb6 || echo "No KB-6 volumes found"

# Local Development
build-local: ## Build Go binary locally
	@echo "Building KB-6 Formulary binary..."
	@go build -o bin/kb6-formulary ./main.go
	@echo "✅ Binary built: bin/kb6-formulary"

run-local: ## Run locally (requires local DB/Redis/Elasticsearch)
	@echo "Starting KB-6 Formulary locally..."
	@go run ./main.go

build-seeder: ## Build seeder binary locally
	@echo "Building seeder binary..."
	@go build -o bin/kb6-seeder ./cmd/seeder/main.go
	@echo "✅ Seeder built: bin/kb6-seeder"

# Development operations
dev-up: ## Start services in development mode
	@echo "Starting KB-6 in development mode..."
	@docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d
	@echo "✅ Development services started"

test: ## Run tests
	@echo "Running KB-6 Formulary tests..."
	@go test ./... -v -race -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Tests completed. Coverage report: coverage.html"

test-integration: ## Run integration tests with Docker
	@echo "Running integration tests..."
	@docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit
	@docker-compose -f docker-compose.test.yml down -v

# Data operations
seed: ## Seed database with production formulary data
	@echo "Seeding KB-6 with production data..."
	@docker compose --profile seeder up kb6-seeder
	@echo "✅ Production data loaded"

seed-build: ## Build and run seeder
	@echo "Building and running seeder..."
	@docker compose build kb6-seeder
	@docker compose --profile seeder up kb6-seeder
	@echo "✅ Seeder completed"

migrate: ## Run database migrations
	@echo "Running KB-6 database migrations..."
	@for file in ./migrations/*.sql; do \
		echo "Applying $$file..."; \
		docker compose exec -T kb6-postgres psql -U kb6_admin -d kb_formulary -f /migrations/$$(basename $$file); \
	done
	@echo "✅ Migrations completed"

backup: ## Backup database and Elasticsearch data
	@echo "Creating backup of KB-6 data..."
	@mkdir -p ./backups
	@docker-compose exec postgres pg_dump -U postgres kb_formulary > ./backups/kb6-db-$$(date +%Y%m%d_%H%M%S).sql
	@docker-compose exec elasticsearch curl -X POST "localhost:9200/_snapshot/kb6_backup/snapshot_$$(date +%Y%m%d_%H%M%S)?wait_for_completion=true"
	@echo "✅ Backup completed"

restore: ## Restore database from backup (provide BACKUP_FILE)
	@if [ -z "$(BACKUP_FILE)" ]; then echo "❌ Please provide BACKUP_FILE=path/to/backup.sql"; exit 1; fi
	@echo "Restoring KB-6 database from $(BACKUP_FILE)..."
	@docker-compose exec -T postgres psql -U postgres kb_formulary < $(BACKUP_FILE)
	@echo "✅ Database restored"

# Maintenance operations
clean: ## Clean up containers, images, and volumes
	@echo "Cleaning up KB-6 resources..."
	@docker-compose down -v --rmi local --remove-orphans
	@docker system prune -f
	@echo "✅ Cleanup completed"

clean-data: ## Remove all data volumes (destructive!)
	@echo "⚠️  This will delete all KB-6 data. Are you sure? [y/N]"
	@read -r REPLY; \
	if [ "$$REPLY" = "y" ] || [ "$$REPLY" = "Y" ]; then \
		docker-compose down -v; \
		docker volume rm $$(docker volume ls -q | grep kb6) 2>/dev/null || true; \
		echo "✅ All data volumes removed"; \
	else \
		echo "❌ Cancelled"; \
	fi

reset: ## Full reset - rebuild and restart everything
	@echo "Performing full reset of KB-6 Formulary..."
	@$(MAKE) down
	@$(MAKE) clean
	@$(MAKE) build
	@$(MAKE) up
	@echo "✅ Full reset completed"

# Monitoring operations
monitor: ## Open monitoring dashboards
	@echo "Opening monitoring dashboards..."
	@echo "🔍 Grafana: http://localhost:3000 (admin/admin)"
	@echo "📊 Prometheus: http://localhost:9090"
	@echo "🔍 Kibana: http://localhost:5601"
	@echo "📈 Elasticsearch: http://localhost:9200/_cat/health"

metrics: ## Show current metrics
	@echo "KB-6 Formulary Metrics"
	@echo "====================="
	@echo ""
	@echo "Service Health:"
	@curl -s http://localhost:8087/health | jq '.status' || echo "Service unavailable"
	@echo ""
	@echo "Elasticsearch Health:"
	@curl -s -u elastic:changeme http://localhost:9200/_cluster/health | jq '.status' || echo "Elasticsearch unavailable"
	@echo ""
	@echo "Redis Info:"
	@docker-compose exec redis redis-cli info | grep "used_memory_human" || echo "Redis unavailable"

# Security operations
security-scan: ## Run security scan on Docker images
	@echo "Running security scan on KB-6 images..."
	@docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/app -w /app \
		aquasec/trivy image kb6-formulary:latest
	@echo "✅ Security scan completed"

# Documentation
docs: ## Generate API documentation
	@echo "Generating KB-6 API documentation..."
	@curl -s http://localhost:8087/api/v1/docs | jq '.' > ./docs/api-docs.json
	@echo "✅ API documentation saved to ./docs/api-docs.json"

# Environment information
env-info: ## Show environment information
	@echo "KB-6 Formulary Environment Information"
	@echo "====================================="
	@echo "Docker Version: $$(docker --version)"
	@echo "Docker Compose Version: $$(docker-compose --version)"
	@echo "Go Version: $$(go version 2>/dev/null || echo 'Go not installed')"
	@echo ""
	@echo "System Resources:"
	@docker system df
	@echo ""
	@echo "Active Networks:"
	@docker network ls | grep kb6