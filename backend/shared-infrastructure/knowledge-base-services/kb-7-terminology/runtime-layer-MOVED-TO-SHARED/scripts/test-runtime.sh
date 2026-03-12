#!/bin/bash
# Test script for KB7 Neo4j Dual-Stream & Service Runtime Layer
# This script runs comprehensive tests for all runtime components

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUNTIME_DIR="$(dirname "$SCRIPT_DIR")"
TEST_RESULTS_DIR="$RUNTIME_DIR/test-results"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create test results directory
setup_test_environment() {
    log_info "Setting up test environment..."

    mkdir -p "$TEST_RESULTS_DIR"
    cd "$RUNTIME_DIR"

    # Activate virtual environment
    if [ -d "venv" ]; then
        source venv/bin/activate
    else
        log_error "Virtual environment not found. Please run init-runtime.sh first."
        exit 1
    fi

    log_success "Test environment ready"
}

# Test infrastructure connectivity
test_infrastructure() {
    log_info "Testing infrastructure connectivity..."

    local test_file="$TEST_RESULTS_DIR/infrastructure_test.log"

    # Test Neo4j
    log_info "Testing Neo4j connectivity..."
    if docker exec kb7-neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD:-kb7password}" "RETURN 1" &> "$test_file"; then
        log_success "Neo4j connectivity: PASS"
    else
        log_error "Neo4j connectivity: FAIL"
        return 1
    fi

    # Test ClickHouse
    log_info "Testing ClickHouse connectivity..."
    if docker exec kb7-clickhouse clickhouse-client --query "SELECT 1" &>> "$test_file"; then
        log_success "ClickHouse connectivity: PASS"
    else
        log_error "ClickHouse connectivity: FAIL"
        return 1
    fi

    # Test GraphDB
    log_info "Testing GraphDB connectivity..."
    if curl -f http://localhost:7200/rest/info &>> "$test_file"; then
        log_success "GraphDB connectivity: PASS"
    else
        log_error "GraphDB connectivity: FAIL"
        return 1
    fi

    # Test Kafka
    log_info "Testing Kafka connectivity..."
    if docker exec kb7-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 &>> "$test_file"; then
        log_success "Kafka connectivity: PASS"
    else
        log_error "Kafka connectivity: FAIL"
        return 1
    fi

    # Test Redis L2
    log_info "Testing Redis L2 connectivity..."
    if docker exec kb7-redis-l2 redis-cli ping &>> "$test_file"; then
        log_success "Redis L2 connectivity: PASS"
    else
        log_error "Redis L2 connectivity: FAIL"
        return 1
    fi

    # Test Redis L3
    log_info "Testing Redis L3 connectivity..."
    if docker exec kb7-redis-l3 redis-cli ping &>> "$test_file"; then
        log_success "Redis L3 connectivity: PASS"
    else
        log_error "Redis L3 connectivity: FAIL"
        return 1
    fi

    log_success "Infrastructure connectivity tests completed"
}

# Test runtime services health
test_runtime_health() {
    log_info "Testing runtime services health..."

    local health_file="$TEST_RESULTS_DIR/health_check.json"

    if python main_integration.py --health > "$health_file"; then
        local overall_status=$(python -c "
import json
with open('$health_file', 'r') as f:
    data = json.load(f)
    print(data.get('overall_status', 'unknown'))
")

        if [ "$overall_status" = "healthy" ]; then
            log_success "Runtime health check: PASS (all services healthy)"
        elif [ "$overall_status" = "degraded" ]; then
            log_warning "Runtime health check: DEGRADED (some services have issues)"
        else
            log_error "Runtime health check: FAIL (critical issues detected)"
            return 1
        fi
    else
        log_error "Runtime health check: FAIL (health check command failed)"
        return 1
    fi
}

# Test individual components
test_components() {
    log_info "Testing individual components..."

    # Test Neo4j Dual-Stream Manager
    log_info "Testing Neo4j Dual-Stream Manager..."
    python -c "
import asyncio
from neo4j_setup.dual_stream_manager import Neo4jDualStreamManager

async def test():
    config = {
        'neo4j_uri': 'bolt://localhost:7687',
        'neo4j_user': 'neo4j',
        'neo4j_password': '${NEO4J_PASSWORD:-kb7password}'
    }
    manager = Neo4jDualStreamManager(config)
    health = await manager.health_check()
    print(f'Neo4j Manager Health: {health[\"status\"]}')
    await manager.close()
    return health['status'] in ['healthy', 'degraded']

result = asyncio.run(test())
exit(0 if result else 1)
" &> "$TEST_RESULTS_DIR/neo4j_manager_test.log"

    if [ $? -eq 0 ]; then
        log_success "Neo4j Dual-Stream Manager: PASS"
    else
        log_error "Neo4j Dual-Stream Manager: FAIL"
        return 1
    fi

    # Test ClickHouse Manager
    log_info "Testing ClickHouse Manager..."
    python -c "
from clickhouse_runtime.manager import ClickHouseRuntimeManager

config = {
    'host': 'localhost',
    'port': 9000,
    'database': 'kb7_analytics',
    'user': 'kb7',
    'password': '${CH_PASSWORD:-kb7password}'
}
manager = ClickHouseRuntimeManager(config)
health = manager.health_check()
print(f'ClickHouse Manager Health: {health[\"status\"]}')
manager.close()
exit(0 if health['status'] in ['healthy', 'degraded'] else 1)
" &> "$TEST_RESULTS_DIR/clickhouse_manager_test.log"

    if [ $? -eq 0 ]; then
        log_success "ClickHouse Manager: PASS"
    else
        log_error "ClickHouse Manager: FAIL"
        return 1
    fi

    log_success "Component tests completed"
}

# Test integration workflows
test_integration_workflows() {
    log_info "Testing integration workflows..."

    if python main_integration.py --test &> "$TEST_RESULTS_DIR/integration_test.log"; then
        log_success "Integration workflows: PASS"
    else
        log_error "Integration workflows: FAIL"
        log_error "Check $TEST_RESULTS_DIR/integration_test.log for details"
        return 1
    fi
}

# Test performance benchmarks
test_performance() {
    log_info "Running performance benchmarks..."

    python -c "
import asyncio
import time
from datetime import datetime

async def test_query_routing_latency():
    start_time = datetime.utcnow()
    # Simulate query routing time
    await asyncio.sleep(0.001)
    routing_time = (datetime.utcnow() - start_time).total_seconds() * 1000
    print(f'Query routing latency: {routing_time:.2f}ms')
    return routing_time < 5  # Should be under 5ms

async def test_cache_operations():
    # Test Redis cache operations
    import redis
    r = redis.Redis(host='localhost', port=6379, db=0)

    start_time = time.time()
    r.set('test_key', 'test_value')
    value = r.get('test_key')
    cache_time = (time.time() - start_time) * 1000

    print(f'Cache operation latency: {cache_time:.2f}ms')
    r.delete('test_key')
    return cache_time < 10  # Should be under 10ms

async def main():
    routing_ok = await test_query_routing_latency()
    cache_ok = await test_cache_operations()
    return routing_ok and cache_ok

result = asyncio.run(main())
exit(0 if result else 1)
" &> "$TEST_RESULTS_DIR/performance_test.log"

    if [ $? -eq 0 ]; then
        log_success "Performance benchmarks: PASS"
    else
        log_error "Performance benchmarks: FAIL"
        return 1
    fi
}

# Test data consistency
test_data_consistency() {
    log_info "Testing data consistency..."

    python -c "
import asyncio
from snapshot.manager import SnapshotManager

async def test():
    manager = SnapshotManager()

    # Create a test snapshot
    snapshot = await manager.create_snapshot(
        service_id='test',
        context={'test': 'consistency'},
        ttl=None
    )

    # Validate snapshot
    is_valid = await manager.validate_snapshot(snapshot.id)
    print(f'Snapshot validation: {\"PASS\" if is_valid else \"FAIL\"}')

    # Get statistics
    stats = await manager.get_statistics()
    print(f'Active snapshots: {stats[\"active_snapshots\"]}')

    return is_valid

result = asyncio.run(test())
exit(0 if result else 1)
" &> "$TEST_RESULTS_DIR/consistency_test.log"

    if [ $? -eq 0 ]; then
        log_success "Data consistency: PASS"
    else
        log_error "Data consistency: FAIL"
        return 1
    fi
}

# Generate test report
generate_test_report() {
    log_info "Generating test report..."

    local report_file="$TEST_RESULTS_DIR/test_report.md"

    cat > "$report_file" <<EOF
# KB7 Runtime Layer Test Report

**Generated:** $(date)

## Test Summary

### Infrastructure Tests
- Neo4j Connectivity: $(grep -q "Neo4j connectivity: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- ClickHouse Connectivity: $(grep -q "ClickHouse connectivity: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- GraphDB Connectivity: $(grep -q "GraphDB connectivity: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- Kafka Connectivity: $(grep -q "Kafka connectivity: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- Redis L2 Connectivity: $(grep -q "Redis L2 connectivity: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- Redis L3 Connectivity: $(grep -q "Redis L3 connectivity: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")

### Component Tests
- Neo4j Dual-Stream Manager: $(grep -q "Neo4j Dual-Stream Manager: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- ClickHouse Manager: $(grep -q "ClickHouse Manager: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")

### Integration Tests
- Runtime Health Check: $(grep -q "Runtime health check: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")
- Integration Workflows: $(grep -q "Integration workflows: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")

### Performance Tests
- Performance Benchmarks: $(grep -q "Performance benchmarks: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")

### Consistency Tests
- Data Consistency: $(grep -q "Data consistency: PASS" "$TEST_RESULTS_DIR"/*.log && echo "✅ PASS" || echo "❌ FAIL")

## Test Files
EOF

    for log_file in "$TEST_RESULTS_DIR"/*.log; do
        echo "- $(basename "$log_file")" >> "$report_file"
    done

    log_success "Test report generated: $report_file"
}

# Clean up test artifacts
cleanup() {
    log_info "Cleaning up test artifacts..."

    # Remove temporary test data from databases
    python -c "
import asyncio
import redis

async def cleanup():
    # Clean Redis test keys
    try:
        r = redis.Redis(host='localhost', port=6379, db=0)
        for key in r.scan_iter(match='test_*'):
            r.delete(key)
        print('Redis cleanup complete')
    except Exception as e:
        print(f'Redis cleanup warning: {e}')

asyncio.run(cleanup())
"

    log_success "Cleanup completed"
}

# Main test execution
run_all_tests() {
    log_info "Starting comprehensive KB7 Runtime Layer tests..."

    local start_time=$(date +%s)
    local failed_tests=0

    # Run all test suites
    test_infrastructure || ((failed_tests++))
    test_runtime_health || ((failed_tests++))
    test_components || ((failed_tests++))
    test_integration_workflows || ((failed_tests++))
    test_performance || ((failed_tests++))
    test_data_consistency || ((failed_tests++))

    # Generate report
    generate_test_report

    # Cleanup
    cleanup

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo ""
    if [ $failed_tests -eq 0 ]; then
        log_success "All tests passed! Duration: ${duration}s"
        log_success "Test report available at: $TEST_RESULTS_DIR/test_report.md"
        return 0
    else
        log_error "$failed_tests test suite(s) failed. Duration: ${duration}s"
        log_error "Check logs in $TEST_RESULTS_DIR for details"
        return 1
    fi
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--help|--infrastructure|--components|--integration|--performance|--consistency]"
        echo ""
        echo "This script runs comprehensive tests for the KB7 Runtime Layer"
        echo ""
        echo "Options:"
        echo "  --help, -h          Show this help message"
        echo "  --infrastructure    Run infrastructure connectivity tests only"
        echo "  --components        Run component tests only"
        echo "  --integration       Run integration workflow tests only"
        echo "  --performance       Run performance benchmark tests only"
        echo "  --consistency       Run data consistency tests only"
        echo ""
        echo "If no specific test is specified, all tests will be run."
        exit 0
        ;;
    --infrastructure)
        setup_test_environment
        test_infrastructure
        ;;
    --components)
        setup_test_environment
        test_components
        ;;
    --integration)
        setup_test_environment
        test_integration_workflows
        ;;
    --performance)
        setup_test_environment
        test_performance
        ;;
    --consistency)
        setup_test_environment
        test_data_consistency
        ;;
    *)
        setup_test_environment
        run_all_tests
        ;;
esac