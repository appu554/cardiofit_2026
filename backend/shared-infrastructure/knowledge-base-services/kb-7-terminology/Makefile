# KB-7 Terminology Service - Makefile
# Provides comprehensive build, test, and development commands

# Variables
SERVICE_NAME = kb-7-terminology
BINARY_NAME = kb-7-terminology
DOCKER_COMPOSE_TEST = docker-compose.test.yml
DOCKER_COMPOSE_SEMANTIC = docker-compose.semantic.yml
TEST_DB_URL = postgresql://kb_test_user:kb_test_password@localhost:5434/clinical_governance_test
TEST_REDIS_URL = redis://localhost:6381/8
AUDIT_DB_URL = postgresql://kb_audit_user:kb_audit_password@localhost:5435/clinical_audit_test
POLICY_DB_URL = postgresql://kb_policy_user:kb_policy_password@localhost:5436/clinical_policy_test
GRAPHDB_URL = http://localhost:7200
SPARQL_PROXY_URL = http://localhost:8095
SEMANTIC_PROJECT = kb7-semantic

# Default target
.DEFAULT_GOAL := help

## Help
help: ## Display this help message
	@echo "KB-7 Terminology Service - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

## Development
deps: ## Install Go dependencies
	go mod download
	go mod tidy

fmt: ## Format Go code
	go fmt ./...
	goimports -w .

lint: ## Run linters
	golangci-lint run
	go vet ./...

build: ## Build the service binary
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME) ./cmd/server

build-local: ## Build for local development
	go build -o bin/$(BINARY_NAME) ./cmd/server

build-audit: ## Build audit command-line tool
	go build -o bin/audit ./cmd/audit

build-policy: ## Build policy command-line tool
	go build -o bin/policy ./cmd/policy

build-loader: ## Build kb7-loader CLI tool (Phase 5)
	go build -o bin/kb7-loader ./cmd/kb7-loader

build-all: build build-audit build-policy build-loader ## Build all binaries

run: ## Run the service locally
	go run ./cmd/server

## Testing Infrastructure
test-env-up: ## Start Docker test environment
	@echo "Starting Docker test environment..."
	docker-compose -f $(DOCKER_COMPOSE_TEST) up -d
	@echo "Waiting for services to be ready..."
	sleep 10
	@$(MAKE) test-env-wait

test-env-down: ## Stop Docker test environment  
	@echo "Stopping Docker test environment..."
	docker-compose -f $(DOCKER_COMPOSE_TEST) down -v
	docker-compose -f $(DOCKER_COMPOSE_TEST) rm -f

test-env-wait: ## Wait for test environment to be ready
	@echo "Waiting for PostgreSQL to be ready..."
	@timeout=60; \
	while [ $$timeout -gt 0 ]; do \
		if docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T postgres-test pg_isready -h localhost -p 5432 -U kb_test_user; then \
			echo "PostgreSQL is ready!"; \
			break; \
		fi; \
		echo "Waiting for PostgreSQL... ($$timeout seconds remaining)"; \
		sleep 2; \
		timeout=$$((timeout-2)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "PostgreSQL failed to become ready within timeout"; \
		exit 1; \
	fi
	@echo "Waiting for Redis to be ready..."
	@timeout=30; \
	while [ $$timeout -gt 0 ]; do \
		if docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T redis-test redis-cli ping | grep -q PONG; then \
			echo "Redis is ready!"; \
			break; \
		fi; \
		echo "Waiting for Redis... ($$timeout seconds remaining)"; \
		sleep 1; \
		timeout=$$((timeout-1)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "Redis failed to become ready within timeout"; \
		exit 1; \
	fi

test-env-status: ## Check test environment status
	@echo "=== Docker Test Environment Status ==="
	@docker-compose -f $(DOCKER_COMPOSE_TEST) ps
	@echo ""
	@echo "=== PostgreSQL Status ==="
	@docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T postgres-test pg_isready -h localhost -p 5432 -U kb_test_user || echo "PostgreSQL not ready"
	@echo ""
	@echo "=== Redis Status ==="
	@docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T redis-test redis-cli ping || echo "Redis not ready"

## Database Operations
db-init: ## Initialize test database with schema
	@echo "Initializing test database..."
	docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T postgres-test psql -h localhost -p 5432 -U kb_test_user -d clinical_governance_test -f /docker-entrypoint-initdb.d/init-test-db.sql

audit-db-init: ## Initialize audit database with schema
	@echo "Initializing audit database..."
	@if [ -f "scripts/init-audit-db.sql" ]; then \
		PGPASSWORD=kb_audit_password psql -h localhost -p 5435 -U kb_audit_user -d clinical_audit_test -f scripts/init-audit-db.sql; \
	else \
		echo "Error: scripts/init-audit-db.sql not found"; \
		exit 1; \
	fi

policy-db-init: ## Initialize policy database with schema
	@echo "Initializing policy database..."
	@if [ -f "scripts/init-policy-db.sql" ]; then \
		PGPASSWORD=kb_policy_password psql -h localhost -p 5436 -U kb_policy_user -d clinical_policy_test -f scripts/init-policy-db.sql; \
	else \
		echo "Error: scripts/init-policy-db.sql not found"; \
		exit 1; \
	fi

init-all-dbs: db-init audit-db-init policy-db-init ## Initialize all databases

db-migrate: ## Run database migrations
	@echo "Running database migrations..."
	export DATABASE_URL=$(TEST_DB_URL) && go run ./cmd/migrate

db-status: ## Check database schema status
	@echo "Checking database status..."
	docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T postgres-test psql -h localhost -p 5432 -U kb_test_user -d clinical_governance_test -c "SELECT * FROM get_test_status();"

db-reset: ## Reset test database to clean state
	@echo "Resetting test database..."
	docker-compose -f $(DOCKER_COMPOSE_TEST) exec -T postgres-test psql -h localhost -p 5432 -U kb_test_user -d clinical_governance_test -c "DELETE FROM concept_mappings; DELETE FROM value_set_concepts; DELETE FROM terminology_concepts; DELETE FROM value_sets; DELETE FROM terminology_systems;"

## Testing
test-unit: ## Run unit tests only (no Docker required)
	@echo "Running unit tests..."
	go test -v -tags=unit ./tests/unit/...

test-integration: ## Run integration tests (requires Docker)
	@echo "Running integration tests..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) REDIS_URL=$(TEST_REDIS_URL) && \
	go test -v -tags=integration ./tests/integration/...

test-database: ## Run database integration tests only
	@echo "Running database integration tests..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) && \
	go test -v -run TestDatabaseIntegrationSuite ./tests/integration/database_test.go

test-api: ## Run API integration tests only  
	@echo "Running API integration tests..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) REDIS_URL=$(TEST_REDIS_URL) && \
	go test -v -run TestAPIIntegrationSuite ./tests/integration/api_test.go

test-all: test-env-up test-unit test-integration test-env-down ## Run all tests with Docker environment

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) REDIS_URL=$(TEST_REDIS_URL) && \
	go test -v -coverprofile=coverage.out -coverpkg=./... ./tests/... && \
	go tool cover -html=coverage.out -o coverage.html && \
	go tool cover -func=coverage.out

test-benchmark: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) REDIS_URL=$(TEST_REDIS_URL) && \
	go test -v -bench=. -benchmem -run=^$$ ./tests/integration/...

test-policy-engine: ## Run policy engine tests
	@echo "Running policy engine tests..."
	export DATABASE_URL=$(POLICY_DB_URL) REDIS_URL=$(TEST_REDIS_URL) && \
	go test -v -tags=policy ./tests/policy/...

test-audit-system: ## Run audit system tests
	@echo "Running audit system tests..."
	export DATABASE_URL=$(AUDIT_DB_URL) && \
	go test -v -tags=audit ./tests/audit/...

## Test Data Management
test-data-load: ## Load test data into test database
	@echo "Loading test data..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) && \
	go run ./tests/cmd/load-test-data

test-data-clean: ## Clean test data from test database
	@echo "Cleaning test data..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) && \
	go run ./tests/cmd/clean-test-data

test-data-stats: ## Show test data statistics
	@echo "Test data statistics..."
	export TEST_ENV=docker DATABASE_URL=$(TEST_DB_URL) && \
	go run ./tests/cmd/test-data-stats

## Continuous Integration
ci-deps: ## Install CI dependencies
	@echo "Installing CI dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

ci-lint: ## Run all linting checks for CI
	@echo "Running CI linting checks..."
	go fmt ./...
	goimports -w .
	go vet ./...
	golangci-lint run
	go mod tidy

ci-test: ## Run all tests for CI (assumes Docker environment available)
	@echo "Running CI test suite..."
	$(MAKE) test-env-up
	$(MAKE) test-unit
	$(MAKE) test-integration
	$(MAKE) test-coverage
	$(MAKE) test-env-down

ci-build: ## Build for CI
	@echo "Building for CI..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/$(BINARY_NAME) ./cmd/server

## Docker Operations
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(SERVICE_NAME):latest .

docker-build-test: ## Build Docker test image
	@echo "Building Docker test image..."
	docker build -f Dockerfile.test -t $(SERVICE_NAME):test .

docker-run: ## Run service in Docker
	@echo "Running service in Docker..."
	docker run --rm -p 8080:8080 --env-file .env $(SERVICE_NAME):latest

docker-clean: ## Clean Docker resources
	@echo "Cleaning Docker resources..."
	docker system prune -f
	docker volume prune -f

## Development Tools
dev-setup: deps ci-deps ## Set up development environment
	@echo "Development environment setup complete!"

dev-check: fmt lint test-unit ## Quick development checks
	@echo "Development checks complete!"

dev-full-check: test-env-up test-all test-env-down ## Full development validation
	@echo "Full development validation complete!"

## ValueSet Materialization (BUILD TIME - Required before KB-7 starts)
materialize: ## Run full ValueSet materialization (explicit + intensional)
	@echo "Running full ValueSet materialization..."
	@echo "This is a BUILD TIME job - precomputes all expansions for O(1) runtime lookups"
	python3 scripts/kb7-materialize-all.py

materialize-explicit: ## Run explicit ValueSet materialization only (no Neo4j)
	@echo "Running explicit ValueSet materialization..."
	python3 scripts/kb7-materialize-all.py --explicit-only

materialize-intensional: ## Run intensional ValueSet materialization only (requires Neo4j)
	@echo "Running intensional ValueSet materialization..."
	python3 scripts/kb7-materialize-all.py --intensional-only

materialize-dry-run: ## Preview materialization without changes
	@echo "Previewing materialization (dry run)..."
	python3 scripts/kb7-materialize-all.py --dry-run

materialize-status: ## Check materialization status
	@echo "Checking materialization status..."
	@PGPASSWORD=$${POSTGRES_PASSWORD:-password} psql -h $${POSTGRES_HOST:-localhost} -p $${POSTGRES_PORT:-5432} -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-kb_terminology} -c "SELECT * FROM v_materialization_health;" 2>/dev/null || echo "Run migrations first: make migrate"

logs-test: ## Show test environment logs
	docker-compose -f $(DOCKER_COMPOSE_TEST) logs -f

logs-postgres: ## Show PostgreSQL test logs
	docker-compose -f $(DOCKER_COMPOSE_TEST) logs -f postgres-test

logs-redis: ## Show Redis test logs  
	docker-compose -f $(DOCKER_COMPOSE_TEST) logs -f redis-test

## Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f kb-7-terminology-test
	go clean -cache
	go clean -testcache

clean-all: clean test-env-down docker-clean ## Complete cleanup
	@echo "Complete cleanup finished!"

## Quick Commands (shortcuts)
up: test-env-up ## Quick: Start test environment
down: test-env-down ## Quick: Stop test environment  
test: test-all ## Quick: Run all tests
check: dev-check ## Quick: Run development checks

## Clinical Governance
validate-terminology: ## Validate terminology changes
	@echo "Validating terminology changes..."
	go run ./cmd/validate-terminology

validate-fhir: ## Validate FHIR compliance
	@echo "Validating FHIR compliance..."
	go run ./cmd/validate-fhir

check-terminology-conflicts: ## Check for terminology conflicts
	@echo "Checking terminology conflicts..."
	go run ./cmd/check-conflicts

validate-terminology-mappings: ## Validate terminology mappings
	@echo "Validating terminology mappings..."
	go run ./cmd/validate-mappings

check-deprecated-terms: ## Check for deprecated terminologies
	@echo "Checking deprecated terminologies..."
	go run ./cmd/check-deprecated

## Policy Engine Commands
policy-validate: build-policy ## Validate changes against policies
	@echo "Running policy validation..."
	./bin/policy validate --policy-set=clinical-safety --actor=cli-user

policy-validate-pr: build-policy ## Validate PR files against policies
	@echo "Validating PR against clinical policies..."
	@if [ -z "$(PR_FILES)" ]; then \
		echo "Usage: make policy-validate-pr PR_FILES='file1.go,file2.go'"; \
		exit 1; \
	fi
	./bin/policy validate --policy-set=clinical-safety --pr-files="$(PR_FILES)" --actor=github-action

policy-list-rules: build-policy ## List all policy rules
	@echo "Listing policy rules..."
	./bin/policy list-rules --active-only

policy-report: build-policy ## Generate policy evaluation report
	@echo "Generating policy report..."
	./bin/policy report --days=7 --format=text

## Audit System Commands
audit-create-pr: build-audit ## Create audit entry for PR
	@echo "Creating PR audit entry..."
	@if [ -z "$(PR_NUMBER)" ] || [ -z "$(AUTHOR)" ] || [ -z "$(BRANCH)" ]; then \
		echo "Usage: make audit-create-pr PR_NUMBER=123 AUTHOR=user BRANCH=feature-branch"; \
		exit 1; \
	fi
	./bin/audit create-pr-audit --pr-number="$(PR_NUMBER)" --author="$(AUTHOR)" --branch="$(BRANCH)" --safety-score="$(SAFETY_SCORE)" --requires-review="$(REQUIRES_REVIEW)"

audit-list-events: build-audit ## List recent audit events
	@echo "Listing audit events..."
	./bin/audit list-events --limit=20

audit-list-sessions: build-audit ## List audit sessions
	@echo "Listing audit sessions..."
	./bin/audit list-sessions --limit=10

audit-report: build-audit ## Generate audit report
	@echo "Generating audit report..."
	./bin/audit generate-report --days=7 --format=text

## Validation
validate-config: ## Validate service configuration
	@echo "Validating configuration..."
	go run ./cmd/validate-config

validate-schema: ## Validate database schema
	@echo "Validating database schema..."
	export DATABASE_URL=$(TEST_DB_URL) && go run ./cmd/validate-schema

## Phase 3: Semantic Web Infrastructure
semantic-deploy: ## Deploy OntoNext GraphDB and semantic services
	@echo "Deploying KB-7 Semantic Web Infrastructure (Phase 3)..."
	./scripts/deploy-semantic.sh

## KB-7 Container Management
kb7-container-run: ## Build and run KB-7 all-in-one container
	@echo "Building and running KB-7 semantic container..."
	./scripts/run-kb7-container.sh run

kb7-container-build: ## Build KB-7 container image only
	@echo "Building KB-7 container image..."
	./scripts/run-kb7-container.sh build

kb7-container-stop: ## Stop KB-7 container
	@echo "Stopping KB-7 container..."
	./scripts/run-kb7-container.sh stop

kb7-container-status: ## Show KB-7 container status
	@echo "Checking KB-7 container status..."
	./scripts/run-kb7-container.sh status

kb7-container-logs: ## Show KB-7 container logs
	./scripts/run-kb7-container.sh logs

kb7-container-shell: ## Open shell in KB-7 container
	./scripts/run-kb7-container.sh shell

semantic-up: ## Start semantic services
	@echo "Starting semantic services..."
	docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) up -d

semantic-down: ## Stop semantic services
	@echo "Stopping semantic services..."
	docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) down

semantic-status: ## Check semantic services status
	@echo "Semantic services status:"
	@docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) ps

semantic-logs: ## View semantic services logs
	docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) logs -f

semantic-restart: ## Restart semantic services
	@echo "Restarting semantic services..."
	docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) restart

## GraphDB Operations
graphdb-health: ## Check GraphDB health
	@echo "Checking GraphDB health..."
	@curl -f $(GRAPHDB_URL)/rest/repositories >/dev/null 2>&1 && echo "✅ GraphDB is healthy" || echo "❌ GraphDB is not responding"

graphdb-repos: ## List GraphDB repositories
	@echo "GraphDB repositories:"
	@curl -s $(GRAPHDB_URL)/rest/repositories | jq -r '.[] | .id' 2>/dev/null || curl -s $(GRAPHDB_URL)/rest/repositories

graphdb-load-ontology: ## Load core KB-7 ontology into GraphDB
	@echo "Loading core KB-7 ontology..."
	@if [ -f "semantic/ontologies/kb7-core.ttl" ]; then \
		curl -X POST -H "Content-Type: application/x-turtle" \
		-T semantic/ontologies/kb7-core.ttl \
		"$(GRAPHDB_URL)/repositories/kb7-terminology/statements" && \
		echo "✅ Core ontology loaded successfully"; \
	else \
		echo "❌ Core ontology file not found: semantic/ontologies/kb7-core.ttl"; \
	fi

graphdb-clear: ## Clear all data from KB-7 repository
	@echo "⚠️  WARNING: This will clear all data from kb7-terminology repository"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] && \
	curl -X DELETE "$(GRAPHDB_URL)/repositories/kb7-terminology/statements" && \
	echo "✅ Repository cleared" || echo "❌ Operation cancelled"

## SPARQL Operations
sparql-test: ## Test SPARQL endpoint
	@echo "Testing SPARQL endpoint..."
	@curl -s -X POST -H "Content-Type: application/json" \
	-d '{"query": "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 5"}' \
	"$(SPARQL_PROXY_URL)/sparql" | jq . 2>/dev/null || \
	echo "❌ SPARQL endpoint test failed"

sparql-query: ## Execute custom SPARQL query (usage: make sparql-query QUERY="SELECT * WHERE {?s ?p ?o} LIMIT 10")
	@if [ -z "$(QUERY)" ]; then \
		echo "Usage: make sparql-query QUERY=\"SELECT * WHERE {?s ?p ?o} LIMIT 10\""; \
	else \
		curl -s -X POST -H "Content-Type: application/json" \
		-d "{\"query\": \"$(QUERY)\"}" \
		"$(SPARQL_PROXY_URL)/sparql" | jq .; \
	fi

sparql-concepts: ## Query all clinical concepts
	@echo "Querying clinical concepts..."
	@curl -s -X POST -H "Content-Type: application/json" \
	-d '{"query": "PREFIX kb7: <http://cardiofit.ai/kb7/ontology#> SELECT ?concept ?label WHERE { ?concept a kb7:ClinicalConcept . OPTIONAL { ?concept rdfs:label ?label } } LIMIT 20"}' \
	"$(SPARQL_PROXY_URL)/sparql" | jq .

sparql-mappings: ## Query concept mappings
	@echo "Querying concept mappings..."
	@curl -s -X POST -H "Content-Type: application/json" \
	-d '{"query": "PREFIX kb7: <http://cardiofit.ai/kb7/ontology#> SELECT ?mapping ?sourceCode ?targetCode ?confidence WHERE { ?mapping a kb7:ConceptMapping ; kb7:sourceCode ?sourceCode ; kb7:targetCode ?targetCode ; kb7:mappingConfidence ?confidence } LIMIT 20"}' \
	"$(SPARQL_PROXY_URL)/sparql" | jq .

## ROBOT Tool Operations
robot-validate: ## Validate ontologies using ROBOT
	@echo "Running ontology validation with ROBOT..."
	@docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) exec robot-service python3 scripts/validate_ontologies.py

robot-convert: ## Convert RDF formats using ROBOT (usage: make robot-convert INPUT=file.owl OUTPUT=file.ttl)
	@if [ -z "$(INPUT)" ] || [ -z "$(OUTPUT)" ]; then \
		echo "Usage: make robot-convert INPUT=file.owl OUTPUT=file.ttl"; \
	else \
		docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) exec robot-service python3 scripts/convert_rdf.py $(INPUT) $(OUTPUT); \
	fi

robot-reason: ## Run reasoning pipeline using ROBOT
	@echo "Running reasoning pipeline..."
	@docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) exec robot-service python3 scripts/reasoning_pipeline.py

robot-shell: ## Open interactive shell in ROBOT container
	@docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) exec robot-service bash

## RDF/Turtle Operations
rdf-validate: ## Validate Turtle files
	@echo "Validating Turtle files..."
	@find semantic/ontologies -name "*.ttl" -exec echo "Validating {}" \; -exec python3 -c "import rdflib; rdflib.Graph().parse('{}', format='turtle'); print('✅ Valid')" \; 2>/dev/null || echo "❌ Some files have validation errors"

rdf-stats: ## Show RDF statistics
	@echo "RDF Statistics:"
	@echo "Turtle files: $$(find semantic/ontologies -name '*.ttl' | wc -l)"
	@echo "OWL files: $$(find semantic/ontologies -name '*.owl' | wc -l)"
	@echo "Total size: $$(du -sh semantic/ontologies 2>/dev/null | cut -f1 || echo '0')"

## Semantic Reasoning
reason-execute: ## Execute semantic reasoning
	@echo "Executing semantic reasoning..."
	@curl -s -X POST -H "Content-Type: application/json" \
	-d '{"clinical_domain": "medication", "safety_level": "high"}' \
	"$(SPARQL_PROXY_URL)/reasoning/execute" | jq . 2>/dev/null || \
	echo "Reasoning endpoint not available"

reason-clinical-rules: ## Apply clinical reasoning rules
	@echo "Applying clinical reasoning rules..."
	@curl -s -X POST -H "Content-Type: application/json" \
	-d '{"patient_id": "test-001", "clinical_domain": "medication"}' \
	"$(SPARQL_PROXY_URL)/reasoning/clinical-rules" | jq . 2>/dev/null || \
	echo "Clinical rules endpoint not available"

## Monitoring and Diagnostics
semantic-health: ## Check all semantic services health
	@echo "Checking semantic services health..."
	@echo "GraphDB: $$(curl -s -w '%{http_code}' -o /dev/null $(GRAPHDB_URL)/rest/repositories && echo '✅ OK' || echo '❌ DOWN')"
	@echo "SPARQL Proxy: $$(curl -s -w '%{http_code}' -o /dev/null $(SPARQL_PROXY_URL)/health && echo '✅ OK' || echo '❌ DOWN')"
	@echo "Redis: $$(redis-cli -p 6381 ping 2>/dev/null && echo '✅ OK' || echo '❌ DOWN')"

semantic-stats: ## Show semantic infrastructure statistics
	@echo "Semantic Infrastructure Statistics:"
	@echo "Repository size: $$(curl -s '$(GRAPHDB_URL)/rest/repositories/kb7-terminology/size' 2>/dev/null || echo 'N/A')"
	@echo "Active queries: $$(docker stats --no-stream --format '{{.Container}}: {{.CPUPerc}}' $$(docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) ps -q) 2>/dev/null || echo 'N/A')"

## Phase 3 Integration Commands
phase3-setup: semantic-deploy graphdb-load-ontology robot-validate ## Complete Phase 3 setup
	@echo "✅ Phase 3: Semantic Web Infrastructure setup complete"
	@echo "📊 GraphDB Workbench: $(GRAPHDB_URL)"
	@echo "🔍 SPARQL Proxy: $(SPARQL_PROXY_URL)"

phase3-test: semantic-health sparql-test rdf-stats ## Test Phase 3 functionality
	@echo "✅ Phase 3 testing complete"

phase3-clean: semantic-down ## Clean Phase 3 resources
	@echo "Cleaning Phase 3 resources..."
	@docker-compose -f $(DOCKER_COMPOSE_SEMANTIC) -p $(SEMANTIC_PROJECT) down -v
	@docker system prune -f
	@echo "✅ Phase 3 cleanup complete"

## Documentation
docs-generate: ## Generate documentation
	@echo "Generating documentation..."
	go run ./cmd/generate-docs

docs-serve: ## Serve documentation locally
	@echo "Serving documentation on http://localhost:6060"
	godoc -http=:6060

.PHONY: help deps fmt lint build build-local build-audit build-policy build-all run \
		test-env-up test-env-down test-env-wait test-env-status \
		db-init audit-db-init policy-db-init init-all-dbs db-migrate db-status db-reset \
		test-unit test-integration test-database test-api test-all test-coverage test-benchmark test-policy-engine test-audit-system \
		test-data-load test-data-clean test-data-stats \
		ci-deps ci-lint ci-test ci-build \
		docker-build docker-build-test docker-run docker-clean \
		dev-setup dev-check dev-full-check \
		logs-test logs-postgres logs-redis \
		clean clean-all \
		up down test check \
		validate-terminology validate-fhir check-terminology-conflicts validate-terminology-mappings check-deprecated-terms \
		policy-validate policy-validate-pr policy-list-rules policy-report \
		audit-create-pr audit-list-events audit-list-sessions audit-report \
		validate-config validate-schema \
		semantic-deploy semantic-up semantic-down semantic-status semantic-logs semantic-restart \
		graphdb-health graphdb-repos graphdb-load-ontology graphdb-clear \
		sparql-test sparql-query sparql-concepts sparql-mappings \
		robot-validate robot-convert robot-reason robot-shell \
		rdf-validate rdf-stats reason-execute reason-clinical-rules \
		semantic-health semantic-stats phase3-setup phase3-test phase3-clean \
		docs-generate docs-serve