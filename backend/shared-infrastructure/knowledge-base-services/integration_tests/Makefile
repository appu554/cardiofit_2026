# Makefile for Cross-Service Integration Tests

.PHONY: help install health test test-scenario clean check-python

# Default target
help: ## Show this help message
	@echo "Cross-Service Integration Tests"
	@echo "==============================="
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make health                                    # Quick health check"
	@echo "  make test                                      # Run all tests"
	@echo "  make test-scenario SCENARIO=terminology_check # Run specific scenario"

check-python: ## Check if Python and required packages are available
	@python --version >/dev/null 2>&1 || (echo "❌ Python not found. Please install Python 3.9+"; exit 1)
	@python -c "import aiohttp" >/dev/null 2>&1 || (echo "❌ aiohttp not found. Run 'make install'"; exit 1)
	@echo "✅ Python environment ready"

install: ## Install required dependencies
	@echo "📦 Installing dependencies..."
	pip install -r requirements.txt
	@echo "✅ Dependencies installed"

health: check-python ## Quick health check of all KB services
	@echo "🏥 Checking health of all KB services..."
	python run_tests.py --quick

list: check-python ## List available test scenarios
	@echo "📋 Available test scenarios:"
	python run_tests.py --list-scenarios

test: check-python ## Run all integration tests
	@echo "🧪 Running all integration tests..."
	python run_tests.py --verbose

test-scenario: check-python ## Run specific test scenario (use SCENARIO=name)
ifndef SCENARIO
	@echo "❌ Please specify SCENARIO. Example: make test-scenario SCENARIO=terminology_consistency_check"
	@echo "Available scenarios:"
	@python run_tests.py --list-scenarios | grep -E "^\d+\." | head -10
else
	@echo "🎯 Running scenario: $(SCENARIO)"
	python run_tests.py --scenario $(SCENARIO) --verbose
endif

test-quick: check-python ## Run quick subset of tests
	@echo "⚡ Running quick integration tests..."
	python run_tests.py --scenario terminology_consistency_check --verbose

test-full: check-python ## Run comprehensive test suite with detailed reporting
	@echo "🔬 Running comprehensive integration test suite..."
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	python run_tests.py --verbose --report "comprehensive_report_$$TIMESTAMP.md" && \
	echo "📊 Comprehensive report saved to comprehensive_report_$$TIMESTAMP.md"

validate-services: check-python ## Validate that all required services are accessible
	@echo "🔍 Validating service accessibility..."
	@echo "Checking KB-1 Drug Rules (8081)..."
	@curl -s -f http://localhost:8081/api/v1/health >/dev/null && echo "✅ KB-1 accessible" || echo "❌ KB-1 not accessible"
	@curl -s -f http://localhost:8082/api/v1/health >/dev/null && echo "✅ KB-2 accessible" || echo "❌ KB-2 not accessible"  
	@curl -s -f http://localhost:8084/api/v1/health >/dev/null && echo "✅ KB-3 accessible" || echo "❌ KB-3 not accessible"
	@curl -s -f http://localhost:8085/api/v1/health >/dev/null && echo "✅ KB-4 accessible" || echo "❌ KB-4 not accessible"
	@curl -s -f http://localhost:8086/api/v1/health >/dev/null && echo "✅ KB-5 accessible" || echo "❌ KB-5 not accessible"
	@curl -s -f http://localhost:8087/api/v1/health >/dev/null && echo "✅ KB-6 accessible" || echo "❌ KB-6 not accessible"
	@curl -s -f http://localhost:8088/api/v1/health >/dev/null && echo "✅ KB-7 accessible" || echo "✅ KB-7 not accessible"

logs: ## View recent test logs
	@echo "📋 Recent test logs:"
	@ls -la integration_tests_*.log 2>/dev/null | head -5 || echo "No test logs found"

reports: ## View recent test reports  
	@echo "📊 Recent test reports:"
	@ls -la integration_test_report_*.md comprehensive_report_*.md 2>/dev/null | head -5 || echo "No test reports found"

clean: ## Clean up test artifacts (logs, reports)
	@echo "🧹 Cleaning up test artifacts..."
	@rm -f integration_tests_*.log
	@rm -f integration_test_report_*.md
	@rm -f comprehensive_report_*.md
	@rm -rf __pycache__
	@rm -rf .pytest_cache
	@echo "✅ Cleanup completed"

dev: ## Setup development environment
	@echo "🔧 Setting up development environment..."
	make install
	@echo "Creating development test data..."
	@mkdir -p test_data
	@echo "✅ Development environment ready"

ci: check-python ## Run tests in CI mode (no interaction, detailed logging)
	@echo "🤖 Running tests in CI mode..."
	@TIMESTAMP=$$(date +%Y%m%d_%H%M%S); \
	python run_tests.py --report "ci_report_$$TIMESTAMP.md" --verbose > "ci_output_$$TIMESTAMP.log" 2>&1; \
	EXIT_CODE=$$?; \
	echo "Exit code: $$EXIT_CODE"; \
	if [ $$EXIT_CODE -eq 0 ]; then \
		echo "✅ All tests passed"; \
	else \
		echo "❌ Some tests failed"; \
		tail -20 "ci_output_$$TIMESTAMP.log"; \
	fi; \
	exit $$EXIT_CODE

performance: check-python ## Run performance-focused tests
	@echo "⚡ Running performance tests..."
	python run_tests.py --scenario performance_stress_test --verbose

# Service-specific test targets
test-terminology: check-python ## Test terminology service specifically
	@echo "📚 Testing KB-7 Terminology Service..."
	python -c "
import asyncio
import aiohttp
async def test_terminology():
    async with aiohttp.ClientSession() as session:
        async with session.get('http://localhost:8088/api/v1/health') as resp:
            if resp.status == 200:
                print('✅ KB-7 Terminology Service is healthy')
            else:
                print('❌ KB-7 Terminology Service is not healthy')
asyncio.run(test_terminology())
"

# Documentation targets  
docs: ## View README
	@echo "📖 Integration Test Documentation"
	@echo "================================="
	@head -20 README.md

version: ## Show version information
	@echo "Cross-Service Integration Tests v1.0.0"
	@python --version
	@echo "Dependencies:"
	@pip list | grep -E "(aiohttp|pytest|pyyaml)" || echo "Dependencies not installed"

# Windows-specific targets (using PowerShell instead of curl)
ifeq ($(OS),Windows_NT)
validate-services-windows: check-python ## Validate services on Windows
	@echo "🔍 Validating service accessibility (Windows)..."
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8081/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-1 accessible' } catch { Write-Host '❌ KB-1 not accessible' }"
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8082/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-2 accessible' } catch { Write-Host '❌ KB-2 not accessible' }"
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8084/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-3 accessible' } catch { Write-Host '❌ KB-3 not accessible' }"
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8085/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-4 accessible' } catch { Write-Host '❌ KB-4 not accessible' }"
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8086/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-5 accessible' } catch { Write-Host '❌ KB-5 not accessible' }"
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8087/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-6 accessible' } catch { Write-Host '❌ KB-6 not accessible' }"
	@powershell -Command "try { Invoke-WebRequest -Uri 'http://localhost:8088/api/v1/health' -UseBasicParsing | Out-Null; Write-Host '✅ KB-7 accessible' } catch { Write-Host '❌ KB-7 not accessible' }"
endif