#!/bin/bash
set -e

# ========================================
# Module 8 Storage Projectors Startup Script
# ========================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env.module8"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.module8-complete.yml"
INFRASTRUCTURE_COMPOSE="${SCRIPT_DIR}/docker-compose.module8-infrastructure.yml"

# Service ports for health checks
declare -A SERVICE_PORTS=(
    ["postgresql-projector"]=8050
    ["mongodb-projector"]=8051
    ["elasticsearch-projector"]=8052
    ["clickhouse-projector"]=8053
    ["influxdb-projector"]=8054
    ["ups-projector"]=8055
    ["fhir-store-projector"]=8056
    ["neo4j-graph-projector"]=8057
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
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# ========================================
# Prerequisite Checks
# ========================================

check_prerequisites() {
    print_header "Checking Prerequisites"

    # Check if .env.module8 exists
    if [ ! -f "$ENV_FILE" ]; then
        print_error "Environment file not found: $ENV_FILE"
        print_info "Copy .env.module8.example to .env.module8 and configure"
        echo ""
        echo "  cp .env.module8.example .env.module8"
        echo "  # Edit .env.module8 with your credentials"
        echo ""
        exit 1
    fi
    print_success "Environment file found"

    # Load environment variables
    source "$ENV_FILE"

    # Check required Kafka credentials
    if [ -z "$KAFKA_BOOTSTRAP_SERVERS" ] || [ -z "$KAFKA_SASL_USERNAME" ] || [ -z "$KAFKA_SASL_PASSWORD" ]; then
        print_error "Missing Kafka credentials in $ENV_FILE"
        print_info "Required: KAFKA_BOOTSTRAP_SERVERS, KAFKA_SASL_USERNAME, KAFKA_SASL_PASSWORD"
        exit 1
    fi
    print_success "Kafka credentials configured"

    # Check PostgreSQL credentials
    if [ -z "$POSTGRES_PASSWORD" ]; then
        print_error "Missing POSTGRES_PASSWORD in $ENV_FILE"
        exit 1
    fi
    print_success "PostgreSQL credentials configured"

    # Check Google credentials for FHIR Store
    if [ -n "$GOOGLE_CREDENTIALS_PATH" ] && [ ! -f "$GOOGLE_CREDENTIALS_PATH" ]; then
        print_warning "Google credentials file not found: $GOOGLE_CREDENTIALS_PATH"
        print_info "FHIR Store projector may fail to start"
    else
        print_success "Google credentials configured"
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    print_success "Docker is installed"

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed"
        exit 1
    fi
    print_success "Docker Compose is installed"
}

# ========================================
# Database Container Checks
# ========================================

check_external_containers() {
    print_header "Checking External Database Containers"

    # Check PostgreSQL container (a2f55d83b1fa)
    if docker ps --format '{{.ID}}' | grep -q "^a2f55d83b1fa"; then
        print_success "PostgreSQL container (a2f55d83b1fa) is running"
    else
        print_warning "PostgreSQL container (a2f55d83b1fa) is not running"
        print_info "PostgreSQL projector and UPS projector may fail"
    fi

    # Check InfluxDB container (8502fd5d078d)
    if docker ps --format '{{.ID}}' | grep -q "^8502fd5d078d"; then
        print_success "InfluxDB container (8502fd5d078d) is running"
    else
        print_warning "InfluxDB container (8502fd5d078d) is not running"
        print_info "InfluxDB projector may fail"
    fi

    # Check Neo4j container (e8b3df4d8a02)
    if docker ps --format '{{.ID}}' | grep -q "^e8b3df4d8a02"; then
        print_success "Neo4j container (e8b3df4d8a02) is running"
    else
        print_warning "Neo4j container (e8b3df4d8a02) is not running"
        print_info "Neo4j Graph projector may fail"
    fi
}

# ========================================
# Network Setup
# ========================================

setup_network() {
    print_header "Setting Up Network"

    # Create module8-network if it doesn't exist
    if ! docker network ls | grep -q "module8-network"; then
        print_info "Creating module8-network..."
        docker network create module8-network --subnet 172.28.0.0/16
        print_success "Network created"
    else
        print_success "Network already exists"
    fi

    # Connect external containers to module8-network if not already connected
    for container_id in a2f55d83b1fa 8502fd5d078d e8b3df4d8a02; do
        if docker ps --format '{{.ID}}' | grep -q "^$container_id"; then
            if ! docker network inspect module8-network 2>/dev/null | grep -q "$container_id"; then
                print_info "Connecting container $container_id to module8-network..."
                docker network connect module8-network "$container_id" 2>/dev/null || true
            fi
        fi
    done
}

# ========================================
# Infrastructure Services
# ========================================

start_infrastructure() {
    print_header "Starting Infrastructure Services"

    print_info "Starting MongoDB, Elasticsearch, ClickHouse, Redis..."
    docker-compose -f "$COMPOSE_FILE" up -d mongodb elasticsearch clickhouse redis

    print_info "Waiting for infrastructure services to be healthy (60s)..."
    sleep 60

    # Check infrastructure health
    print_info "Checking infrastructure health..."

    if docker-compose -f "$COMPOSE_FILE" ps mongodb | grep -q "healthy"; then
        print_success "MongoDB is healthy"
    else
        print_warning "MongoDB may not be ready"
    fi

    if docker-compose -f "$COMPOSE_FILE" ps elasticsearch | grep -q "healthy"; then
        print_success "Elasticsearch is healthy"
    else
        print_warning "Elasticsearch may not be ready"
    fi

    if docker-compose -f "$COMPOSE_FILE" ps clickhouse | grep -q "healthy"; then
        print_success "ClickHouse is healthy"
    else
        print_warning "ClickHouse may not be ready"
    fi

    if docker-compose -f "$COMPOSE_FILE" ps redis | grep -q "healthy"; then
        print_success "Redis is healthy"
    else
        print_warning "Redis may not be ready"
    fi
}

# ========================================
# Projector Services
# ========================================

start_projectors() {
    print_header "Starting All 8 Storage Projectors"

    print_info "Starting projector services..."
    docker-compose -f "$COMPOSE_FILE" up -d \
        postgresql-projector \
        mongodb-projector \
        elasticsearch-projector \
        clickhouse-projector \
        influxdb-projector \
        ups-projector \
        fhir-store-projector \
        neo4j-graph-projector

    print_info "Waiting for projectors to start (30s)..."
    sleep 30
}

# ========================================
# Health Checks
# ========================================

check_service_health() {
    print_header "Running Health Checks"

    local all_healthy=true

    for service in "${!SERVICE_PORTS[@]}"; do
        local port=${SERVICE_PORTS[$service]}

        if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then
            print_success "$service (port $port) is healthy"
        else
            print_error "$service (port $port) is NOT healthy"
            all_healthy=false
        fi
    done

    echo ""
    if [ "$all_healthy" = true ]; then
        print_success "All services are healthy!"
    else
        print_warning "Some services are not healthy. Check logs for details."
        print_info "Run: ./logs-module8.sh [service-name]"
    fi
}

# ========================================
# Display Service URLs
# ========================================

show_service_urls() {
    print_header "Service URLs"

    echo "📊 Storage Projector Services:"
    echo "  PostgreSQL Projector:    http://localhost:8050"
    echo "  MongoDB Projector:       http://localhost:8051"
    echo "  Elasticsearch Projector: http://localhost:8052"
    echo "  ClickHouse Projector:    http://localhost:8053"
    echo "  InfluxDB Projector:      http://localhost:8054"
    echo "  UPS Projector:           http://localhost:8055"
    echo "  FHIR Store Projector:    http://localhost:8056"
    echo "  Neo4j Graph Projector:   http://localhost:8057"
    echo ""
    echo "🗄️  Infrastructure Services:"
    echo "  MongoDB:                 http://localhost:27017"
    echo "  Elasticsearch:           http://localhost:9200"
    echo "  ClickHouse:              http://localhost:8123"
    echo "  Redis:                   http://localhost:6379"
    echo ""
    echo "📋 Management Commands:"
    echo "  Health Check:            ./health-check-module8.sh"
    echo "  View Logs:               ./logs-module8.sh [service-name]"
    echo "  Stop All:                ./stop-module8-projectors.sh"
    echo ""
}

# ========================================
# Show Next Steps
# ========================================

show_next_steps() {
    print_header "Next Steps"

    echo "1. Check service health:"
    echo "   ./health-check-module8.sh"
    echo ""
    echo "2. Monitor logs:"
    echo "   ./logs-module8.sh -f"
    echo ""
    echo "3. Test projectors:"
    echo "   python test-module8-projectors.py"
    echo ""
    echo "4. View metrics:"
    echo "   # Open http://localhost:9090 (Prometheus)"
    echo "   # Open http://localhost:3000 (Grafana)"
    echo ""
}

# ========================================
# Main Execution
# ========================================

main() {
    print_header "🚀 Module 8 Storage Projectors Startup"

    # Run checks
    check_prerequisites
    check_external_containers

    # Setup network
    setup_network

    # Start services
    start_infrastructure
    start_projectors

    # Verify health
    check_service_health

    # Show information
    show_service_urls
    show_next_steps

    print_header "✅ Startup Complete"
}

# Run main function
main "$@"
