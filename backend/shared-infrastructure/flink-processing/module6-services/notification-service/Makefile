.PHONY: help build run test docker-build docker-up docker-down clean

help:
	@echo "CardioFit Notification Service - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make build        - Build the application with Maven"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run unit tests"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start services with docker-compose"
	@echo "  make docker-down  - Stop docker-compose services"
	@echo "  make docker-logs  - View docker-compose logs"
	@echo ""
	@echo "Integration:"
	@echo "  make integration-test - Run integration tests"
	@echo "  make kafka-test       - Send test alert to Kafka"
	@echo ""

build:
	@echo "Building application..."
	mvn clean package -DskipTests

build-with-tests:
	@echo "Building application with tests..."
	mvn clean package

run:
	@echo "Running application..."
	./run.sh

test:
	@echo "Running unit tests..."
	mvn test

clean:
	@echo "Cleaning build artifacts..."
	mvn clean
	rm -rf logs/

docker-build:
	@echo "Building Docker image..."
	docker build -t cardiofit-notification-service:latest .

docker-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services started!"
	@echo "Notification Service: http://localhost:8070"
	@echo "Grafana: http://localhost:3000"
	@echo "Prometheus: http://localhost:9090"

docker-down:
	@echo "Stopping docker-compose services..."
	docker-compose down

docker-logs:
	@echo "Viewing docker-compose logs..."
	docker-compose logs -f notification-service

integration-test:
	@echo "Running integration tests..."
	./test-notification-service.sh

kafka-test:
	@echo "Sending test alert to Kafka..."
	@echo '{"alert_id":"test-$(shell date +%s)","patient_id":"patient-123","patient_name":"John Doe","alert_type":"CRITICAL_VITAL_SIGN","severity":"CRITICAL","title":"Critical Heart Rate","message":"Heart rate exceeds threshold: 180 bpm","timestamp":"$(shell date -u +%Y-%m-%dT%H:%M:%SZ)","assigned_to":["user123"],"priority_score":0.95}' | \
		kafka-console-producer --broker-list localhost:29092 --topic composed-alerts

# Quick start for development
dev: build docker-up
	@echo "Development environment ready!"

# Full test suite
test-all: build test integration-test
	@echo "All tests completed!"

# Production deployment check
deploy-check:
	@echo "Checking deployment readiness..."
	@test -f .env || (echo "Error: .env file not found" && exit 1)
	@test -f firebase-credentials.json || echo "Warning: firebase-credentials.json not found"
	@echo "Deployment check passed!"
