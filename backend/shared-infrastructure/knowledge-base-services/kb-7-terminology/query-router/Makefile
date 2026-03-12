.PHONY: help build run test clean docker-build docker-run deps fmt lint

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the query router binary"
	@echo "  run          - Run the query router locally"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  deps         - Download dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  health       - Check service health"

# Build the binary
build:
	@echo "Building KB7 Query Router..."
	go build -o bin/query-router ./cmd/main.go

# Run locally
run:
	@echo "Starting KB7 Query Router..."
	go run ./cmd/main.go

# Run with environment file
run-env:
	@echo "Starting KB7 Query Router with .env file..."
	set -a && source .env && set +a && go run ./cmd/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t kb7-query-router:latest .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -p 8087:8087 -p 8088:8088 --env-file .env kb7-query-router:latest

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Generate mocks (if using mockery)
generate-mocks:
	@echo "Generating mocks..."
	@if command -v mockery >/dev/null 2>&1; then \
		mockery --all --output ./mocks; \
	else \
		echo "mockery not installed. Run: go install github.com/vektra/mockery/v2@latest"; \
	fi

# Security scan
security-scan:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Check service health
health:
	@echo "Checking service health..."
	@curl -s http://localhost:8087/health | jq . || echo "Service not running or jq not installed"

# Check service readiness
ready:
	@echo "Checking service readiness..."
	@curl -s http://localhost:8087/ready | jq . || echo "Service not running or jq not installed"

# Get metrics
metrics:
	@echo "Getting service metrics..."
	@curl -s http://localhost:8087/api/v1/metrics | jq . || echo "Service not running or jq not installed"

# Get Prometheus metrics
prometheus-metrics:
	@echo "Getting Prometheus metrics..."
	@curl -s http://localhost:8088/metrics

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@cp .env.example .env
	@echo "Development environment ready!"
	@echo "1. Update .env with your configuration"
	@echo "2. Run 'make run' to start the service"

# Production build
prod-build:
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/query-router ./cmd/main.go

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Default environment variables
export API_PORT ?= 8087
export METRICS_PORT ?= 8088
export LOG_LEVEL ?= info