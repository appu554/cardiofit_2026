# KB7 Terminology Phase 3.5 Benchmarking Makefile
# ==================================================

.PHONY: help install check-services test-all test-validation test-performance test-load test-google-fhir clean reports

# Default target
help:
	@echo "KB7 Terminology Phase 3.5 Benchmarking System"
	@echo "=============================================="
	@echo ""
	@echo "Available targets:"
	@echo "  install              Install dependencies and setup environment"
	@echo "  check-services       Check if all required services are running"
	@echo "  test-all            Run complete Phase 3.5 test suite"
	@echo "  test-validation     Run Phase 3.5 success criteria validation"
	@echo "  test-performance    Run performance tests with realistic workloads"
	@echo "  test-load           Run load testing scenarios"
	@echo "  test-google-fhir    Run Google FHIR integration tests"
	@echo "  test-quick          Run quick validation (reduced iterations)"
	@echo "  test-ci             Run tests suitable for CI/CD pipeline"
	@echo "  reports             Generate comprehensive reports"
	@echo "  clean               Clean up test artifacts and reports"
	@echo "  docker-build        Build Docker image for testing"
	@echo "  docker-test         Run tests in Docker container"
	@echo ""
	@echo "Configuration:"
	@echo "  CONFIG_FILE         Path to configuration file (default: test_config.yaml)"
	@echo "  OUTPUT_DIR          Output directory for reports (default: reports)"
	@echo "  LOG_LEVEL           Logging level (default: INFO)"

# Configuration variables
CONFIG_FILE ?= test_config.yaml
OUTPUT_DIR ?= reports
LOG_LEVEL ?= INFO
PYTHON ?= python3
PIP ?= pip3

# Install dependencies and setup environment
install:
	@echo "Installing KB7 benchmarking dependencies..."
	$(PIP) install -r requirements.txt
	@echo "Creating output directory..."
	mkdir -p $(OUTPUT_DIR)
	@echo "Validating configuration..."
	$(PYTHON) -c "import yaml; yaml.safe_load(open('$(CONFIG_FILE)'))"
	@echo "Installation complete!"

# Check if all required services are running
check-services:
	@echo "Checking KB7 service health..."
	@echo "PostgreSQL (port 5433):"
	@timeout 5 bash -c '</dev/tcp/localhost/5433' && echo "✅ PostgreSQL accessible" || echo "❌ PostgreSQL not accessible"
	@echo "Redis (port 6380):"
	@timeout 5 bash -c '</dev/tcp/localhost/6380' && echo "✅ Redis accessible" || echo "❌ Redis not accessible"
	@echo "Neo4j (port 7687):"
	@timeout 5 bash -c '</dev/tcp/localhost/7687' && echo "✅ Neo4j accessible" || echo "❌ Neo4j not accessible"
	@echo "Query Router (port 8090):"
	@curl -s -f http://localhost:8090/health >/dev/null && echo "✅ Query Router healthy" || echo "❌ Query Router not healthy"
	@echo "FHIR Service (port 8014):"
	@curl -s -f http://localhost:8014/health >/dev/null && echo "✅ FHIR Service healthy" || echo "❌ FHIR Service not healthy"

# Run complete Phase 3.5 test suite
test-all: check-services
	@echo "Running complete KB7 Phase 3.5 test suite..."
	$(PYTHON) run_all_tests.py \
		--config $(CONFIG_FILE) \
		--output-dir $(OUTPUT_DIR) \
		--performance-workload realistic_mixed \
		--load-scenario stress_test \
		--google-fhir-iterations 100

# Run Phase 3.5 success criteria validation
test-validation: check-services
	@echo "Running Phase 3.5 success criteria validation..."
	$(PYTHON) phase35_validation.py \
		--config $(CONFIG_FILE) \
		--output $(OUTPUT_DIR)/phase35_validation.json

# Run performance tests with realistic workloads
test-performance: check-services
	@echo "Running performance tests..."
	$(PYTHON) performance_tests.py \
		--workload realistic_mixed \
		--duration 300 \
		--users 10 \
		--config $(CONFIG_FILE) \
		--output $(OUTPUT_DIR)/performance_results.json

# Run load testing scenarios
test-load: check-services
	@echo "Running load testing..."
	$(PYTHON) load_testing.py \
		--scenario stress_test \
		--config $(CONFIG_FILE) \
		--output $(OUTPUT_DIR)/load_test_results.json

# Run Google FHIR integration tests
test-google-fhir:
	@echo "Running Google FHIR integration tests..."
	$(PYTHON) google_fhir_benchmark.py \
		--iterations 100 \
		--config $(CONFIG_FILE) \
		--output $(OUTPUT_DIR)/google_fhir_results.json

# Quick validation for development
test-quick: check-services
	@echo "Running quick validation (reduced iterations)..."
	$(PYTHON) run_all_tests.py \
		--config $(CONFIG_FILE) \
		--output-dir $(OUTPUT_DIR) \
		--performance-workload cache_optimized \
		--skip-load \
		--google-fhir-iterations 20

# CI/CD optimized test run
test-ci: check-services
	@echo "Running CI/CD optimized tests..."
	$(PYTHON) run_all_tests.py \
		--config $(CONFIG_FILE) \
		--output-dir $(OUTPUT_DIR) \
		--sequential \
		--performance-workload realistic_mixed \
		--skip-google-fhir \
		--fail-fast

# Run individual workload tests
test-workload-mixed:
	$(PYTHON) performance_tests.py --workload realistic_mixed --config $(CONFIG_FILE)

test-workload-reasoning:
	$(PYTHON) performance_tests.py --workload heavy_reasoning --config $(CONFIG_FILE)

test-workload-fhir:
	$(PYTHON) performance_tests.py --workload fhir_heavy --config $(CONFIG_FILE)

test-workload-cache:
	$(PYTHON) performance_tests.py --workload cache_optimized --config $(CONFIG_FILE)

# Run individual load scenarios
test-stress:
	$(PYTHON) load_testing.py --scenario stress_test --config $(CONFIG_FILE)

test-soak:
	$(PYTHON) load_testing.py --scenario soak_test --config $(CONFIG_FILE)

test-spike:
	$(PYTHON) load_testing.py --scenario spike_test --config $(CONFIG_FILE)

test-capacity:
	$(PYTHON) load_testing.py --scenario capacity_test --config $(CONFIG_FILE)

# Generate comprehensive reports
reports:
	@echo "Generating comprehensive reports..."
	@if [ -d "$(OUTPUT_DIR)" ]; then \
		echo "Found results in $(OUTPUT_DIR)"; \
		echo "Generating summary report..."; \
		$(PYTHON) -c "import json, glob; \
		files = glob.glob('$(OUTPUT_DIR)/*.json'); \
		print('📊 Report Summary:'); \
		[print(f'  📄 {f}') for f in files]"; \
		if [ -f "$(OUTPUT_DIR)/test_suite_report.html" ]; then \
			echo "🌐 HTML report available: $(OUTPUT_DIR)/test_suite_report.html"; \
		fi; \
		if [ -f "$(OUTPUT_DIR)/junit_results.xml" ]; then \
			echo "📋 JUnit report available: $(OUTPUT_DIR)/junit_results.xml"; \
		fi; \
	else \
		echo "❌ No reports found. Run tests first."; \
	fi

# Validate Phase 3.5 success criteria specifically
validate-criteria: check-services
	@echo "Validating specific Phase 3.5 success criteria..."
	@echo "1. PostgreSQL lookup queries <10ms (95th percentile)"
	$(PYTHON) -c "
import asyncio
from phase35_validation import Phase35Validator
async def main():
    validator = Phase35Validator('$(CONFIG_FILE)')
    result = await validator._validate_postgresql_performance()
    print(f'   Result: {result.actual} vs target {result.target} - {'✅ PASS' if result.passed else '❌ FAIL'}')
    validator.close()
asyncio.run(main())
"
	@echo "2. GraphDB reasoning queries <50ms (95th percentile)"
	@echo "3. Query router uptime >99.9%"
	@echo "4. Data migration completed with 100% integrity"
	@echo "5. FHIR endpoints using hybrid architecture"
	@echo "6. Cache hit ratio >90% for frequent operations"
	@echo "7. Performance monitoring and alerting operational"

# Performance regression testing
test-regression:
	@echo "Running performance regression tests..."
	@if [ -f "$(OUTPUT_DIR)/baseline_results.json" ]; then \
		$(PYTHON) run_all_tests.py \
			--config $(CONFIG_FILE) \
			--output-dir $(OUTPUT_DIR) \
			--baseline-file $(OUTPUT_DIR)/baseline_results.json \
			--fail-on-regression; \
	else \
		echo "❌ No baseline found. Creating baseline..."; \
		$(PYTHON) run_all_tests.py \
			--config $(CONFIG_FILE) \
			--output-dir $(OUTPUT_DIR); \
		cp $(OUTPUT_DIR)/test_suite_results.json $(OUTPUT_DIR)/baseline_results.json; \
		echo "✅ Baseline created: $(OUTPUT_DIR)/baseline_results.json"; \
	fi

# Clean up test artifacts and reports
clean:
	@echo "Cleaning up test artifacts..."
	rm -rf $(OUTPUT_DIR)/*
	rm -f *.log
	rm -rf __pycache__/
	rm -rf *.pyc
	find . -name "*.pyc" -delete
	find . -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
	@echo "Cleanup complete!"

# Docker operations
docker-build:
	@echo "Building Docker image for KB7 benchmarking..."
	docker build -t kb7-benchmarks:latest .

docker-test: docker-build
	@echo "Running tests in Docker container..."
	docker run --rm \
		--network host \
		-v $(PWD)/$(OUTPUT_DIR):/app/reports \
		kb7-benchmarks:latest \
		--config test_config.yaml \
		--output-dir reports

# Development helpers
dev-setup: install
	@echo "Setting up development environment..."
	$(PIP) install -e .
	pre-commit install || echo "pre-commit not available"

lint:
	@echo "Running code quality checks..."
	flake8 *.py || echo "flake8 not available"
	black --check *.py || echo "black not available"
	mypy *.py || echo "mypy not available"

format:
	@echo "Formatting code..."
	black *.py || echo "black not available"
	isort *.py || echo "isort not available"

# Monitoring and alerting test
test-monitoring:
	@echo "Testing monitoring and alerting systems..."
	@curl -s http://localhost:8090/metrics | head -20 || echo "❌ Metrics endpoint not available"
	@echo "Checking Prometheus metrics format..."
	$(PYTHON) -c "
import httpx, asyncio
from prometheus_client.parser import text_string_to_metric_families
async def check():
    try:
        async with httpx.AsyncClient() as client:
            resp = await client.get('http://localhost:8090/metrics', timeout=5.0)
            metrics = list(text_string_to_metric_families(resp.text))
            print(f'✅ Found {len(metrics)} metric families')
            for m in metrics[:3]:
                print(f'   📊 {m.name}: {m.help}')
    except Exception as e:
        print(f'❌ Metrics check failed: {e}')
asyncio.run(check())
"

# Performance profiling
profile-performance:
	@echo "Running performance profiling..."
	$(PYTHON) -m cProfile -o $(OUTPUT_DIR)/performance_profile.prof performance_tests.py --workload realistic_mixed --duration 60
	@echo "Profile saved to $(OUTPUT_DIR)/performance_profile.prof"
	$(PYTHON) -c "
import pstats
p = pstats.Stats('$(OUTPUT_DIR)/performance_profile.prof')
p.sort_stats('cumulative').print_stats(20)
" > $(OUTPUT_DIR)/performance_profile.txt
	@echo "Profile analysis saved to $(OUTPUT_DIR)/performance_profile.txt"

# Memory usage analysis
profile-memory:
	@echo "Running memory profiling..."
	$(PYTHON) -m memory_profiler performance_tests.py --workload cache_optimized --duration 60

# Network performance test
test-network:
	@echo "Testing network performance..."
	@echo "Query Router latency:"
	@for i in {1..10}; do \
		curl -w "@-" -o /dev/null -s "http://localhost:8090/health" <<< "time_total: %{time_total}s\n"; \
	done
	@echo "FHIR Service latency:"
	@for i in {1..10}; do \
		curl -w "@-" -o /dev/null -s "http://localhost:8014/health" <<< "time_total: %{time_total}s\n"; \
	done

# Database performance test
test-database:
	@echo "Testing database performance..."
	$(PYTHON) -c "
import time, psycopg2, redis, statistics
from neo4j import GraphDatabase

# PostgreSQL test
times = []
for i in range(10):
    start = time.perf_counter()
    conn = psycopg2.connect(host='localhost', port=5433, database='terminology_db', user='kb7_user', password='kb7_password')
    conn.cursor().execute('SELECT 1')
    conn.close()
    times.append((time.perf_counter() - start) * 1000)
print(f'PostgreSQL connection: {statistics.mean(times):.1f}ms avg')

# Redis test
times = []
for i in range(10):
    start = time.perf_counter()
    r = redis.Redis(host='localhost', port=6380, db=0)
    r.ping()
    r.close()
    times.append((time.perf_counter() - start) * 1000)
print(f'Redis connection: {statistics.mean(times):.1f}ms avg')

# Neo4j test
times = []
for i in range(10):
    start = time.perf_counter()
    driver = GraphDatabase.driver('bolt://localhost:7687', auth=('neo4j', 'kb7_neo4j'))
    with driver.session() as session:
        session.run('RETURN 1')
    driver.close()
    times.append((time.perf_counter() - start) * 1000)
print(f'Neo4j connection: {statistics.mean(times):.1f}ms avg')
"

# Export metrics to external systems
export-metrics:
	@echo "Exporting metrics to external systems..."
	@if [ -f "$(OUTPUT_DIR)/test_suite_results.json" ]; then \
		echo "Converting to Prometheus format..."; \
		$(PYTHON) -c "
import json
with open('$(OUTPUT_DIR)/test_suite_results.json') as f:
    data = json.load(f)
    print('# KB7 Phase 3.5 Test Metrics')
    print(f'kb7_test_success_rate {data.get(\"summary_statistics\", {}).get(\"success_rate_pct\", 0)}')
    print(f'kb7_test_duration_seconds {data.get(\"total_duration_seconds\", 0)}')
    print(f'kb7_test_total_count {data.get(\"summary_statistics\", {}).get(\"total_tests\", 0)}')
"; \
	else \
		echo "❌ No test results found to export"; \
	fi

# Continuous monitoring setup
setup-monitoring:
	@echo "Setting up continuous monitoring..."
	@echo "Creating monitoring configuration..."
	@cat > $(OUTPUT_DIR)/monitoring_config.yaml << EOF
monitoring:
  interval_minutes: 15
  tests:
    - test_validation
    - test_performance
  alerts:
    - condition: success_rate < 95
      severity: critical
    - condition: p95_latency > 100
      severity: warning
EOF
	@echo "Monitoring configuration created: $(OUTPUT_DIR)/monitoring_config.yaml"

# Generate test documentation
docs:
	@echo "Generating test documentation..."
	$(PYTHON) -c "
import importlib.util
import inspect

modules = ['phase35_validation', 'performance_tests', 'load_testing', 'google_fhir_benchmark']
for mod_name in modules:
    spec = importlib.util.spec_from_file_location(mod_name, f'{mod_name}.py')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)

    print(f'## {mod_name.replace(\"_\", \" \").title()}')
    print(f'**Module**: {mod_name}.py')
    print(f'**Purpose**: {module.__doc__.split(chr(10))[0] if module.__doc__ else \"No description\"}')

    classes = [cls for name, cls in inspect.getmembers(module, inspect.isclass) if not name.startswith('_')]
    for cls in classes[:3]:  # Limit to first 3 classes
        print(f'- **{cls.__name__}**: {cls.__doc__.split(chr(10))[0] if cls.__doc__ else \"No description\"}')
    print()
" > $(OUTPUT_DIR)/test_documentation.md
	@echo "Documentation generated: $(OUTPUT_DIR)/test_documentation.md"

# Show current system status
status:
	@echo "KB7 Terminology System Status"
	@echo "============================="
	@echo "Services:"
	@make check-services
	@echo ""
	@echo "Recent test results:"
	@if [ -d "$(OUTPUT_DIR)" ]; then \
		ls -la $(OUTPUT_DIR)/*.json 2>/dev/null | tail -5 || echo "  No recent results"; \
	else \
		echo "  No results directory"; \
	fi
	@echo ""
	@echo "System resources:"
	@echo "  CPU: $$(top -l 1 | grep "CPU usage" | awk '{print $$3}' | cut -d% -f1 2>/dev/null || echo 'N/A')% usage"
	@echo "  Memory: $$(top -l 1 | grep "PhysMem" | awk '{print $$2}' 2>/dev/null || echo 'N/A') used"
	@echo "  Disk: $$(df -h . | tail -1 | awk '{print $$5}' 2>/dev/null || echo 'N/A') used"