# =============================================================================
# COMPREHENSIVE TESTING TARGETS - Add to existing Makefile
# =============================================================================

# Override existing test targets with comprehensive implementations
test: test-comprehensive ## Run comprehensive test suite

test-comprehensive: ## Run all test suites with comprehensive coverage
	@echo "🧪 Running Medication Service V2 Comprehensive Test Suite"
	@echo "📋 Test Configuration: tests/test_config.yaml"
	@go run tests/test_runner.go --verbose

test-unit-comprehensive: ## Run comprehensive unit tests with >90% coverage
	@echo "🔬 Running comprehensive unit tests..."
	@go test -v -race -coverprofile=coverage-unit.out -covermode=atomic \
		-timeout=5m ./internal/application/services/tests/...
	@echo "📊 Unit test coverage:"
	@go tool cover -func=coverage-unit.out | tail -1

test-recipe-resolver: ## Test recipe resolver with <10ms performance target
	@echo "⚡ Testing recipe resolver performance (<10ms target)..."
	@go test -v -race -timeout=2m \
		-run TestRecipeResolverService_Performance \
		./internal/application/services/tests/
	@go test -v -bench=BenchmarkRecipeResolverService \
		-benchtime=10s ./internal/application/services/tests/

test-integration-comprehensive: dev-infrastructure ## Run comprehensive integration tests
	@echo "🔄 Running comprehensive integration tests..."
	@echo "⏱️  Target: <250ms end-to-end performance"
	@sleep 5  # Wait for infrastructure
	@go test -v -race -tags=integration -timeout=15m \
		-coverprofile=coverage-integration.out \
		./tests/integration/...

test-performance-comprehensive: dev-infrastructure ## Run comprehensive performance tests
	@echo "🚀 Running performance tests with targets:"
	@echo "   • <250ms end-to-end response time"
	@echo "   • 1000+ RPS sustained throughput" 
	@echo "   • <512MB memory usage"
	@echo "   • <10ms recipe resolution"
	@go test -v -tags=performance -timeout=30m \
		./tests/performance/...

test-clinical-safety: ## Run clinical safety and FHIR compliance tests
	@echo "🏥 Running clinical safety tests..."
	@echo "   • Drug interaction detection"
	@echo "   • Dosage calculation accuracy"
	@echo "   • FHIR R4 compliance validation"
	@echo "   • Clinical decision support rules"
	@go test -v -race -tags=clinical -timeout=10m \
		./tests/clinical/...

test-security-compliance: ## Run security and HIPAA compliance tests
	@echo "🔒 Running security compliance tests..."
	@echo "   • JWT authentication validation"
	@echo "   • Role-based authorization"
	@echo "   • HIPAA audit trail compliance"
	@echo "   • Input sanitization and validation"
	@go test -v -race -tags=security -timeout=10m \
		./tests/security/...

test-load: dev-infrastructure ## Run load testing with 1000+ RPS target
	@echo "📈 Running load tests (1000+ RPS target)..."
	@echo "⚠️  WARNING: This will generate significant load on the system"
	@go test -v -tags=performance -timeout=30m \
		-run TestThroughputUnderLoad \
		./tests/performance/...

test-memory: ## Run memory usage tests (<512MB target)
	@echo "💾 Running memory usage tests..."
	@go test -v -tags=performance -timeout=10m \
		-run TestMemoryUsageUnderLoad \
		./tests/performance/...

test-cache-performance: ## Test caching effectiveness (30%+ improvement)
	@echo "⚡ Testing cache performance..."
	@go test -v -tags=performance -timeout=5m \
		-run TestCachePerformanceImprovement \
		./tests/performance/...

# =============================================================================
# HEALTHCARE-SPECIFIC TESTING
# =============================================================================

test-drug-interactions: ## Test drug interaction detection
	@echo "💊 Testing drug interaction detection..."
	@go test -v -race -tags=clinical -timeout=5m \
		-run TestDrugInteractionDetection \
		./tests/clinical/...

test-dosage-accuracy: ## Test dosage calculation accuracy
	@echo "⚗️  Testing dosage calculation accuracy..."
	@go test -v -race -tags=clinical -timeout=5m \
		-run TestDosageCalculationAccuracy \
		./tests/clinical/...

test-fhir-compliance: ## Test FHIR R4 compliance
	@echo "📋 Testing FHIR R4 compliance..."
	@go test -v -race -tags=clinical -timeout=5m \
		-run TestFHIRResourceCompliance \
		./tests/clinical/...

test-patient-safety: ## Test patient safety rules
	@echo "🛡️  Testing patient safety rules..."
	@go test -v -race -tags=clinical -timeout=10m \
		-run "TestAllergyContraindicationChecks|TestAgeBasedSafetyChecks|TestOrganFunctionBasedAdjustments" \
		./tests/clinical/...

test-clinical-workflows: ## Test complete clinical workflows
	@echo "🏥 Testing clinical workflows..."
	@go test -v -race -tags=clinical -timeout=10m \
		-run "TestPediatricPatientWorkflow|TestRenalImpairedPatientWorkflow" \
		./tests/integration/...

# =============================================================================
# PERFORMANCE VALIDATION
# =============================================================================

validate-performance: ## Validate all performance targets
	@echo "🎯 Validating performance targets..."
	@echo "Running performance validation suite..."
	@go test -v -tags=performance -timeout=30m \
		-run "TestMedicationProposalEndToEndPerformance|TestRecipeResolverPerformance|TestThroughputUnderLoad|TestMemoryUsageUnderLoad" \
		./tests/performance/...

benchmark-all: ## Run all benchmarks
	@echo "⚡ Running comprehensive benchmarks..."
	@go test -bench=. -benchmem -timeout=10m \
		./internal/application/services/tests/...
	@go test -bench=. -benchmem -timeout=10m \
		./tests/performance/...

profile-memory: dev-infrastructure ## Profile memory usage
	@echo "🔬 Profiling memory usage..."
	@go test -memprofile=mem.prof -bench=BenchmarkMedicationService \
		./internal/application/services/tests/
	@go tool pprof -http=:8080 mem.prof

profile-cpu: dev-infrastructure ## Profile CPU usage
	@echo "🔬 Profiling CPU usage..."
	@go test -cpuprofile=cpu.prof -bench=BenchmarkMedicationService \
		./internal/application/services/tests/
	@go tool pprof -http=:8080 cpu.prof

# =============================================================================
# SECURITY TESTING
# =============================================================================

test-auth: ## Test authentication and authorization
	@echo "🔐 Testing authentication and authorization..."
	@go test -v -race -tags=security -timeout=5m \
		-run "TestJWTAuthenticationValidation|TestRoleBasedAuthorization" \
		./tests/security/...

test-hipaa-compliance: ## Test HIPAA compliance
	@echo "📋 Testing HIPAA compliance..."
	@go test -v -race -tags=security -timeout=5m \
		-run TestHIPAAAuditCompliance \
		./tests/security/...

test-input-validation: ## Test input sanitization and validation
	@echo "🛡️  Testing input validation..."
	@go test -v -race -tags=security -timeout=5m \
		-run TestInputSanitizationAndValidation \
		./tests/security/...

test-data-protection: ## Test data encryption and protection
	@echo "🔒 Testing data protection..."
	@go test -v -race -tags=security -timeout=5m \
		-run TestDataEncryptionInTransit \
		./tests/security/...

# =============================================================================
# COVERAGE AND REPORTING
# =============================================================================

coverage-comprehensive: ## Generate comprehensive coverage report
	@echo "📊 Generating comprehensive coverage report..."
	@go test -coverprofile=coverage-unit.out ./internal/...
	@go test -tags=integration -coverprofile=coverage-integration.out ./tests/integration/...
	@echo "mode: atomic" > coverage-combined.out
	@grep -h -v "mode: atomic" coverage-unit.out coverage-integration.out >> coverage-combined.out
	@go tool cover -html=coverage-combined.out -o coverage-comprehensive.html
	@go tool cover -func=coverage-combined.out | tail -1
	@echo "📋 Comprehensive coverage report: coverage-comprehensive.html"

coverage-check: ## Check coverage meets minimum requirements
	@echo "✅ Checking coverage requirements..."
	@go test -coverprofile=coverage-check.out ./internal/...
	@go tool cover -func=coverage-check.out | tail -1 | awk '{print $$3}' | sed 's/%//' | \
		awk '{if($$1 < 90) {print "❌ Coverage " $$1 "% is below 90% requirement"; exit 1} else {print "✅ Coverage " $$1 "% meets requirement"}}'

test-report: ## Generate comprehensive test report
	@echo "📋 Generating test report..."
	@mkdir -p test-results
	@go run tests/test_runner.go --output test-results/test-report.json
	@echo "📊 Test report generated: test-results/test-report.json"

# =============================================================================
# CI/CD INTEGRATION
# =============================================================================

ci-test: ## Run tests in CI environment
	@echo "🤖 Running CI test suite..."
	@export SKIP_INTEGRATION_TESTS=false
	@export SKIP_PERFORMANCE_TESTS=false
	@export SKIP_CLINICAL_TESTS=false
	@export SKIP_SECURITY_TESTS=false
	@go run tests/test_runner.go --verbose --output ci-results.json

ci-test-quick: ## Run quick CI tests (unit + basic integration)
	@echo "⚡ Running quick CI test suite..."
	@go test -v -race -timeout=10m ./internal/...
	@go test -v -race -tags=integration -timeout=10m -short ./tests/integration/...

pre-commit: lint audit test-unit coverage-check ## Run pre-commit checks
	@echo "✅ All pre-commit checks passed!"

# =============================================================================
# TEST UTILITIES
# =============================================================================

test-setup: ## Setup test environment
	@echo "🔧 Setting up test environment..."
	@docker-compose -f deployments/docker-compose.test.yml up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 15
	@echo "✅ Test environment ready"

test-cleanup: ## Clean up test environment
	@echo "🧹 Cleaning up test environment..."
	@docker-compose -f deployments/docker-compose.test.yml down -v
	@rm -f coverage-*.out *.prof test-results/*
	@echo "✅ Test environment cleaned"

test-db-reset: ## Reset test database
	@echo "🔄 Resetting test database..."
	@docker-compose -f deployments/docker-compose.test.yml down postgres-test
	@docker-compose -f deployments/docker-compose.test.yml up -d postgres-test
	@sleep 5
	@echo "✅ Test database reset"

test-help: ## Show detailed test help
	@echo "🧪 Medication Service V2 - Comprehensive Test Suite"
	@echo ""
	@echo "PERFORMANCE TARGETS:"
	@echo "  • End-to-end: <250ms (95th percentile)"
	@echo "  • Recipe resolution: <10ms average"
	@echo "  • Throughput: 1000+ RPS sustained"
	@echo "  • Memory usage: <512MB per instance"
	@echo "  • Coverage: >90% unit tests, >85% overall"
	@echo ""
	@echo "HEALTHCARE COMPLIANCE:"
	@echo "  • FHIR R4 compliance validation"
	@echo "  • Clinical safety rule verification"
	@echo "  • Drug interaction detection"
	@echo "  • HIPAA audit trail compliance"
	@echo ""
	@echo "SECURITY VALIDATION:"
	@echo "  • JWT authentication & authorization"
	@echo "  • Role-based access control (RBAC)"
	@echo "  • Input sanitization & validation"
	@echo "  • Data encryption in transit"
	@echo ""
	@echo "USAGE EXAMPLES:"
	@echo "  make test-comprehensive     # Run all test suites"
	@echo "  make test-clinical-safety   # Clinical safety only"
	@echo "  make validate-performance   # Performance targets only"
	@echo "  make test-security-compliance # Security tests only"
	@echo "  make coverage-comprehensive # Generate coverage report"

# Quick aliases for common test scenarios
test-quick: test-unit ## Quick unit tests only
test-safety: test-clinical-safety ## Alias for clinical safety tests
test-perf: validate-performance ## Alias for performance validation
test-auth-only: test-auth ## Alias for authentication tests
test-complete: test-comprehensive coverage-comprehensive test-report ## Complete test suite with reporting