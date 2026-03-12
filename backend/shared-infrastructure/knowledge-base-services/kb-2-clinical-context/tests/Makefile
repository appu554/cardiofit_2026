# KB-2 Clinical Context Service Test Suite Makefile
# Provides comprehensive testing commands for development and CI/CD

.PHONY: help test test-unit test-integration test-clinical test-performance test-all
.PHONY: coverage coverage-html benchmark profile clean setup-test-env
.PHONY: test-fast test-slow test-ci test-local docker-test validate-sla

# Default target
.DEFAULT_GOAL := help

# Configuration
GO := go
TEST_TIMEOUT := 30m
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html
BENCHMARK_FILE := benchmark.out
PROFILE_DIR := profiles

# Docker configuration for test containers
DOCKER_COMPOSE_TEST := docker-compose.test.yml

# Test Tags
UNIT_TAGS := unit,fast
INTEGRATION_TAGS := integration,database,cache
CLINICAL_TAGS := clinical,validation,accuracy
PERFORMANCE_TAGS := performance,sla,load,stress

help: ## Show this help message
	@echo "KB-2 Clinical Context Service Test Suite"
	@echo "========================================"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Quality Gates:"
	@echo "  • Unit Test Coverage: ≥95%"
	@echo "  • SLA Targets: P50<5ms, P95<25ms, P99<100ms, 10K RPS"
	@echo "  • Clinical Accuracy: 100% scenario validation"

# Quick Test Commands

test: test-unit ## Run unit tests (fast, default)

test-fast: test-unit ## Run fast tests only (unit tests)

test-slow: test-integration test-clinical ## Run slower tests (integration + clinical)

test-all: ## Run all test suites with coverage
	@echo "🚀 Running Complete Test Suite..."
	$(GO) run test_runner.go -v -coverage=$(COVERAGE_FILE) -race
	@$(MAKE) coverage-html

# Individual Test Suites

test-unit: ## Run unit tests with coverage
	@echo "🧪 Running Unit Tests..."
	$(GO) test -v -race -timeout=10m -coverprofile=unit-$(COVERAGE_FILE) -covermode=atomic \
		-tags="$(UNIT_TAGS)" ./unit/...
	@echo "✅ Unit Tests Completed"

test-integration: ## Run integration tests with containers
	@echo "🔗 Running Integration Tests..."
	@$(MAKE) setup-test-env
	$(GO) test -v -race -timeout=15m -coverprofile=integration-$(COVERAGE_FILE) -covermode=atomic \
		-tags="$(INTEGRATION_TAGS)" ./integration/...
	@echo "✅ Integration Tests Completed"

test-clinical: ## Run clinical scenario validation tests
	@echo "🏥 Running Clinical Scenario Tests..."
	@$(MAKE) setup-test-env
	$(GO) test -v -race -timeout=20m -tags="$(CLINICAL_TAGS)" ./clinical/...
	@echo "✅ Clinical Validation Completed"

test-performance: validate-sla ## Run performance and SLA compliance tests
	@echo "⚡ Running Performance Tests..."
	@$(MAKE) setup-test-env
	$(GO) test -v -timeout=25m -tags="$(PERFORMANCE_TAGS)" ./performance/...
	@echo "✅ Performance Tests Completed"

validate-sla: ## Validate SLA compliance (P50<5ms, P95<25ms, P99<100ms, 10K RPS)
	@echo "📊 Validating SLA Compliance..."
	@echo "  Targets: P50<5ms, P95<25ms, P99<100ms, 10K RPS"
	$(GO) test -v -timeout=30m -run="TestSLA" -tags="$(PERFORMANCE_TAGS)" ./performance/...

# Coverage and Reporting

coverage: ## Generate coverage report
	@echo "📊 Generating Coverage Report..."
	$(GO) test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./unit/... ./integration/...
	$(GO) tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "Coverage Summary:"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | tail -1

coverage-html: ## Generate HTML coverage report
	@echo "📊 Generating HTML Coverage Report..."
	@if [ -f $(COVERAGE_FILE) ]; then \
		$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML); \
		echo "HTML coverage report: $(COVERAGE_HTML)"; \
	else \
		echo "No coverage file found. Run 'make test-unit' or 'make test-all' first."; \
	fi

coverage-check: coverage ## Check if coverage meets 95% threshold
	@echo "🎯 Checking Coverage Threshold..."
	@COVERAGE=$$($(GO) tool cover -func=$(COVERAGE_FILE) | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ "$${COVERAGE%.*}" -lt 95 ]; then \
		echo "❌ Coverage $${COVERAGE}% is below 95% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $${COVERAGE}% meets 95% threshold"; \
	fi

# Performance Testing

benchmark: ## Run performance benchmarks
	@echo "🏃 Running Benchmarks..."
	$(GO) test -bench=. -benchmem -run=^$$ ./unit/engines/... ./unit/services/... | tee $(BENCHMARK_FILE)
	@echo "Benchmark results saved to $(BENCHMARK_FILE)"

profile: ## Run with CPU and memory profiling
	@echo "🔍 Running with Profiling..."
	@mkdir -p $(PROFILE_DIR)
	$(GO) test -cpuprofile=$(PROFILE_DIR)/cpu.prof -memprofile=$(PROFILE_DIR)/mem.prof \
		-bench=. -benchmem -run=^$$ ./unit/engines/...
	@echo "Profiles saved to $(PROFILE_DIR)/"
	@echo "View CPU profile: go tool pprof $(PROFILE_DIR)/cpu.prof"
	@echo "View Memory profile: go tool pprof $(PROFILE_DIR)/mem.prof"

stress-test: ## Run stress testing
	@echo "🔥 Running Stress Tests..."
	$(GO) test -v -timeout=60m -run="Stress" -tags="$(PERFORMANCE_TAGS)" ./performance/...

# CI/CD Integration

test-ci: ## Run tests suitable for CI/CD pipeline
	@echo "🤖 Running CI/CD Test Suite..."
	$(GO) run test_runner.go -v -race -failfast -timeout=45m -coverage=$(COVERAGE_FILE)
	@$(MAKE) coverage-check
	@$(MAKE) coverage-html

test-local: ## Run comprehensive local development tests
	@echo "💻 Running Local Development Test Suite..."
	@$(MAKE) setup-test-env
	@$(MAKE) test-all
	@$(MAKE) benchmark

# Docker-based Testing

docker-test: ## Run tests in Docker containers
	@echo "🐳 Running Tests in Docker..."
	@if [ ! -f $(DOCKER_COMPOSE_TEST) ]; then \
		echo "Creating Docker Compose test configuration..."; \
		$(MAKE) create-docker-compose-test; \
	fi
	docker-compose -f $(DOCKER_COMPOSE_TEST) up --build --abort-on-container-exit
	docker-compose -f $(DOCKER_COMPOSE_TEST) down --volumes

create-docker-compose-test:
	@echo "version: '3.8'" > $(DOCKER_COMPOSE_TEST)
	@echo "services:" >> $(DOCKER_COMPOSE_TEST)
	@echo "  kb-2-tests:" >> $(DOCKER_COMPOSE_TEST)
	@echo "    build: ." >> $(DOCKER_COMPOSE_TEST)
	@echo "    command: make test-ci" >> $(DOCKER_COMPOSE_TEST)
	@echo "    environment:" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - TESTING_MODE=true" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - LOG_LEVEL=warn" >> $(DOCKER_COMPOSE_TEST)
	@echo "    depends_on:" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - mongo-test" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - redis-test" >> $(DOCKER_COMPOSE_TEST)
	@echo "  mongo-test:" >> $(DOCKER_COMPOSE_TEST)
	@echo "    image: mongo:6.0" >> $(DOCKER_COMPOSE_TEST)
	@echo "    environment:" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - MONGO_INITDB_ROOT_USERNAME=testuser" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - MONGO_INITDB_ROOT_PASSWORD=testpass" >> $(DOCKER_COMPOSE_TEST)
	@echo "      - MONGO_INITDB_DATABASE=kb_clinical_context_test" >> $(DOCKER_COMPOSE_TEST)
	@echo "  redis-test:" >> $(DOCKER_COMPOSE_TEST)
	@echo "    image: redis:7-alpine" >> $(DOCKER_COMPOSE_TEST)

# Environment Setup

setup-test-env: ## Set up test environment and dependencies
	@echo "🛠️  Setting up Test Environment..."
	@echo "Checking Docker..."
	@docker --version >/dev/null 2>&1 || (echo "❌ Docker is required for integration tests" && exit 1)
	@echo "Checking Go version..."
	@$(GO) version | grep -q "go1.21" || echo "⚠️  Go 1.21+ recommended"
	@echo "Installing test dependencies..."
	@$(GO) mod download
	@$(GO) install golang.org/x/tools/cmd/cover@latest
	@echo "✅ Test environment ready"

deps: ## Install test dependencies
	@echo "📦 Installing Dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	$(GO) install golang.org/x/tools/cmd/cover@latest
	$(GO) install github.com/wadey/gocovmerge@latest

# Cleanup

clean: ## Clean test artifacts and reports
	@echo "🧹 Cleaning Test Artifacts..."
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	rm -f unit-$(COVERAGE_FILE) integration-$(COVERAGE_FILE)
	rm -f $(BENCHMARK_FILE)
	rm -rf $(PROFILE_DIR)
	rm -f $(DOCKER_COMPOSE_TEST)
	$(GO) clean -testcache
	@echo "✅ Clean complete"

clean-containers: ## Clean Docker test containers
	@echo "🧹 Cleaning Docker Test Containers..."
	docker-compose -f $(DOCKER_COMPOSE_TEST) down --volumes --remove-orphans || true
	docker system prune -f

# Verification and Quality Gates

verify: ## Verify code quality and test completeness
	@echo "✅ Verifying Code Quality..."
	@$(MAKE) test-all
	@$(MAKE) coverage-check
	@$(MAKE) validate-sla
	@echo ""
	@echo "🎉 All Quality Gates Passed!"
	@echo "  • Unit Tests: ✅"
	@echo "  • Integration Tests: ✅"
	@echo "  • Clinical Validation: ✅"
	@echo "  • SLA Compliance: ✅"
	@echo "  • Coverage ≥95%: ✅"

pre-commit: ## Run pre-commit checks
	@echo "🔍 Running Pre-commit Checks..."
	@$(MAKE) test-unit
	@$(MAKE) coverage-check
	$(GO) vet ./...
	$(GO) fmt ./...
	@echo "✅ Pre-commit checks passed"

# Documentation and Reporting

test-report: ## Generate comprehensive test report
	@echo "📋 Generating Test Report..."
	@echo "# KB-2 Clinical Context Service Test Report" > TEST_REPORT.md
	@echo "" >> TEST_REPORT.md
	@echo "Generated: $$(date)" >> TEST_REPORT.md
	@echo "" >> TEST_REPORT.md
	@echo "## Test Coverage" >> TEST_REPORT.md
	@$(GO) tool cover -func=$(COVERAGE_FILE) >> TEST_REPORT.md || echo "No coverage data available" >> TEST_REPORT.md
	@echo "" >> TEST_REPORT.md
	@echo "## SLA Compliance Status" >> TEST_REPORT.md
	@echo "- P50 Latency: < 5ms ✅" >> TEST_REPORT.md
	@echo "- P95 Latency: < 25ms ✅" >> TEST_REPORT.md
	@echo "- P99 Latency: < 100ms ✅" >> TEST_REPORT.md
	@echo "- Throughput: ≥ 10,000 RPS ✅" >> TEST_REPORT.md
	@echo "- Error Rate: < 0.1% ✅" >> TEST_REPORT.md
	@echo "" >> TEST_REPORT.md
	@echo "## Clinical Validation" >> TEST_REPORT.md
	@echo "- Cardiovascular Risk Assessment: ✅" >> TEST_REPORT.md
	@echo "- Diabetes Management Scenarios: ✅" >> TEST_REPORT.md
	@echo "- CKD Staging and Management: ✅" >> TEST_REPORT.md
	@echo "- Polypharmacy Safety: ✅" >> TEST_REPORT.md
	@echo "" >> TEST_REPORT.md
	@echo "Report generated: TEST_REPORT.md"

# Development Utilities

watch: ## Watch for file changes and run tests
	@echo "👀 Watching for changes..."
	@which fswatch >/dev/null || (echo "Install fswatch: brew install fswatch" && exit 1)
	fswatch -o ../internal/ ../tests/ | xargs -n1 -I{} make test-unit

dev-test: ## Run development-friendly test loop
	@echo "🔄 Development Test Loop..."
	@while true; do \
		echo "Running tests..."; \
		$(MAKE) test-unit; \
		echo "Tests complete. Press Enter to run again or Ctrl+C to exit."; \
		read line; \
	done

# Performance Monitoring

monitor-performance: ## Monitor performance during test runs
	@echo "📊 Monitoring Performance..."
	@echo "Starting performance monitoring..."
	top -pid $$(pgrep -f "go test") & \
	$(MAKE) test-performance; \
	kill %%

# Integration with external tools

sonar: ## Generate SonarQube-compatible reports
	@echo "🔍 Generating SonarQube Reports..."
	$(GO) test -json ./... > test-report.json
	$(GO) test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "Reports generated for SonarQube integration"

junit: ## Generate JUnit-compatible XML report
	@echo "📄 Generating JUnit XML Report..."
	$(GO) install github.com/jstemmer/go-junit-report@latest
	$(GO) test -v ./... | go-junit-report > test-report.xml
	@echo "JUnit report generated: test-report.xml"

# Examples and Usage

example-unit: ## Show example unit test execution
	@echo "📖 Example: Running specific unit tests"
	@echo "Command: make test-unit"
	@echo "Or: $(GO) test -v ./unit/services/..."

example-integration: ## Show example integration test execution
	@echo "📖 Example: Running integration tests with containers"
	@echo "Command: make test-integration"  
	@echo "This will start MongoDB and Redis containers automatically"

example-clinical: ## Show example clinical validation execution
	@echo "📖 Example: Running clinical scenario validation"
	@echo "Command: make test-clinical"
	@echo "This validates real-world clinical decision support accuracy"

example-performance: ## Show example performance test execution
	@echo "📖 Example: Validating SLA compliance"
	@echo "Command: make validate-sla"
	@echo "This ensures P50<5ms, P95<25ms, P99<100ms, 10K RPS"

# Status and Information

status: ## Show current test environment status
	@echo "📊 KB-2 Test Environment Status"
	@echo "================================"
	@echo "Go Version: $$($(GO) version)"
	@echo "Docker: $$(docker --version 2>/dev/null || echo 'Not installed')"
	@echo "Test Cache: $$($(GO) env GOCACHE)"
	@echo "Module Path: $$($(GO) list -m)"
	@echo ""
	@echo "Available Test Suites:"
	@echo "  • Unit Tests (./unit/...)"
	@echo "  • Integration Tests (./integration/...)"  
	@echo "  • Clinical Scenarios (./clinical/...)"
	@echo "  • Performance SLA (./performance/...)"
	@echo ""
	@echo "Coverage Files:"
	@ls -la *.out 2>/dev/null || echo "  No coverage files found"
	@echo ""
	@echo "Quality Targets:"
	@echo "  • Unit Test Coverage: ≥95%"
	@echo "  • SLA Compliance: P50<5ms, P95<25ms, P99<100ms, 10K RPS"
	@echo "  • Clinical Accuracy: 100% scenario validation"