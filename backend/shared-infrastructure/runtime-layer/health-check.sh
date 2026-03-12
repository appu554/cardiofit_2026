#!/bin/bash

# ============================================
# CardioFit Runtime Layer Health Check Script
# ============================================
# This script verifies all runtime layer services are healthy

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Configuration
TIMEOUT=10
VERBOSE=false
JSON_OUTPUT=false

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

# Function to print colored output
print_status() {
    if [ "$JSON_OUTPUT" != true ]; then
        echo -e "${BLUE}[$(date +'%H:%M:%S')] $1${NC}"
    fi
}

print_success() {
    if [ "$JSON_OUTPUT" != true ]; then
        echo -e "${GREEN}[$(date +'%H:%M:%S')] ✅ $1${NC}"
    fi
}

print_warning() {
    if [ "$JSON_OUTPUT" != true ]; then
        echo -e "${YELLOW}[$(date +'%H:%M:%S')] ⚠️  $1${NC}"
    fi
}

print_error() {
    if [ "$JSON_OUTPUT" != true ]; then
        echo -e "${RED}[$(date +'%H:%M:%S')] ❌ $1${NC}"
    fi
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to test HTTP endpoint
test_http_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="${3:-200}"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if [ "$VERBOSE" = true ]; then
        print_status "Testing $name at $url..."
    fi

    if command_exists curl; then
        local response=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout $TIMEOUT "$url" 2>/dev/null || echo "000")

        if [ "$response" = "$expected_status" ]; then
            print_success "$name is healthy (HTTP $response)"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            print_error "$name health check failed (HTTP $response, expected $expected_status)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    else
        print_warning "curl not available, skipping HTTP check for $name"
        return 0
    fi
}

# Function to test TCP port
test_tcp_port() {
    local name="$1"
    local host="$2"
    local port="$3"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if [ "$VERBOSE" = true ]; then
        print_status "Testing $name TCP connection to $host:$port..."
    fi

    if command_exists nc; then
        if nc -z -w$TIMEOUT "$host" "$port" 2>/dev/null; then
            print_success "$name is reachable on $host:$port"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            print_error "$name is not reachable on $host:$port"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    elif command_exists telnet; then
        if timeout $TIMEOUT telnet "$host" "$port" </dev/null &>/dev/null; then
            print_success "$name is reachable on $host:$port"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            print_error "$name is not reachable on $host:$port"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    else
        print_warning "nc/telnet not available, skipping TCP check for $name"
        return 0
    fi
}

# Function to test Redis
test_redis() {
    local name="Redis"
    local host="localhost"
    local port="6379"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if [ "$VERBOSE" = true ]; then
        print_status "Testing $name with PING command..."
    fi

    if command_exists redis-cli; then
        local response=$(timeout $TIMEOUT redis-cli -h "$host" -p "$port" ping 2>/dev/null || echo "FAILED")

        if [ "$response" = "PONG" ]; then
            print_success "$name is healthy (PONG received)"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            print_error "$name health check failed (no PONG response)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    else
        # Fallback to TCP test
        test_tcp_port "$name" "$host" "$port"
    fi
}

# Function to test Neo4j
test_neo4j() {
    local name="Neo4j"

    # Test HTTP interface
    test_http_endpoint "$name HTTP" "http://localhost:7474"

    # Test Bolt port
    test_tcp_port "$name Bolt" "localhost" "7687"
}

# Function to test Kafka
test_kafka() {
    local name="Kafka"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if [ "$VERBOSE" = true ]; then
        print_status "Testing $name with topic list..."
    fi

    if docker ps --format "table {{.Names}}" | grep -q "runtime-kafka"; then
        local response=$(timeout $TIMEOUT docker exec runtime-kafka kafka-topics --bootstrap-server localhost:9092 --list 2>/dev/null || echo "FAILED")

        if [ "$response" != "FAILED" ]; then
            print_success "$name is healthy (topic list retrieved)"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            print_error "$name health check failed (cannot list topics)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    else
        print_error "$name container not running"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

# Function to test MongoDB
test_mongodb() {
    local name="$1"
    local container="$2"
    local port="$3"

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if [ "$VERBOSE" = true ]; then
        print_status "Testing $name with ping command..."
    fi

    if docker ps --format "table {{.Names}}" | grep -q "$container"; then
        local response=$(timeout $TIMEOUT docker exec "$container" mongosh --eval "db.adminCommand('ping')" --quiet 2>/dev/null || echo "FAILED")

        if echo "$response" | grep -q '"ok".*1'; then
            print_success "$name is healthy (ping successful)"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            print_error "$name health check failed (ping failed)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    else
        # Fallback to TCP test
        test_tcp_port "$name" "localhost" "$port"
    fi
}

# Function to check Docker containers
check_containers() {
    local containers=(
        "runtime-neo4j"
        "runtime-graphdb"
        "runtime-zookeeper"
        "runtime-kafka"
        "runtime-redis"
        "runtime-clickhouse"
        "runtime-flink-jobmanager"
        "runtime-flink-taskmanager"
        "runtime-query-router"
        "runtime-cache-prefetcher"
        "runtime-evidence-envelope"
        "runtime-mongodb-evidence"
        "runtime-sla-monitoring"
        "runtime-mongodb-sla"
        "runtime-prometheus"
        "runtime-grafana"
    )

    print_status "Checking Docker containers..."

    for container in "${containers[@]}"; do
        TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

        local status=$(docker ps -a --format "table {{.Names}}\t{{.Status}}" | grep "^$container" | awk '{print $2}' || echo "missing")

        if [ "$status" = "Up" ]; then
            if [ "$VERBOSE" = true ]; then
                print_success "Container $container is running"
            fi
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
        elif [ "$status" = "missing" ]; then
            print_warning "Container $container is not created"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        else
            print_error "Container $container is not running (status: $status)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi
    done
}

# Function to run all health checks
run_health_checks() {
    print_status "Running comprehensive health checks..."
    echo ""

    # Check Docker containers first
    check_containers
    echo ""

    # Core Infrastructure
    print_status "Checking core infrastructure services..."
    test_neo4j
    test_http_endpoint "GraphDB" "http://localhost:7200/rest/info"
    test_redis
    test_kafka
    test_http_endpoint "ClickHouse" "http://localhost:8123/ping" "200"
    echo ""

    # Stream Processing
    print_status "Checking stream processing services..."
    test_http_endpoint "Flink JobManager" "http://localhost:8081/config"
    echo ""

    # Application Services
    print_status "Checking application services..."
    test_http_endpoint "Query Router" "http://localhost:8070/health"
    test_http_endpoint "Cache Prefetcher" "http://localhost:8055/health"
    test_http_endpoint "Evidence Envelope" "http://localhost:8060/health"
    test_mongodb "MongoDB Evidence" "runtime-mongodb-evidence" "27018"
    echo ""

    # Monitoring Services
    print_status "Checking monitoring services..."
    test_http_endpoint "SLA Monitoring" "http://localhost:8050/health"
    test_mongodb "MongoDB SLA" "runtime-mongodb-sla" "27019"
    test_http_endpoint "Prometheus" "http://localhost:9090/-/healthy"
    test_http_endpoint "Grafana" "http://localhost:3000/api/health"
    echo ""
}

# Function to show detailed service information
show_detailed_info() {
    print_status "Detailed Service Information:"
    echo ""

    # Service endpoints
    local services=(
        "Neo4j Browser|http://localhost:7474"
        "GraphDB Workbench|http://localhost:7200"
        "Flink Dashboard|http://localhost:8081"
        "Query Router API|http://localhost:8070"
        "Cache Prefetcher API|http://localhost:8055"
        "Evidence Envelope API|http://localhost:8060"
        "SLA Monitoring API|http://localhost:8050"
        "Prometheus|http://localhost:9090"
        "Grafana|http://localhost:3000"
    )

    for service in "${services[@]}"; do
        local name=$(echo "$service" | cut -d'|' -f1)
        local url=$(echo "$service" | cut -d'|' -f2)
        echo -e "${BLUE}$name:${NC} $url"
    done
    echo ""

    # Resource usage
    if command_exists docker; then
        print_status "Resource Usage:"
        docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" | grep "runtime-"
        echo ""
    fi
}

# Function to generate JSON report
generate_json_report() {
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local status="healthy"

    if [ $FAILED_CHECKS -gt 0 ]; then
        status="unhealthy"
    elif [ $PASSED_CHECKS -eq 0 ]; then
        status="unknown"
    fi

    cat << EOF
{
  "timestamp": "$timestamp",
  "status": "$status",
  "summary": {
    "total_checks": $TOTAL_CHECKS,
    "passed": $PASSED_CHECKS,
    "failed": $FAILED_CHECKS
  },
  "health_score": $(echo "scale=2; $PASSED_CHECKS * 100 / $TOTAL_CHECKS" | bc 2>/dev/null || echo "0"),
  "runtime_layer": {
    "version": "1.0.0",
    "deployment": "docker-compose"
  }
}
EOF
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v, --verbose     Show detailed check information"
    echo "  -j, --json        Output results in JSON format"
    echo "  -t, --timeout N   Set connection timeout in seconds (default: $TIMEOUT)"
    echo "  -i, --info        Show detailed service information"
    echo "  -q, --quick       Run only basic checks"
    echo "  -h, --help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                Run all health checks"
    echo "  $0 -v             Run with verbose output"
    echo "  $0 -j             Output JSON report"
    echo "  $0 -i             Show service information only"
    echo ""
}

# Main execution
main() {
    local SHOW_INFO_ONLY=false
    local QUICK_CHECK=false

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -j|--json)
                JSON_OUTPUT=true
                shift
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -i|--info)
                SHOW_INFO_ONLY=true
                shift
                ;;
            -q|--quick)
                QUICK_CHECK=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    if [ "$SHOW_INFO_ONLY" = true ]; then
        show_detailed_info
        exit 0
    fi

    if [ "$JSON_OUTPUT" != true ]; then
        print_status "CardioFit Runtime Layer Health Check"
        echo ""
    fi

    # Run health checks
    if [ "$QUICK_CHECK" = true ]; then
        # Quick check - only test HTTP endpoints
        test_http_endpoint "Neo4j" "http://localhost:7474"
        test_http_endpoint "GraphDB" "http://localhost:7200/rest/info"
        test_http_endpoint "Flink" "http://localhost:8081/config"
        test_http_endpoint "Query Router" "http://localhost:8070/health"
    else
        run_health_checks
    fi

    # Generate summary
    if [ "$JSON_OUTPUT" = true ]; then
        generate_json_report
    else
        echo ""
        print_status "Health Check Summary:"
        echo -e "${BLUE}Total Checks:  ${NC}$TOTAL_CHECKS"
        echo -e "${GREEN}Passed:        ${NC}$PASSED_CHECKS"

        if [ $FAILED_CHECKS -gt 0 ]; then
            echo -e "${RED}Failed:        ${NC}$FAILED_CHECKS"
        else
            echo -e "${GREEN}Failed:        ${NC}$FAILED_CHECKS"
        fi

        local health_score=$(echo "scale=1; $PASSED_CHECKS * 100 / $TOTAL_CHECKS" | bc 2>/dev/null || echo "0")
        echo -e "${BLUE}Health Score:  ${NC}${health_score}%"
        echo ""

        if [ $FAILED_CHECKS -eq 0 ]; then
            print_success "All services are healthy! 🎉"
        elif [ $FAILED_CHECKS -lt $((TOTAL_CHECKS / 2)) ]; then
            print_warning "Some services need attention"
        else
            print_error "Multiple services are unhealthy"
        fi

        if [ "$VERBOSE" != true ] && [ $FAILED_CHECKS -gt 0 ]; then
            echo ""
            print_status "For detailed information, run: $0 --verbose"
            print_status "To view service logs, run: docker-compose logs [service-name]"
        fi
    fi

    # Exit with appropriate code
    if [ $FAILED_CHECKS -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function with all arguments
main "$@"