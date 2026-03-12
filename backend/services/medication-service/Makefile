# Consolidated Medication Service Makefile
# This manages both the Python medication service and the Go knowledge-base services

.PHONY: help build test clean run-dev run-prod health logs stop

# Default target
help:
	@echo "Consolidated Medication Service - Available targets:"
	@echo ""
	@echo "=== MAIN SERVICES ==="
	@echo "  run-all           - Start all services (Python + Knowledge Bases + Flow2)"
	@echo "  stop-all          - Stop all services"
	@echo "  health-all        - Check health of all services"
	@echo "  logs-all          - Show logs from all services"
	@echo "  test-all          - Run all tests"
	@echo ""
	@echo "=== PYTHON MEDICATION SERVICE ==="
	@echo "  run-medication    - Start Python medication service"
	@echo "  test-medication   - Test Python medication service"
	@echo "  logs-medication   - Show medication service logs"
	@echo ""
	@echo "=== GO KNOWLEDGE-BASE SERVICES ==="
	@echo "  run-kb            - Start knowledge-base services"
	@echo "  stop-kb           - Stop knowledge-base services"
	@echo "  test-kb           - Test knowledge-base services"
	@echo "  logs-kb           - Show knowledge-base logs"
	@echo "  health-kb         - Check knowledge-base health"
	@echo ""
	@echo "=== GO FLOW2 ENGINE ==="
	@echo "  run-flow2         - Start Flow2 Go engine"
	@echo "  test-flow2        - Test Flow2 Go engine"
	@echo "  logs-flow2        - Show Flow2 logs"
	@echo ""
	@echo "=== RUST ENGINE ==="
	@echo "  run-rust          - Start Rust engine"
	@echo "  test-rust         - Test Rust engine"
	@echo ""
	@echo "=== UTILITIES ==="
	@echo "  clean             - Clean all build artifacts"
	@echo "  build-all         - Build all services"
	@echo "  setup             - Setup development environment"

# Main orchestration targets
run-all: run-kb run-flow2 run-rust run-medication
	@echo "🚀 All medication services started!"
	@echo ""
	@echo "Service URLs:"
	@echo "  Python Medication Service: http://localhost:8004"
	@echo "  Flow2 Go Engine:          http://localhost:8080"
	@echo "  KB-Drug-Rules:            http://localhost:8081"
	@echo "  KB-Guideline-Evidence:    http://localhost:8084"
	@echo "  KB-7-Terminology:         http://localhost:8087"
	@echo "  Rust Recipe Engine:       http://localhost:8090"
	@echo ""
	@echo "Use 'make health-all' to check service status"

stop-all:
	@echo "🛑 Stopping all medication services..."
	$(MAKE) stop-medication || true
	$(MAKE) stop-kb || true
	$(MAKE) stop-flow2 || true
	$(MAKE) stop-rust || true
	@echo "All services stopped"

health-all:
	@echo "🔍 Checking health of all services..."
	@echo ""
	@echo "=== Python Medication Service ==="
	@curl -f -s http://localhost:8004/health && echo "✅ Medication Service: HEALTHY" || echo "❌ Medication Service: UNHEALTHY"
	@echo ""
	@echo "=== Flow2 Go Engine ==="
	@curl -f -s http://localhost:8080/health && echo "✅ Flow2 Engine: HEALTHY" || echo "❌ Flow2 Engine: UNHEALTHY"
	@echo ""
	@echo "=== Knowledge-Base Services ==="
	@curl -f -s http://localhost:8081/health && echo "✅ KB-Drug-Rules: HEALTHY" || echo "❌ KB-Drug-Rules: UNHEALTHY"
	@curl -f -s http://localhost:8084/health && echo "✅ KB-Guidelines: HEALTHY" || echo "❌ KB-Guidelines: UNHEALTHY"
	@curl -f -s http://localhost:8087/health && echo "✅ KB-7-Terminology: HEALTHY" || echo "❌ KB-7-Terminology: UNHEALTHY"
	@echo ""
	@echo "=== Rust Recipe Engine ==="
	@curl -f -s http://localhost:8090/health && echo "✅ Rust Engine: HEALTHY" || echo "❌ Rust Engine: UNHEALTHY"

logs-all:
	@echo "📋 Showing logs from all services..."
	@echo "Use Ctrl+C to stop following logs"
	@echo ""
	$(MAKE) logs-medication &
	$(MAKE) logs-kb &
	$(MAKE) logs-flow2 &
	wait

test-all:
	@echo "🧪 Running all tests..."
	$(MAKE) test-medication
	$(MAKE) test-kb
	$(MAKE) test-flow2
	$(MAKE) test-rust
	@echo "All tests completed!"

# Python Medication Service targets
run-medication:
	@echo "🐍 Starting Python medication service..."
	@if [ ! -f requirements.txt ]; then \
		echo "❌ requirements.txt not found in current directory"; \
		exit 1; \
	fi
	@if ! command -v python >/dev/null 2>&1; then \
		echo "❌ Python not found. Please install Python 3.8+"; \
		exit 1; \
	fi
	python run_service.py

stop-medication:
	@echo "Stopping Python medication service..."
	@pkill -f "run_service.py" 2>/dev/null || true
	@pkill -f "uvicorn.*medication" 2>/dev/null || true

test-medication:
	@echo "Testing Python medication service..."
	@if command -v pytest >/dev/null 2>&1; then \
		pytest tests/ -v; \
	else \
		echo "pytest not found, installing..."; \
		pip install pytest; \
		pytest tests/ -v; \
	fi

logs-medication:
	@echo "📋 Python Medication Service logs:"
	@tail -f logs/*.log 2>/dev/null || echo "No log files found. Service may not be running."

# Knowledge-Base Services targets
run-kb:
	@echo "🧠 Starting knowledge-base services..."
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) run-kb-docker

stop-kb:
	@echo "Stopping knowledge-base services..."
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) stop-kb

test-kb:
	@echo "Testing knowledge-base services..."
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) test

logs-kb:
	@echo "📋 Knowledge-base services logs:"
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) logs-kb

health-kb:
	@echo "Checking knowledge-base services health..."
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) health

# Flow2 Go Engine targets
run-flow2:
	@echo "⚡ Starting Flow2 Go engine..."
	@if [ ! -f flow2-go-engine/go.mod ]; then \
		echo "❌ Flow2 Go engine not found"; \
		exit 1; \
	fi
	cd flow2-go-engine && go run cmd/server/main.go &

stop-flow2:
	@echo "Stopping Flow2 Go engine..."
	@pkill -f "flow2-go-engine" 2>/dev/null || true

test-flow2:
	@echo "Testing Flow2 Go engine..."
	cd flow2-go-engine && go test -v ./...

logs-flow2:
	@echo "📋 Flow2 Go engine logs:"
	@echo "Flow2 logs would appear here (implement logging)"

# Rust Engine targets
run-rust:
	@echo "🦀 Starting Rust recipe engine..."
	@if [ ! -f flow2-rust-engine/Cargo.toml ]; then \
		echo "❌ Rust engine not found"; \
		exit 1; \
	fi
	cd flow2-rust-engine && cargo run --bin medication_service &

stop-rust:
	@echo "Stopping Rust engine..."
	@pkill -f "medication_service" 2>/dev/null || true

test-rust:
	@echo "Testing Rust engine..."
	cd flow2-rust-engine && cargo test

# Build targets
build-all:
	@echo "🔨 Building all services..."
	$(MAKE) build-kb
	$(MAKE) build-flow2
	$(MAKE) build-rust
	@echo "All services built!"

build-kb:
	@echo "Building knowledge-base services..."
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) build

build-flow2:
	@echo "Building Flow2 Go engine..."
	cd flow2-go-engine && go build -o bin/flow2-engine cmd/server/main.go

build-rust:
	@echo "Building Rust engine..."
	cd flow2-rust-engine && cargo build --release

# Development setup
setup:
	@echo "🛠️ Setting up development environment..."
	@echo ""
	@echo "=== Python Dependencies ==="
	@if [ -f requirements.txt ]; then \
		pip install -r requirements.txt; \
	else \
		echo "requirements.txt not found, skipping Python setup"; \
	fi
	@echo ""
	@echo "=== Go Dependencies ==="
	@if [ -f flow2-go-engine/go.mod ]; then \
		cd flow2-go-engine && go mod download; \
	fi
	@echo ""
	@echo "=== Rust Dependencies ==="
	@if [ -f flow2-rust-engine/Cargo.toml ]; then \
		cd flow2-rust-engine && cargo fetch; \
	fi
	@echo ""
	@echo "=== Knowledge-Base Setup ==="
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) dev-setup
	@echo ""
	@echo "✅ Development environment setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make run-all' to start all services"
	@echo "  2. Run 'make health-all' to verify services are running"
	@echo "  3. Run 'make test-all' to run all tests"

# Utility targets
clean:
	@echo "🧹 Cleaning build artifacts..."
	@echo "Cleaning Python cache..."
	find . -name "*.pyc" -delete
	find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "Cleaning Go builds..."
	cd flow2-go-engine && rm -rf bin/ 2>/dev/null || true
	@echo "Cleaning Rust builds..."
	cd flow2-rust-engine && cargo clean 2>/dev/null || true
	@echo "Cleaning knowledge-base builds..."
	cd ../../shared-infrastructure/knowledge-base-services && $(MAKE) clean
	@echo "Clean complete!"

# Documentation and API testing
docs:
	@echo "📖 Generating documentation..."
	@echo "Opening service documentation..."
	@echo ""
	@echo "Service Documentation:"
	@echo "  Python Medication Service: http://localhost:8004/docs"
	@echo "  Flow2 Go Engine:          http://localhost:8080/docs"
	@echo "  KB-Drug-Rules:            http://localhost:8081/swagger"

test-integration:
	@echo "🔗 Running integration tests..."
	@echo "Testing service interactions..."
	@# Test medication service -> knowledge base integration
	@curl -f -s http://localhost:8004/health >/dev/null && \
	 curl -f -s http://localhost:8081/health >/dev/null && \
	 echo "✅ Integration test passed: Medication <-> Knowledge Base" || \
	 echo "❌ Integration test failed: Services not responding"

# Environment info
info:
	@echo "🔍 Consolidated Medication Service - Environment Info"
	@echo "=================================================="
	@echo ""
	@echo "=== System Info ==="
	@echo "Python version: $(shell python --version 2>/dev/null || echo "Not installed")"
	@echo "Go version: $(shell go version 2>/dev/null || echo "Not installed")"
	@echo "Rust version: $(shell rustc --version 2>/dev/null || echo "Not installed")"
	@echo "Docker version: $(shell docker --version 2>/dev/null || echo "Not installed")"
	@echo ""
	@echo "=== Service Structure ==="
	@echo "📁 medication-service/"
	@echo "├── app/                     (Python FastAPI service)"
	@echo "├── flow2-go-engine/         (Go orchestration engine)"
	@echo "├── flow2-rust-engine/       (Rust clinical rules engine)"
	@echo "└── knowledge/               (Clinical knowledge data)"
	@echo ""
	@echo "📁 ../../shared-infrastructure/knowledge-base-services/"
	@echo "├── kb-drug-rules/           (Drug calculation rules)"
	@echo "├── kb-guideline-evidence/   (Clinical guidelines)"
	@echo "└── api-gateway/             (Knowledge base gateway)"
	@echo ""
	@echo "=== Port Assignments ==="
	@echo "8004 - Python Medication Service"
	@echo "8080 - Flow2 Go Engine"
	@echo "8081 - KB-Drug-Rules"
	@echo "8084 - KB-Guideline-Evidence"
	@echo "8087 - KB-7-Terminology"
	@echo "8090 - Rust Recipe Engine"
	@echo ""
	@echo "=== Running Services ==="
	@echo "Checking service status..."
	@$(MAKE) health-all 2>/dev/null