# KB-2 Clinical Context Go Service Makefile

.PHONY: help build test run clean docker-build docker-run deps lint format dev install

# Variables
SERVICE_NAME=kb-2-clinical-context-go
DOCKER_IMAGE=kb2-clinical-context
DOCKER_TAG=latest
PORT=8088
GO_VERSION=1.21

# Help target
help: ## Show this help message
	@echo 'Usage: make [TARGET]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
deps: ## Download dependencies
	go mod download
	go mod tidy

install: ## Install the service binary
	go install ./cmd/server

build: ## Build the service binary
	CGO_ENABLED=0 go build -a -installsuffix cgo -o bin/$(SERVICE_NAME) ./cmd/server

build-linux: ## Build for Linux
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/$(SERVICE_NAME)-linux ./cmd/server

run: ## Run the service locally
	go run ./cmd/server

dev: ## Run the service in development mode with hot reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Installing air for hot reload..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Testing targets
test: ## Run tests
	go test -v ./...

test-race: ## Run tests with race detection
	go test -race -v ./...

test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Quality targets
lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
		golangci-lint run; \
	fi

format: ## Format Go code
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
		goimports -w .; \
	fi

vet: ## Run go vet
	go vet ./...

# Docker targets
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run -d \
		--name $(SERVICE_NAME) \
		-p $(PORT):$(PORT) \
		-e DATABASE_URL=mongodb://host.docker.internal:27017 \
		-e REDIS_URL=host.docker.internal:6379 \
		-e ENVIRONMENT=development \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-run-it: ## Run Docker container interactively
	docker run -it --rm \
		--name $(SERVICE_NAME) \
		-p $(PORT):$(PORT) \
		-e DATABASE_URL=mongodb://host.docker.internal:27017 \
		-e REDIS_URL=host.docker.internal:6379 \
		-e ENVIRONMENT=development \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-stop: ## Stop Docker container
	docker stop $(SERVICE_NAME) || true
	docker rm $(SERVICE_NAME) || true

docker-logs: ## View Docker container logs
	docker logs -f $(SERVICE_NAME)

# Knowledge base targets
load-phenotypes: ## Load phenotype definitions (placeholder)
	@echo "Loading phenotype definitions..."
	@echo "Implementation needed to load from knowledge-base/phenotypes/"

load-risk-models: ## Load risk models (placeholder)
	@echo "Loading risk models..."
	@echo "Implementation needed to load from knowledge-base/risk-models/"

load-treatment-prefs: ## Load treatment preferences (placeholder)
	@echo "Loading treatment preferences..."
	@echo "Implementation needed to load from knowledge-base/treatment-preferences/"

# Infrastructure targets
start-deps: ## Start development dependencies (MongoDB, Redis)
	docker-compose -f docker-compose.dev.yml up -d mongodb redis

stop-deps: ## Stop development dependencies
	docker-compose -f docker-compose.dev.yml down

logs-deps: ## View dependency logs
	docker-compose -f docker-compose.dev.yml logs -f

# Health and monitoring targets
health: ## Check service health
	@echo "Checking service health..."
	@curl -f http://localhost:$(PORT)/health || echo "Service not responding"

ready: ## Check service readiness
	@echo "Checking service readiness..."
	@curl -f http://localhost:$(PORT)/ready || echo "Service not ready"

metrics: ## View Prometheus metrics
	@curl -s http://localhost:$(PORT)/metrics | head -20

# API testing targets
test-phenotypes: ## Test phenotype evaluation endpoint
	@echo "Testing phenotype evaluation..."
	@curl -X POST http://localhost:$(PORT)/v1/phenotypes/evaluate \
		-H "Content-Type: application/json" \
		-d '{"patients":[{"id":"test-123","age":65,"gender":"male","conditions":["diabetes","hypertension"]}]}'

test-risk: ## Test risk assessment endpoint
	@echo "Testing risk assessment..."
	@curl -X POST http://localhost:$(PORT)/v1/risk/assess \
		-H "Content-Type: application/json" \
		-d '{"patient_id":"test-123","patient_data":{"id":"test-123","age":65,"gender":"male","conditions":["diabetes"]}}'

test-treatment: ## Test treatment preferences endpoint
	@echo "Testing treatment preferences..."
	@curl -X POST http://localhost:$(PORT)/v1/treatment/preferences \
		-H "Content-Type: application/json" \
		-d '{"patient_id":"test-123","condition":"diabetes","patient_data":{"id":"test-123","age":65,"gender":"male","conditions":["diabetes"]}}'

test-context: ## Test context assembly endpoint
	@echo "Testing context assembly..."
	@curl -X POST http://localhost:$(PORT)/v1/context/assemble \
		-H "Content-Type: application/json" \
		-d '{"patient_id":"test-123","patient_data":{"id":"test-123","age":65,"gender":"male","conditions":["diabetes"]}}'

# Performance testing targets
load-test: ## Run basic load test
	@if command -v hey >/dev/null 2>&1; then \
		hey -n 1000 -c 10 -m GET http://localhost:$(PORT)/health; \
	else \
		echo "Installing hey for load testing..."; \
		go install github.com/rakyll/hey@latest; \
		hey -n 1000 -c 10 -m GET http://localhost:$(PORT)/health; \
	fi

stress-test: ## Run stress test on phenotype evaluation
	@if command -v hey >/dev/null 2>&1; then \
		hey -n 500 -c 50 -m POST \
		-H "Content-Type: application/json" \
		-d '{"patients":[{"id":"test-123","age":65,"gender":"male","conditions":["diabetes"]}]}' \
		http://localhost:$(PORT)/v1/phenotypes/evaluate; \
	else \
		echo "Please install hey: go install github.com/rakyll/hey@latest"; \
	fi

# Security targets
security-scan: ## Run security scan with gosec
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "Installing gosec..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Documentation targets
docs: ## Generate API documentation
	@echo "API documentation available at: http://localhost:$(PORT)/v1/docs"
	@echo "OpenAPI spec available at: ./api/openapi.yaml"

swagger-ui: ## Run Swagger UI (requires Docker)
	docker run -d --name swagger-ui -p 8080:8080 \
		-e SWAGGER_JSON=/openapi.yaml \
		-v $(PWD)/api/openapi.yaml:/openapi.yaml \
		swaggerapi/swagger-ui
	@echo "Swagger UI available at: http://localhost:8080"

stop-swagger: ## Stop Swagger UI
	docker stop swagger-ui || true
	docker rm swagger-ui || true

# Cleanup targets
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache
	go clean -modcache

clean-docker: ## Clean Docker resources
	docker stop $(SERVICE_NAME) || true
	docker rm $(SERVICE_NAME) || true
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true

# Release targets
version: ## Show current version
	@echo "Service: $(SERVICE_NAME)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Docker Image: $(DOCKER_IMAGE):$(DOCKER_TAG)"

release: clean build test ## Build and test for release
	@echo "Release build complete"

# All-in-one targets
all: deps build test lint ## Run all quality checks
	@echo "All checks passed"

quick-start: start-deps build run ## Quick start for development
	@echo "Service started at http://localhost:$(PORT)"

full-test: test test-race test-coverage benchmark ## Run complete test suite
	@echo "Full test suite completed"

# Example usage
example: ## Show example usage
	@echo "Examples:"
	@echo "  make quick-start  # Start dependencies and service"
	@echo "  make test-api     # Test all API endpoints"
	@echo "  make load-test    # Run performance tests"
	@echo "  make clean        # Clean up build artifacts"
	@echo ""
	@echo "Health check: curl http://localhost:$(PORT)/health"
	@echo "API docs: http://localhost:$(PORT)/v1/docs"

test-api: test-phenotypes test-risk test-treatment test-context ## Test all API endpoints