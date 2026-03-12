.PHONY: help build run test clean docker-build docker-up docker-down proto lint

help:
	@echo "Available commands:"
	@echo "  make build         - Build the notification service binary"
	@echo "  make run           - Run the notification service locally"
	@echo "  make test          - Run all tests"
	@echo "  make test-unit     - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-up     - Start services with Docker Compose"
	@echo "  make docker-down   - Stop services with Docker Compose"
	@echo "  make proto         - Generate protobuf code"
	@echo "  make lint          - Run linters"
	@echo "  make deps          - Download dependencies"

build:
	@echo "Building notification service..."
	go build -o bin/notification-service ./cmd/server

run:
	@echo "Running notification service..."
	go run ./cmd/server/main.go

test:
	@echo "Running all tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-unit:
	@echo "Running unit tests..."
	go test -v -race -short ./tests/unit/...

test-integration:
	@echo "Running integration tests..."
	go test -v -race ./tests/integration/...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out

docker-build:
	@echo "Building Docker image..."
	docker build -t cardiofit/notification-service:latest -f deployments/docker/Dockerfile .

docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose -f deployments/docker/docker-compose.yml up -d

docker-down:
	@echo "Stopping services with Docker Compose..."
	docker-compose -f deployments/docker/docker-compose.yml down

proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go-grpc_out=. pkg/proto/*.proto

lint:
	@echo "Running linters..."
	golangci-lint run

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
