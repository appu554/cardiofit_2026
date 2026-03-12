#!/bin/bash

# ============================================
# CardioFit Runtime Layer Startup Script
# ============================================
# This script starts the complete runtime layer infrastructure

set -e  # Exit on any error

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
COMPOSE_FILE="docker-compose.yml"
ENV_FILE=".env"
DEFAULT_TIMEOUT=300  # 5 minutes
HEALTH_CHECK_INTERVAL=10

# Function to print colored output
print_status() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

print_success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $1${NC}"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."

    if ! command_exists docker; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    if ! command_exists docker-compose; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi

    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running. Please start Docker first."
        exit 1
    fi

    print_success "Prerequisites check passed"
}

# Function to check system resources
check_resources() {
    print_status "Checking system resources..."

    # Check available memory (Linux/macOS)
    if command_exists free; then
        AVAILABLE_MEM=$(free -g | awk '/^Mem:/{print $7}')
    elif command_exists vm_stat; then
        # macOS
        AVAILABLE_MEM=$(vm_stat | perl -ne '/free:\s+(\d+)/ and $free=$1; /inactive:\s+(\d+)/ and $inactive=$1; END {print int(($free+$inactive)*4096/1024/1024/1024)}')
    else
        AVAILABLE_MEM=16  # Assume sufficient memory if can't detect
    fi

    if [ "$AVAILABLE_MEM" -lt 16 ]; then
        print_warning "Available memory is ${AVAILABLE_MEM}GB. Runtime layer recommends 16GB+ for optimal performance."
    else
        print_success "Memory check passed: ${AVAILABLE_MEM}GB available"
    fi

    # Check available disk space
    AVAILABLE_DISK=$(df -BG . | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$AVAILABLE_DISK" -lt 100 ]; then
        print_warning "Available disk space is ${AVAILABLE_DISK}GB. Runtime layer recommends 100GB+ for optimal performance."
    else
        print_success "Disk space check passed: ${AVAILABLE_DISK}GB available"
    fi
}

# Function to setup environment
setup_environment() {
    print_status "Setting up environment..."

    if [ ! -f "$ENV_FILE" ]; then
        print_status "Creating .env file from template..."
        cp .env.example .env
        print_success "Created .env file. Please review and customize if needed."
    else
        print_success "Environment file exists"
    fi

    # Create necessary directories
    mkdir -p monitoring/grafana/dashboards monitoring/grafana/datasources
    mkdir -p logs backups

    print_success "Environment setup completed"
}

# Function to pull latest images
pull_images() {
    print_status "Pulling latest Docker images..."
    if docker-compose -f "$COMPOSE_FILE" pull; then
        print_success "Images pulled successfully"
    else
        print_warning "Some images failed to pull, continuing with existing images..."
    fi
}

# Function to start services in phases
start_services() {
    print_status "Starting Runtime Layer services..."

    # Phase 1: Core Infrastructure
    print_status "Phase 1: Starting core infrastructure (Neo4j, GraphDB, Kafka, Redis, ClickHouse)..."
    docker-compose -f "$COMPOSE_FILE" up -d \
        zookeeper kafka neo4j graphdb redis clickhouse

    print_status "Waiting for core services to be ready..."
    sleep 30

    # Phase 2: Application Services
    print_status "Phase 2: Starting application services (Query Router, Cache Prefetcher, Evidence Envelope)..."
    docker-compose -f "$COMPOSE_FILE" up -d \
        query-router cache-prefetcher evidence-envelope mongodb-evidence

    sleep 20

    # Phase 3: Stream Processing
    print_status "Phase 3: Starting stream processing (Flink)..."
    docker-compose -f "$COMPOSE_FILE" up -d \
        flink-jobmanager flink-taskmanager

    sleep 15

    # Phase 4: Monitoring
    print_status "Phase 4: Starting monitoring services (SLA Monitor, Prometheus, Grafana)..."
    docker-compose -f "$COMPOSE_FILE" up -d \
        sla-monitoring mongodb-sla prometheus grafana

    print_success "All services started"
}

# Function to wait for services to be healthy
wait_for_services() {
    print_status "Waiting for services to be healthy (timeout: ${DEFAULT_TIMEOUT}s)..."

    local start_time=$(date +%s)
    local services=(
        "neo4j:7474"
        "graphdb:7200"
        "kafka:29092"
        "redis:6379"
        "clickhouse:8123"
        "flink-jobmanager:8081"
        "query-router:8070"
        "cache-prefetcher:8055"
        "evidence-envelope:8060"
        "sla-monitoring:8050"
        "prometheus:9090"
        "grafana:3000"
    )

    for service in "${services[@]}"; do
        local service_name=$(echo "$service" | cut -d: -f1)
        local port=$(echo "$service" | cut -d: -f2)

        print_status "Checking $service_name (port $port)..."

        local timeout_reached=false
        while ! nc -z localhost "$port" >/dev/null 2>&1; do
            local current_time=$(date +%s)
            local elapsed=$((current_time - start_time))

            if [ $elapsed -gt $DEFAULT_TIMEOUT ]; then
                timeout_reached=true
                break
            fi

            sleep $HEALTH_CHECK_INTERVAL
        done

        if [ "$timeout_reached" = true ]; then
            print_warning "$service_name not ready after ${DEFAULT_TIMEOUT}s (may still be starting)"
        else
            print_success "$service_name is ready"
        fi
    done
}

# Function to show service status
show_status() {
    print_status "Runtime Layer Service Status:"
    echo ""
    docker-compose -f "$COMPOSE_FILE" ps
    echo ""

    print_status "Service Endpoints:"
    echo -e "${BLUE}Neo4j Browser:        ${NC}http://localhost:7474"
    echo -e "${BLUE}GraphDB Workbench:    ${NC}http://localhost:7200"
    echo -e "${BLUE}Flink Dashboard:      ${NC}http://localhost:8081"
    echo -e "${BLUE}Query Router:         ${NC}http://localhost:8070"
    echo -e "${BLUE}Cache Prefetcher:     ${NC}http://localhost:8055"
    echo -e "${BLUE}Evidence Envelope:    ${NC}http://localhost:8060"
    echo -e "${BLUE}SLA Monitoring:       ${NC}http://localhost:8050"
    echo -e "${BLUE}Prometheus:           ${NC}http://localhost:9090"
    echo -e "${BLUE}Grafana:              ${NC}http://localhost:3000 (admin/admin)"
    echo ""
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --quick           Skip resource checks and image pulling"
    echo "  --no-monitoring   Skip monitoring services (Prometheus, Grafana)"
    echo "  --timeout N       Set health check timeout in seconds (default: $DEFAULT_TIMEOUT)"
    echo "  --help           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                Start full runtime layer"
    echo "  $0 --quick        Quick start without checks"
    echo "  $0 --no-monitoring  Start without monitoring services"
    echo ""
}

# Function to cleanup on exit
cleanup() {
    if [ $? -ne 0 ]; then
        print_error "Startup failed. Check logs with: docker-compose logs [service-name]"
        echo ""
        print_status "To cleanup failed containers:"
        echo "  docker-compose down"
        echo ""
        print_status "To start individual services for debugging:"
        echo "  docker-compose up [service-name]"
    fi
}

# Main execution
main() {
    # Parse command line arguments
    local QUICK_START=false
    local NO_MONITORING=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --quick)
                QUICK_START=true
                shift
                ;;
            --no-monitoring)
                NO_MONITORING=true
                shift
                ;;
            --timeout)
                DEFAULT_TIMEOUT="$2"
                shift 2
                ;;
            --help)
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

    # Set up cleanup handler
    trap cleanup EXIT

    print_status "Starting CardioFit Runtime Layer..."
    echo ""

    # Run startup sequence
    check_prerequisites

    if [ "$QUICK_START" != true ]; then
        check_resources
        pull_images
    fi

    setup_environment
    start_services

    if [ "$NO_MONITORING" != true ]; then
        wait_for_services
    else
        print_status "Skipping monitoring services..."
        # Stop monitoring services if they were started
        docker-compose -f "$COMPOSE_FILE" stop prometheus grafana sla-monitoring mongodb-sla 2>/dev/null || true
    fi

    show_status

    print_success "Runtime Layer startup completed!"
    echo ""
    print_status "Next steps:"
    echo "1. Review service endpoints above"
    echo "2. Run health checks: ./health-check.sh"
    echo "3. View logs: docker-compose logs [service-name]"
    echo "4. Stop services: docker-compose down"
    echo ""
}

# Run main function with all arguments
main "$@"