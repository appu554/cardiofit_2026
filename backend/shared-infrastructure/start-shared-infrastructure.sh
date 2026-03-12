#!/bin/bash

# ============================================
# CardioFit Shared Infrastructure Startup Script
# Starts both Flink Processing and Runtime Layer
# ============================================

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

# Function to print colored output
print_status() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

print_success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
}

print_error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $1${NC}"
}

# Function to check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."

    if ! command -v docker >/dev/null 2>&1; then
        print_error "Docker is not installed"
        exit 1
    fi

    if ! command -v docker-compose >/dev/null 2>&1; then
        print_error "Docker Compose is not installed"
        exit 1
    fi

    print_success "Prerequisites check passed"
}

# Function to start infrastructure
start_infrastructure() {
    print_status "Starting CardioFit Shared Infrastructure..."
    echo ""

    # Phase 1: Start core dependencies first (Kafka, Zookeeper)
    print_status "Phase 1: Starting message infrastructure..."
    cd runtime-layer
    docker-compose -f docker-compose.yml up -d zookeeper kafka
    cd ..
    sleep 20

    # Phase 2: Start storage infrastructure
    print_status "Phase 2: Starting storage infrastructure..."
    cd runtime-layer
    docker-compose -f docker-compose.yml up -d neo4j graphdb redis clickhouse mongodb-evidence mongodb-sla
    cd ..
    sleep 30

    # Phase 3: Start Flink processing (parallel to runtime)
    print_status "Phase 3: Starting Flink global stream processing..."
    cd runtime-layer
    docker-compose -f docker-compose.yml up -d flink-jobmanager flink-taskmanager
    cd ..
    sleep 20

    # Phase 4: Start runtime layer services
    print_status "Phase 4: Starting runtime layer services..."
    cd runtime-layer
    docker-compose -f docker-compose.yml up -d query-router cache-prefetcher evidence-envelope sla-monitoring
    cd ..
    sleep 15

    # Phase 5: Start monitoring
    print_status "Phase 5: Starting monitoring infrastructure..."
    cd runtime-layer
    docker-compose -f docker-compose.yml up -d prometheus grafana
    cd ..

    print_success "All infrastructure components started"
}

# Function to show status
show_status() {
    echo ""
    print_status "Shared Infrastructure Status:"
    echo ""

    echo -e "${BLUE}=== FLINK PROCESSING (Global Stream Processing) ===${NC}"
    echo -e "Flink JobManager:     http://localhost:8081"
    echo -e "Flink Metrics:        http://localhost:9249/metrics"
    echo -e "Purpose:              Real-time clinical pattern detection"
    echo ""

    echo -e "${BLUE}=== RUNTIME LAYER (Query & Storage Infrastructure) ===${NC}"
    echo -e "Neo4j Browser:        http://localhost:7474"
    echo -e "GraphDB:              http://localhost:7200"
    echo -e "Query Router:         http://localhost:8070"
    echo -e "Cache Prefetcher:     http://localhost:8055"
    echo -e "Evidence Envelope:    http://localhost:8060"
    echo ""

    echo -e "${BLUE}=== MONITORING ===${NC}"
    echo -e "Prometheus:           http://localhost:9090"
    echo -e "Grafana:              http://localhost:3000 (admin/admin)"
    echo ""

    print_status "Architecture Overview:"
    cat << 'EOF'

    Microservices
         ↓
    Kafka Topics ←────────────┐
         ↓                    │
    ┌────┴──────────────┐     │
    │  FLINK PROCESSING │     │ (Events)
    │  (Stream Intel)   ├─────┘
    └────┬──────────────┘
         ↓
    ┌────────────────────┐
    │   RUNTIME LAYER    │
    │  (Storage/Query)   │
    └────────────────────┘
         ↓
    Clinical Services

EOF
}

# Main execution
main() {
    print_status "CardioFit Shared Infrastructure Orchestrator"
    echo "============================================"
    echo "This will start:"
    echo "  • Flink Processing (Global stream processing)"
    echo "  • Runtime Layer (Storage and query infrastructure)"
    echo "============================================"
    echo ""

    check_prerequisites
    start_infrastructure
    show_status

    print_success "Shared Infrastructure startup completed!"
    echo ""
    print_status "Quick health check:"
    echo "  ./runtime-layer/health-check.sh"
    echo ""
    print_status "Stop all services:"
    echo "  ./stop-shared-infrastructure.sh"
    echo ""
}

main "$@"