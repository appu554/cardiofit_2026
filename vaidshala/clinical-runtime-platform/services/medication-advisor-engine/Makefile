# Medication Advisor Engine - Tier 6 Application Engine
# =====================================================

.PHONY: help build run test clean docker lint fmt

# Default target
help:
	@echo "Medication Advisor Engine - Tier 6 Application Engine"
	@echo "======================================================"
	@echo ""
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  run         - Build and run the server"
	@echo "  test        - Run all tests"
	@echo "  test-cover  - Run tests with coverage"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker      - Build Docker image"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo ""
	@echo "Environment Variables:"
	@echo "  PORT                - Server port (default: 8095)"
	@echo "  ENVIRONMENT         - Environment (development/production)"
	@echo "  KB1_DOSING_URL      - KB-1 Dosing service URL"
	@echo "  KB2_INTERACTIONS_URL- KB-2 Interactions service URL"
	@echo "  KB3_GUIDELINES_URL  - KB-3 Guidelines service URL"
	@echo "  KB4_SAFETY_URL      - KB-4 Safety service URL"
	@echo "  REDIS_URL           - Redis connection URL"

# Build binary
build:
	@echo "Building medication-advisor-engine..."
	@go build -o bin/medication-advisor ./cmd/server
	@echo "Build complete: bin/medication-advisor"

# Run server
run: build
	@echo "Starting Medication Advisor Engine on port $${PORT:-8095}..."
	@./bin/medication-advisor

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t medication-advisor-engine:latest .
	@echo "Docker image built: medication-advisor-engine:latest"

# Run Docker container
docker-run: docker
	@echo "Running Docker container..."
	@docker run -p 8095:8095 \
		-e PORT=8095 \
		-e ENVIRONMENT=development \
		medication-advisor-engine:latest

# Lint code
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Formatting complete"

# Verify dependencies
deps:
	@echo "Verifying dependencies..."
	@go mod verify
	@go mod tidy
	@echo "Dependencies verified"

# Health check
health:
	@echo "Checking health..."
	@curl -s http://localhost:8095/health | jq .
