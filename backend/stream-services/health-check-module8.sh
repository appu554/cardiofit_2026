#!/bin/bash

# ========================================
# Module 8 Health Check Script
# ========================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.module8-complete.yml"

# Service ports
declare -A SERVICE_PORTS=(
    ["PostgreSQL Projector"]=8050
    ["MongoDB Projector"]=8051
    ["Elasticsearch Projector"]=8052
    ["ClickHouse Projector"]=8053
    ["InfluxDB Projector"]=8054
    ["UPS Projector"]=8055
    ["FHIR Store Projector"]=8056
    ["Neo4j Graph Projector"]=8057
)

# Infrastructure ports
declare -A INFRA_PORTS=(
    ["MongoDB"]=27017
    ["Elasticsearch"]=9200
    ["ClickHouse"]=8123
    ["Redis"]=6379
)

# ========================================
# Helper Functions
# ========================================

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

# ========================================
# Service Health Checks
# ========================================

check_service_health() {
    local service_name=$1
    local port=$2
    local response

    if response=$(curl -sf "http://localhost:$port/health" 2>&1); then
        # Parse JSON response for detailed stats
        local status=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', 'unknown'))" 2>/dev/null || echo "ok")
        local messages_processed=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('messages_processed', 0))" 2>/dev/null || echo "0")
        local uptime=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('uptime', 'N/A'))" 2>/dev/null || echo "N/A")

        printf "${GREEN}✅ %-25s${NC} Port: %-5s Status: %-10s Processed: %-8s Uptime: %s\n" \
            "$service_name" "$port" "$status" "$messages_processed" "$uptime"
        return 0
    else
        printf "${RED}❌ %-25s${NC} Port: %-5s ${RED}NOT RESPONDING${NC}\n" "$service_name" "$port"
        return 1
    fi
}

check_projector_services() {
    print_header "Storage Projector Services"

    local healthy=0
    local total=${#SERVICE_PORTS[@]}

    for service in "${!SERVICE_PORTS[@]}"; do
        if check_service_health "$service" "${SERVICE_PORTS[$service]}"; then
            ((healthy++))
        fi
    done

    echo ""
    print_info "Projector Health: $healthy/$total services healthy"
    return 0
}

# ========================================
# Infrastructure Health Checks
# ========================================

check_mongodb() {
    if curl -sf "http://localhost:27017" > /dev/null 2>&1 || nc -z localhost 27017 2>/dev/null; then
        print_success "MongoDB (27017) - Running"
        return 0
    else
        print_error "MongoDB (27017) - Not responding"
        return 1
    fi
}

check_elasticsearch() {
    if response=$(curl -sf "http://localhost:9200/_cluster/health" 2>&1); then
        local status=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', 'unknown'))" 2>/dev/null || echo "unknown")
        local color=$(echo "$response" | python3 -c "import sys, json; print(json.load(sys.stdin).get('status', 'unknown'))" 2>/dev/null || echo "unknown")

        if [[ "$status" == "green" ]]; then
            print_success "Elasticsearch (9200) - Status: $status"
        elif [[ "$status" == "yellow" ]]; then
            print_warning "Elasticsearch (9200) - Status: $status"
        else
            print_error "Elasticsearch (9200) - Status: $status"
        fi
        return 0
    else
        print_error "Elasticsearch (9200) - Not responding"
        return 1
    fi
}

check_clickhouse() {
    if response=$(curl -sf "http://localhost:8123/ping" 2>&1); then
        print_success "ClickHouse (8123) - Running"
        return 0
    else
        print_error "ClickHouse (8123) - Not responding"
        return 1
    fi
}

check_redis() {
    if redis-cli -p 6379 ping > /dev/null 2>&1 || nc -z localhost 6379 2>/dev/null; then
        print_success "Redis (6379) - Running"
        return 0
    else
        print_error "Redis (6379) - Not responding"
        return 1
    fi
}

check_infrastructure() {
    print_header "Infrastructure Services"

    local healthy=0
    local total=4

    check_mongodb && ((healthy++))
    check_elasticsearch && ((healthy++))
    check_clickhouse && ((healthy++))
    check_redis && ((healthy++))

    echo ""
    print_info "Infrastructure Health: $healthy/$total services healthy"
    return 0
}

# ========================================
# External Container Checks
# ========================================

check_external_containers() {
    print_header "External Database Containers"

    # Check PostgreSQL (a2f55d83b1fa)
    if docker ps --format '{{.ID}}' | grep -q "^a2f55d83b1fa"; then
        local status=$(docker inspect --format='{{.State.Status}}' a2f55d83b1fa)
        print_success "PostgreSQL (a2f55d83b1fa) - Status: $status"
    else
        print_error "PostgreSQL (a2f55d83b1fa) - Not running"
    fi

    # Check InfluxDB (8502fd5d078d)
    if docker ps --format '{{.ID}}' | grep -q "^8502fd5d078d"; then
        local status=$(docker inspect --format='{{.State.Status}}' 8502fd5d078d)
        print_success "InfluxDB (8502fd5d078d) - Status: $status"
    else
        print_error "InfluxDB (8502fd5d078d) - Not running"
    fi

    # Check Neo4j (e8b3df4d8a02)
    if docker ps --format '{{.ID}}' | grep -q "^e8b3df4d8a02"; then
        local status=$(docker inspect --format='{{.State.Status}}' e8b3df4d8a02)
        print_success "Neo4j (e8b3df4d8a02) - Status: $status"
    else
        print_error "Neo4j (e8b3df4d8a02) - Not running"
    fi
}

# ========================================
# Kafka Consumer Lag Check
# ========================================

check_kafka_lag() {
    print_header "Kafka Consumer Lag"

    # Check if each projector's consumer group has lag
    print_info "Checking consumer group lag..."

    # Get container IDs for projectors
    local containers=$(docker-compose -f "$COMPOSE_FILE" ps -q 2>/dev/null)

    if [ -z "$containers" ]; then
        print_warning "No containers running - cannot check consumer lag"
        return 1
    fi

    # Check log files for lag information
    print_info "Checking recent logs for consumer lag indicators..."

    for service in postgresql-projector mongodb-projector elasticsearch-projector clickhouse-projector \
                   influxdb-projector ups-projector fhir-store-projector neo4j-graph-projector; do
        local lag_info=$(docker-compose -f "$COMPOSE_FILE" logs --tail=20 "$service" 2>/dev/null | grep -i "lag\|offset\|processed" | tail -1)
        if [ -n "$lag_info" ]; then
            echo "  $service: $lag_info"
        fi
    done
}

# ========================================
# Database Connection Tests
# ========================================

check_database_connections() {
    print_header "Database Connection Tests"

    # Test PostgreSQL connection
    if docker exec a2f55d83b1fa pg_isready -U cardiofit > /dev/null 2>&1; then
        print_success "PostgreSQL connection - OK"
    else
        print_error "PostgreSQL connection - Failed"
    fi

    # Test Neo4j connection
    if nc -z localhost 7687 2>/dev/null; then
        print_success "Neo4j connection - OK"
    else
        print_error "Neo4j connection - Failed"
    fi

    # Test InfluxDB connection
    if curl -sf "http://localhost:8086/health" > /dev/null 2>&1; then
        print_success "InfluxDB connection - OK"
    else
        print_error "InfluxDB connection - Failed"
    fi
}

# ========================================
# Container Resource Usage
# ========================================

check_resource_usage() {
    print_header "Container Resource Usage"

    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}" \
        $(docker-compose -f "$COMPOSE_FILE" ps -q 2>/dev/null) 2>/dev/null || print_warning "Unable to get resource stats"
}

# ========================================
# Generate Health Report
# ========================================

generate_health_report() {
    local report_file="${SCRIPT_DIR}/module8-health-report-$(date +%Y%m%d-%H%M%S).txt"

    print_header "Generating Health Report"

    {
        echo "Module 8 Health Report"
        echo "Generated: $(date)"
        echo "========================================"
        echo ""

        # Service health
        echo "PROJECTOR SERVICES:"
        for service in "${!SERVICE_PORTS[@]}"; do
            if curl -sf "http://localhost:${SERVICE_PORTS[$service]}/health" > /dev/null 2>&1; then
                echo "  ✅ $service (${SERVICE_PORTS[$service]}): HEALTHY"
            else
                echo "  ❌ $service (${SERVICE_PORTS[$service]}): UNHEALTHY"
            fi
        done
        echo ""

        # Infrastructure health
        echo "INFRASTRUCTURE SERVICES:"
        docker-compose -f "$COMPOSE_FILE" ps 2>/dev/null || echo "  Unable to get service status"
        echo ""

        # Resource usage
        echo "RESOURCE USAGE:"
        docker stats --no-stream --format "{{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
            $(docker-compose -f "$COMPOSE_FILE" ps -q 2>/dev/null) 2>/dev/null || echo "  Unable to get stats"
        echo ""

        # Recent errors
        echo "RECENT ERRORS (last 50 lines):"
        docker-compose -f "$COMPOSE_FILE" logs --tail=50 2>&1 | grep -i "error\|exception" || echo "  No errors found"

    } > "$report_file"

    print_success "Report saved to: $report_file"
}

# ========================================
# Main Execution
# ========================================

main() {
    print_header "🏥 Module 8 Health Check"

    # Run all checks
    check_projector_services
    check_infrastructure
    check_external_containers
    check_database_connections
    check_kafka_lag
    check_resource_usage

    # Generate report option
    echo ""
    echo -n "Generate detailed health report? [y/N]: "
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        generate_health_report
    fi

    print_header "✅ Health Check Complete"
}

# Run main function
main "$@"
