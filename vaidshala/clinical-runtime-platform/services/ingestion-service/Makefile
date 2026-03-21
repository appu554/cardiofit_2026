# Ingestion Service — Port 8140
.PHONY: help build run test test-cover clean docker lint fmt health

help:
	@echo "Ingestion Service — Port 8140"
	@echo "  build       Build the binary"
	@echo "  run         Build and run"
	@echo "  test        Run all tests"
	@echo "  test-cover  Run tests with coverage"
	@echo "  clean       Clean build artifacts"
	@echo "  docker      Build Docker image"
	@echo "  lint        Run linter"
	@echo "  fmt         Format code"
	@echo "  health      Check service health"

build:
	@go build -o bin/ingestion-service ./cmd/ingestion
	@echo "Built: bin/ingestion-service"

run: build
	@./bin/ingestion-service

test:
	@go test -v ./...

test-cover:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@rm -rf bin/ coverage.out coverage.html

docker:
	@cd ../.. && docker build -f services/ingestion-service/Dockerfile -t ingestion-service:latest .

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...

health:
	@curl -s http://localhost:8140/healthz | python3 -m json.tool
