# Medication Service V2 - Go/Rust Implementation
# Complete build and development automation

.PHONY: help setup build test clean deploy

# Default target
help: ## Show this help message
	@echo "Medication Service V2 (Go/Rust) - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'
	@echo ""
	@echo "Component Services:"
	@echo "  Go Main Service:        http://localhost:8005"
	@echo "  Go Flow2 Engine:        http://localhost:8085"
	@echo "  Rust Clinical Engine:   http://localhost:8095"
	@echo "  KB Drug Rules:          http://localhost:8086"
	@echo "  KB Guidelines:          http://localhost:8089"
	@echo "  PostgreSQL:             localhost:5434"
	@echo "  Redis:                  localhost:6381"

# =============================================================================
# SETUP & INITIALIZATION
# =============================================================================

setup: ## Initial project setup and component migration
	@echo "Setting up Medication Service V2..."
	@./scripts/setup.sh

migrate-components: ## Copy and update components from V1 service
	@echo "Migrating components from V1..."
	@cp -r ../medication-service/flow2-go-engine ./flow2-go-engine-v2
	@cp -r ../medication-service/flow2-rust-engine ./flow2-rust-engine-v2
	@cp -r ../medication-service/knowledge-bases ./knowledge-bases-v2
	@./scripts/update-ports.sh

validate-config: ## Validate all configuration files
	@echo "Validating configuration..."
	@./scripts/validate-config.sh

# =============================================================================
# BUILD TARGETS
# =============================================================================

build: build-go build-rust build-knowledge-bases build-grpc ## Build all components

build-go: ## Build Go main service and Flow2 engine
	@echo "Building Go components..."
	@go build -ldflags="-w -s" -o bin/medication-service-v2 ./cmd/medication-server
	@go build -ldflags="-w -s" -o bin/http-server ./cmd/http-server
	@go build -ldflags="-w -s" -o bin/grpc-server ./cmd/grpc-server
	@cd flow2-go-engine-v2 && go build -ldflags="-w -s" -o ../bin/flow2-go-engine-v2 ./cmd/engine

build-rust: ## Build Rust clinical engine
	@echo "Building Rust clinical engine..."
	@cd flow2-rust-engine-v2 && cargo build --release
	@cp flow2-rust-engine-v2/target/release/clinical-engine bin/rust-clinical-engine-v2

build-knowledge-bases: ## Build knowledge base services
	@echo "Building knowledge base services..."
	@cd knowledge-bases-v2/kb-drug-rules && go build -o ../../bin/kb-drug-rules .
	@cd knowledge-bases-v2/kb-guideline-evidence && go build -o ../../bin/kb-guidelines .

build-grpc: ## Generate gRPC code from protobuf definitions
	@echo "Generating gRPC code..."
	@mkdir -p proto/medication/v1
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/medication.proto

build-docker: ## Build Docker images
	@echo "Building Docker images..."
	@docker build -t clinical-platform/medication-service-v2:latest -f deployments/Dockerfile.production .
	@docker build -t clinical-platform/flow2-go-engine-v2:latest flow2-go-engine-v2/
	@docker build -t clinical-platform/rust-clinical-engine-v2:latest flow2-rust-engine-v2/
	@docker build -t clinical-platform/kb-drug-rules-v2:latest knowledge-bases-v2/kb-drug-rules/
	@docker build -t clinical-platform/kb-guidelines-v2:latest knowledge-bases-v2/kb-guideline-evidence/

# =============================================================================
# DEVELOPMENT
# =============================================================================

dev: dev-infrastructure ## Start development environment with hot reload
	@echo "Starting development environment..."
	@go run ./cmd/medication-server -config configs/development.yaml

dev-infrastructure: ## Start infrastructure services (DB, Redis, monitoring)
	@echo "Starting infrastructure services..."
	@docker-compose -f deployments/docker-compose.dev.yml up -d postgres-v2 redis-v2 jaeger prometheus grafana
	@echo "Waiting for services to be ready..."
	@sleep 10

run-all: ## Start all services in production mode
	@echo "Starting all Medication Service V2 components..."
	@docker-compose -f deployments/docker-compose.production.yml up -d

run-http-server: ## Run HTTP REST API server
	@echo "Starting HTTP REST API server on port 8080..."
	@./bin/http-server -config config/config.yaml

run-grpc-server: ## Run gRPC API server
	@echo "Starting gRPC API server on port 50051..."
	@./bin/grpc-server -config config/config.yaml

run-go-service: ## Run Go main service only
	@echo "Starting Go medication service on port 8005..."
	@./bin/medication-service-v2 -config configs/service.yaml

run-go-engine: ## Run Flow2 Go engine only
	@echo "Starting Flow2 Go engine on port 8085..."
	@cd flow2-go-engine-v2 && ./bin/flow2-go-engine-v2

run-rust-engine: ## Run Rust clinical engine only
	@echo "Starting Rust clinical engine on port 8095..."
	@./bin/rust-clinical-engine-v2

run-knowledge-bases: ## Run knowledge base services
	@echo "Starting knowledge base services..."
	@./bin/kb-drug-rules -port 8086 &
	@./bin/kb-guidelines -port 8089 &
	@echo "KB services started: Drug Rules (8086), Guidelines (8089)"

stop-all: ## Stop all running services
	@echo "Stopping all services..."
	@docker-compose -f deployments/docker-compose.production.yml down
	@pkill -f "medication-service-v2\|flow2.*engine\|kb-.*" || true

# =============================================================================
# TESTING
# =============================================================================

test: test-unit ## Run all tests (alias for test-unit)

test-unit: ## Run unit tests for all components
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage-go.out ./internal/...
	@cd flow2-go-engine-v2 && go test -v -race ./...
	@cd flow2-rust-engine-v2 && cargo test --release
	@cd knowledge-bases-v2/kb-drug-rules && go test -v ./...
	@cd knowledge-bases-v2/kb-guideline-evidence && go test -v ./...

test-integration: dev-infrastructure ## Run integration tests
	@echo "Running integration tests..."
	@sleep 5  # Wait for infrastructure
	@go test -v -tags=integration ./tests/integration/...

test-performance: dev-infrastructure ## Run performance tests
	@echo "Running performance tests..."
	@go test -v -tags=performance -timeout=10m ./tests/performance/...

test-e2e: run-all ## Run end-to-end tests
	@echo "Running E2E tests..."
	@sleep 30  # Wait for all services
	@go test -v -tags=e2e ./tests/e2e/...
	@make stop-all

test-all: test-unit test-integration test-performance ## Run all test suites

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./internal/...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run performance benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./internal/...
	@cd flow2-rust-engine-v2 && cargo bench

# =============================================================================
# CODE QUALITY
# =============================================================================

lint: lint-go lint-rust ## Run linters for all components

lint-go: ## Run Go linters
	@echo "Running Go linters..."
	@golangci-lint run ./...
	@cd flow2-go-engine-v2 && golangci-lint run ./...
	@cd knowledge-bases-v2/kb-drug-rules && golangci-lint run ./...
	@cd knowledge-bases-v2/kb-guideline-evidence && golangci-lint run ./...

lint-rust: ## Run Rust linters
	@echo "Running Rust linters..."
	@cd flow2-rust-engine-v2 && cargo clippy --all-targets --all-features -- -D warnings

format: format-go format-rust ## Format all code

format-go: ## Format Go code
	@echo "Formatting Go code..."
	@goimports -w .
	@gofmt -s -w .

format-rust: ## Format Rust code
	@echo "Formatting Rust code..."
	@cd flow2-rust-engine-v2 && cargo fmt

audit: ## Run security audits
	@echo "Running security audits..."
	@go list -json -m all | nancy sleuth
	@cd flow2-rust-engine-v2 && cargo audit

quality-check: lint audit test-unit ## Run all quality checks

# =============================================================================
# DATABASE OPERATIONS
# =============================================================================

db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	@./bin/medication-service-v2 -migrate-only

db-reset: ## Reset database (WARNING: destroys data)
	@echo "Resetting database..."
	@docker-compose -f deployments/docker-compose.dev.yml down -v postgres-v2
	@docker-compose -f deployments/docker-compose.dev.yml up -d postgres-v2
	@sleep 5
	@make db-migrate

db-seed: ## Load test data
	@echo "Seeding database with test data..."
	@psql -h localhost -p 5434 -U medication_user -d medication_v2 -f scripts/seed-data.sql

db-backup: ## Create database backup
	@echo "Creating database backup..."
	@./scripts/backup.sh

db-analyze-slow-queries: ## Analyze slow database queries
	@echo "Analyzing slow queries..."
	@psql -h localhost -p 5434 -U medication_user -d medication_v2 -c \
		"SELECT query, mean_time, calls FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# =============================================================================
# DEPLOYMENT
# =============================================================================

deploy-dev: ## Deploy to development environment
	@echo "Deploying to development..."
	@./scripts/deploy.sh dev

deploy-staging: ## Deploy to staging environment
	@echo "Deploying to staging..."
	@./scripts/deploy.sh staging

deploy-production: ## Deploy to production environment
	@echo "Deploying to production..."
	@./scripts/deploy.sh production

k8s-deploy: ## Deploy to Kubernetes
	@echo "Deploying to Kubernetes..."
	@./scripts/k8s-deploy.sh deploy

k8s-rollback: ## Rollback Kubernetes deployment
	@echo "Rolling back Kubernetes deployment..."
	@./scripts/k8s-deploy.sh rollback

blue-green-migration: ## Perform blue-green migration from V1 to V2
	@echo "Starting blue-green migration..."
	@./scripts/blue-green-migration.sh

# =============================================================================
# MONITORING & OBSERVABILITY
# =============================================================================

monitoring-start: ## Start monitoring stack (Prometheus, Grafana, Jaeger, etc.)
	@echo "Starting healthcare monitoring stack..."
	@docker-compose -f docker-compose.monitoring.yml up -d
	@echo "Waiting for services to initialize..."
	@sleep 30
	@echo "✅ Monitoring stack started!"
	@echo "  📊 Grafana:          http://localhost:3000 (admin/admin123)"
	@echo "  📈 Prometheus:       http://localhost:9090"
	@echo "  🚨 Alertmanager:     http://localhost:9093"
	@echo "  🔍 Jaeger:           http://localhost:16686"
	@echo "  📋 Kibana:           http://localhost:5601"

monitoring-stop: ## Stop monitoring stack
	@echo "Stopping monitoring stack..."
	@docker-compose -f docker-compose.monitoring.yml down

monitoring-status: ## Check monitoring stack health
	@echo "Monitoring Stack Health:"
	@echo "  Prometheus:      $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:9090/-/healthy 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Grafana:         $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:3000/api/health 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Alertmanager:    $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:9093/-/healthy 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Jaeger:          $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:16686/ 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Elasticsearch:   $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:9200/_cluster/health 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"

monitoring-logs: ## View monitoring stack logs
	@docker-compose -f docker-compose.monitoring.yml logs -f

monitoring-reset: ## Reset monitoring stack (WARNING: loses data)
	@echo "⚠️  WARNING: This will delete all monitoring data!"
	@read -p "Continue? (y/N): " confirm && [ "$$confirm" = "y" ]
	@docker-compose -f docker-compose.monitoring.yml down -v
	@docker volume prune -f
	@make monitoring-start

monitoring-dashboards: ## Import Grafana dashboards
	@echo "Importing Grafana dashboards..."
	@./scripts/import-dashboards.sh

monitoring-alerts-test: ## Test alerting system
	@echo "Testing alerting system..."
	@curl -X POST http://localhost:9093/api/v1/alerts \
		-H "Content-Type: application/json" \
		-d '[{"labels":{"alertname":"TestAlert","severity":"warning","service":"medication-service-v2"},"annotations":{"description":"Test alert from Makefile"}}]'

health-all: ## Check health of all services including monitoring
	@echo "🏥 Healthcare Service Health Check"
	@echo "=================================="
	@echo "Core Services:"
	@echo "  HTTP API Server (8080):    $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/health/live 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  gRPC API Server (50051):   $(shell grpc_health_probe -addr=localhost:50051 2>/dev/null && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Go Main Service (8005):    $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:8005/health/ready 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Go Flow2 Engine (8085):    $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:8085/health 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Rust Engine (8095):       $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:8095/health 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  KB Drug Rules (8086):     $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:8086/health 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  KB Guidelines (8089):     $(shell curl -s -o /dev/null -w '%{http_code}' http://localhost:8089/health 2>/dev/null | grep -q 200 && echo '✅ UP' || echo '❌ DOWN')"
	@echo ""
	@echo "Infrastructure:"
	@echo "  PostgreSQL (5434):        $(shell pg_isready -h localhost -p 5434 2>/dev/null && echo '✅ UP' || echo '❌ DOWN')"
	@echo "  Redis (6381):             $(shell redis-cli -h localhost -p 6381 ping 2>/dev/null | grep -q PONG && echo '✅ UP' || echo '❌ DOWN')"
	@echo ""
	@make monitoring-status

health-detailed: ## Get detailed health information
	@echo "📋 Detailed Health Report"
	@echo "========================="
	@curl -s http://localhost:8080/health/status | jq . || echo "❌ Detailed health endpoint unavailable"

metrics: ## View service metrics and dashboards
	@echo "📊 Opening metrics dashboards..."
	@echo "  Operational Dashboard:     http://localhost:3000/d/medication-operational"
	@echo "  Clinical Safety Dashboard: http://localhost:3000/d/medication-clinical-safety"
	@echo "  Prometheus:               http://localhost:9090"
	@echo "  Grafana:                  http://localhost:3000"
	@echo "  Jaeger Tracing:           http://localhost:16686"
	@echo "  Log Analysis:             http://localhost:5601"
	@if command -v open >/dev/null 2>&1; then \
		open http://localhost:3000; \
		open http://localhost:9090; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open http://localhost:3000; \
		xdg-open http://localhost:9090; \
	fi

metrics-query: ## Query specific metrics
	@echo "📈 Key Medication Service Metrics:"
	@echo "=================================="
	@echo -n "API Requests (5m rate): "
	@curl -s 'http://localhost:9090/api/v1/query?query=rate(medication_service_http_requests_total[5m])' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "N/A"
	@echo -n "P95 Latency: "
	@curl -s 'http://localhost:9090/api/v1/query?query=histogram_quantile(0.95, rate(medication_service_http_request_duration_seconds_bucket[5m]))' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "N/A"
	@echo -n "Error Rate: "
	@curl -s 'http://localhost:9090/api/v1/query?query=rate(medication_service_errors_total[5m])' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "N/A"
	@echo -n "Safety Violations (1h): "
	@curl -s 'http://localhost:9090/api/v1/query?query=increase(medication_service_safety_violations_total[1h])' | jq -r '.data.result[0].value[1]' 2>/dev/null || echo "N/A"

logs: ## View service logs
	@echo "📜 Viewing service logs..."
	@docker-compose -f deployments/docker-compose.production.yml logs -f

logs-monitoring: ## View monitoring stack logs
	@docker-compose -f docker-compose.monitoring.yml logs -f

logs-go: ## View Go service logs only
	@docker-compose -f deployments/docker-compose.production.yml logs -f medication-service-v2

logs-rust: ## View Rust engine logs only
	@docker-compose -f deployments/docker-compose.production.yml logs -f rust-clinical-engine-v2

logs-clinical: ## View clinical decision logs
	@echo "🏥 Clinical Decision Logs (last 100):"
	@curl -s "http://localhost:9200/medication-service-logs-*/_search" \
		-H "Content-Type: application/json" \
		-d '{"query":{"match":{"event_type":"clinical_decision"}},"size":10,"sort":[{"timestamp":{"order":"desc"}}]}' \
		| jq -r '.hits.hits[]._source | "\(.timestamp) - \(.indication): \(.outcome)"' 2>/dev/null || echo "❌ Elasticsearch unavailable"

logs-safety: ## View safety violation logs
	@echo "🚨 Safety Violation Logs:"
	@curl -s "http://localhost:9200/medication-service-logs-*/_search" \
		-H "Content-Type: application/json" \
		-d '{"query":{"match":{"event_category":"safety_violation"}},"size":10,"sort":[{"timestamp":{"order":"desc"}}]}' \
		| jq -r '.hits.hits[]._source | "\(.timestamp) - \(.violation_type): \(.description)"' 2>/dev/null || echo "❌ No safety violations found"

alerts: ## View active alerts
	@echo "🚨 Active Alerts:"
	@curl -s http://localhost:9093/api/v1/alerts | jq -r '.data[] | select(.state=="active") | "\(.labels.severity | ascii_upcase) - \(.labels.alertname): \(.annotations.description)"' 2>/dev/null || echo "❌ Alertmanager unavailable"

alerts-silence: ## Silence test alerts (requires ALERT_ID)
	@if [ -z "$(ALERT_ID)" ]; then echo "Usage: make alerts-silence ALERT_ID=<alert-id>"; exit 1; fi
	@curl -X POST http://localhost:9093/api/v1/silences \
		-H "Content-Type: application/json" \
		-d '{"matchers":[{"name":"alertname","value":"$(ALERT_ID)"}],"startsAt":"$(shell date -u +%Y-%m-%dT%H:%M:%S.000Z)","endsAt":"$(shell date -u -d '+1 hour' +%Y-%m-%dT%H:%M:%S.000Z)","comment":"Silenced via Makefile"}'

stress-test: ## Run stress test against services
	@echo "🔥 Running stress test..."
	@./scripts/stress-test.sh

load-test: ## Run load test with realistic healthcare traffic
	@echo "📊 Running healthcare load test..."
	@./scripts/load-test.sh

performance-report: ## Generate comprehensive performance report
	@echo "📊 Generating performance report..."
	@./scripts/generate-performance-report.sh

# =============================================================================
# DEVELOPMENT UTILITIES
# =============================================================================

debug: ## Start service with debugger
	@echo "Starting service with debugger..."
	@dlv debug ./cmd/medication-server

profile: ## Profile running service
	@echo "Profiling service..."
	@go tool pprof -http=:6060 http://localhost:8005/debug/pprof/profile

trace: ## Collect execution trace
	@echo "Collecting execution trace..."
	@curl http://localhost:8005/debug/pprof/trace?seconds=10 > trace.out
	@go tool trace trace.out

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f *.out *.html
	@cd flow2-rust-engine-v2 && cargo clean
	@docker system prune -f

clean-all: clean ## Clean everything including Docker volumes
	@echo "Cleaning all artifacts and volumes..."
	@docker-compose -f deployments/docker-compose.production.yml down -v
	@docker volume prune -f

# =============================================================================
# DOCUMENTATION
# =============================================================================

docs: ## Generate API documentation
	@echo "Generating API documentation..."
	@swag init -g ./cmd/http-server/main.go -o ./docs/swagger
	@echo "OpenAPI documentation available at docs/openapi.yaml"

docs-serve: ## Serve documentation locally
	@echo "Serving documentation at http://localhost:8080"
	@cd docs && python3 -m http.server 8080

test-api: ## Test API endpoints
	@echo "Testing HTTP API endpoints..."
	@curl -X GET http://localhost:8080/health
	@curl -X GET http://localhost:8080/api/v1/medication-proposals \
		-H "Authorization: Bearer ${JWT_TOKEN}" || echo "Set JWT_TOKEN environment variable"
	@echo "Testing FHIR endpoints..."
	@curl -X GET http://localhost:8080/fhir/r4/metadata

benchmark-api: ## Benchmark API performance
	@echo "Running API performance benchmarks..."
	@./scripts/api-benchmark.sh

# =============================================================================
# MIGRATION UTILITIES
# =============================================================================

migrate-data: ## Migrate data from V1 to V2 database
	@echo "Migrating data from V1 database..."
	@./scripts/data-migration.sh

verify-migration: ## Verify data migration integrity
	@echo "Verifying migration integrity..."
	@./scripts/verify-migration.sh

compare-apis: ## Compare V1 and V2 API responses
	@echo "Comparing API responses..."
	@./scripts/api-comparison.sh

performance-comparison: ## Compare V1 vs V2 performance
	@echo "Running performance comparison..."
	@./scripts/performance-comparison.sh

# =============================================================================
# SECURITY & COMPLIANCE
# =============================================================================

security-scan: ## Run security vulnerability scan
	@echo "Running security scan..."
	@docker run --rm -v $(PWD):/app -w /app securecodewarrior/docker-image-scanner

compliance-check: ## Check HIPAA compliance
	@echo "Running compliance checks..."
	@./scripts/compliance-check.sh

encrypt-secrets: ## Encrypt sensitive configuration
	@echo "Encrypting secrets..."
	@./scripts/encrypt-secrets.sh

# =============================================================================
# SHORTCUTS AND ALIASES
# =============================================================================

up: run-all ## Alias for run-all
down: stop-all ## Alias for stop-all
start: run-all ## Alias for run-all
restart: stop-all run-all ## Restart all services

# Local development workflow
dev-reset: stop-all clean db-reset dev-infrastructure ## Reset entire dev environment

# CI/CD helpers
ci-test: quality-check test-all ## Run all CI tests
cd-deploy: build-docker deploy-production ## Build and deploy to production

# Performance monitoring
perf: metrics profile ## Open performance monitoring tools