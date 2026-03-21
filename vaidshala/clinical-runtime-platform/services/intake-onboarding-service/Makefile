# Intake-Onboarding Service — Port 8141
.PHONY: help build run test test-cover clean docker lint fmt health

help:
	@echo "Intake-Onboarding Service — Port 8141"
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
	@go build -o bin/intake-onboarding-service ./cmd/intake
	@echo "Built: bin/intake-onboarding-service"

run: build
	@./bin/intake-onboarding-service

test:
	@go test -v ./...

test-cover:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@rm -rf bin/ coverage.out coverage.html

docker:
	@cd ../.. && docker build -f services/intake-onboarding-service/Dockerfile -t intake-onboarding-service:latest .

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...

health:
	@curl -s http://localhost:8141/healthz | python3 -m json.tool
