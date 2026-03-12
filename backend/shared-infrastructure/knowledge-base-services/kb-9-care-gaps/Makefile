# KB-9 Care Gaps Service Makefile
# Quality Measure Evaluation & Care Gap Detection

.PHONY: help build run test test-unit test-integration lint clean docker docker-up docker-down health

# Default target
help:
	@echo "╔════════════════════════════════════════════════════════════════╗"
	@echo "║       KB-9 Care Gaps Service - Development Commands            ║"
	@echo "╠════════════════════════════════════════════════════════════════╣"
	@echo "║ build             Build the service binary                     ║"
	@echo "║ run               Run the service locally                      ║"
	@echo "║ test              Run all tests                                ║"
	@echo "║ test-unit         Run unit tests only                          ║"
	@echo "║ test-integration  Run integration tests (requires service)     ║"
	@echo "║ lint              Run linter                                   ║"
	@echo "║ clean             Clean build artifacts                        ║"
	@echo "╠════════════════════════════════════════════════════════════════╣"
	@echo "║ docker            Build Docker image                           ║"
	@echo "║ docker-up         Start services with docker-compose           ║"
	@echo "║ docker-down       Stop docker-compose services                 ║"
	@echo "║ docker-logs       Follow docker-compose logs                   ║"
	@echo "╠════════════════════════════════════════════════════════════════╣"
	@echo "║ health            Check service health                         ║"
	@echo "║ deps              Download dependencies                        ║"
	@echo "║ tidy              Run go mod tidy                              ║"
	@echo "╚════════════════════════════════════════════════════════════════╝"

# Build binary
build:
	@echo "🔨 Building KB-9 Care Gaps Service..."
	@mkdir -p bin
	go build -ldflags="-w -s" -o bin/kb-9-care-gaps ./cmd/server
	@echo "✅ Build complete: bin/kb-9-care-gaps"

# Run locally
run: build
	@echo "🚀 Starting KB-9 Care Gaps Service..."
	./bin/kb-9-care-gaps

# Run with environment variables for development
run-dev:
	@echo "🚀 Starting KB-9 in development mode..."
	PORT=8089 \
	ENVIRONMENT=development \
	LOG_LEVEL=debug \
	GOOGLE_CLOUD_PROJECT_ID=$(GOOGLE_CLOUD_PROJECT_ID) \
	GOOGLE_CLOUD_LOCATION=us-central1 \
	GOOGLE_CLOUD_DATASET_ID=$(GOOGLE_CLOUD_DATASET_ID) \
	GOOGLE_CLOUD_FHIR_STORE_ID=$(GOOGLE_CLOUD_FHIR_STORE_ID) \
	KB3_URL=http://localhost:8083 \
	KB3_ENABLED=true \
	REDIS_URL=redis://localhost:6379/9 \
	FEDERATION_ENABLED=true \
	PLAYGROUND_ENABLED=true \
	METRICS_ENABLED=true \
	go run ./cmd/server

# Run all tests
test:
	@echo "🧪 Running all tests..."
	go test -v ./internal/... ./tests/unit/...
	@echo "✅ All tests passed"

# Run unit tests only
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v ./tests/unit/...
	@echo "✅ Unit tests passed"

# Run integration tests (requires service running)
test-integration:
	@echo "🧪 Running integration tests..."
	@echo "   Ensure KB-9 is running on localhost:8089"
	go test -tags=integration -v ./tests/integration/...
	@echo "✅ Integration tests passed"

# Run integration tests with custom URL
test-integration-url:
	@echo "🧪 Running integration tests against $(KB9_BASE_URL)..."
	KB9_BASE_URL=$(KB9_BASE_URL) go test -tags=integration -v ./tests/integration/...

# Code coverage
coverage:
	@echo "📊 Running tests with coverage..."
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Run linter
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi
	@echo "✅ Lint complete"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Download dependencies
deps:
	@echo "📦 Downloading dependencies..."
	go mod download
	@echo "✅ Dependencies downloaded"

# Tidy modules
tidy:
	@echo "📦 Tidying modules..."
	go mod tidy
	@echo "✅ Modules tidied"

# Build Docker image
docker:
	@echo "🐳 Building Docker image..."
	docker build -t kb-9-care-gaps:latest .
	@echo "✅ Docker image built: kb-9-care-gaps:latest"

# Start with docker-compose
docker-up:
	@echo "🐳 Starting services with docker-compose..."
	docker-compose up -d
	@echo "✅ Services started"
	@echo "   KB-9: http://localhost:8089"
	@echo "   Redis: localhost:6389"

# Start with FHIR server
docker-up-fhir:
	@echo "🐳 Starting services with FHIR server..."
	docker-compose --profile fhir up -d
	@echo "✅ Services started (including HAPI FHIR)"

# Stop docker-compose
docker-down:
	@echo "🐳 Stopping docker-compose services..."
	docker-compose down
	@echo "✅ Services stopped"

# Follow docker-compose logs
docker-logs:
	docker-compose logs -f

# Check service health
health:
	@echo "🏥 Checking KB-9 health..."
	@curl -s http://localhost:8089/health | jq . || echo "⚠️  Service not responding"

# Check all endpoints
check-endpoints:
	@echo "🔍 Checking KB-9 endpoints..."
	@echo "\n📍 Health:"
	@curl -s http://localhost:8089/health | jq .
	@echo "\n📍 Ready:"
	@curl -s http://localhost:8089/ready | jq .
	@echo "\n📍 Measures:"
	@curl -s http://localhost:8089/api/v1/measures | jq .

# Quick development cycle
dev: build run

# Full CI pipeline
ci: deps lint test build

# Generate GraphQL schema documentation
schema-docs:
	@echo "📄 GraphQL schema available at: api/schema.graphql"
	@wc -l api/schema.graphql
