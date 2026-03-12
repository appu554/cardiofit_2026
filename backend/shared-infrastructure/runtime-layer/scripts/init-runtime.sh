#!/bin/bash
# Initialization script for KB7 Neo4j Dual-Stream & Service Runtime Layer
# This script sets up the complete runtime environment

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
DOCKER_COMPOSE_FILE="$RUNTIME_DIR/docker-compose.runtime.yml"

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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi

    # Check Python
    if ! command -v python3 &> /dev/null; then
        log_error "Python 3 is not installed. Please install Python 3.11+ first."
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Set environment variables
setup_environment() {
    log_info "Setting up environment variables..."

    # Set default passwords if not already set
    export NEO4J_PASSWORD=${NEO4J_PASSWORD:-kb7password}
    export CH_PASSWORD=${CH_PASSWORD:-kb7password}
    export GRAFANA_PASSWORD=${GRAFANA_PASSWORD:-admin}

    # Create .env file for Docker Compose
    cat > "$RUNTIME_DIR/.env" <<EOF
NEO4J_PASSWORD=$NEO4J_PASSWORD
CH_PASSWORD=$CH_PASSWORD
GRAFANA_PASSWORD=$GRAFANA_PASSWORD
EOF

    log_success "Environment variables configured"
}

# Start infrastructure services
start_infrastructure() {
    log_info "Starting infrastructure services..."

    cd "$RUNTIME_DIR"

    # Start only infrastructure services first
    docker-compose -f docker-compose.runtime.yml up -d \
        neo4j \
        graphdb \
        clickhouse \
        redis-l2 \
        redis-l3 \
        kafka \
        zookeeper \
        prometheus \
        grafana

    log_success "Infrastructure services started"
}

# Wait for services to be ready
wait_for_services() {
    log_info "Waiting for services to be ready..."

    # Wait for Neo4j
    log_info "Waiting for Neo4j..."
    timeout 120 bash -c 'until docker exec kb7-neo4j cypher-shell -u neo4j -p $NEO4J_PASSWORD "RETURN 1" &>/dev/null; do sleep 5; done'

    # Wait for ClickHouse
    log_info "Waiting for ClickHouse..."
    timeout 120 bash -c 'until docker exec kb7-clickhouse clickhouse-client --query "SELECT 1" &>/dev/null; do sleep 5; done'

    # Wait for GraphDB
    log_info "Waiting for GraphDB..."
    timeout 120 bash -c 'until curl -f http://localhost:7200/rest/info &>/dev/null; do sleep 5; done'

    # Wait for Kafka
    log_info "Waiting for Kafka..."
    timeout 120 bash -c 'until docker exec kb7-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 &>/dev/null; do sleep 5; done'

    log_success "All infrastructure services are ready"
}

# Initialize databases
initialize_databases() {
    log_info "Initializing databases..."

    # Initialize Neo4j databases
    log_info "Creating Neo4j databases..."
    docker exec kb7-neo4j cypher-shell -u neo4j -p "$NEO4J_PASSWORD" \
        "CREATE DATABASE patient_data IF NOT EXISTS; CREATE DATABASE semantic_mesh IF NOT EXISTS;"

    # Initialize ClickHouse
    log_info "Creating ClickHouse database..."
    docker exec kb7-clickhouse clickhouse-client \
        --query "CREATE DATABASE IF NOT EXISTS kb7_analytics;"

    # Create Kafka topics
    log_info "Creating Kafka topics..."
    docker exec kb7-kafka kafka-topics --create --if-not-exists \
        --topic kb-changes --partitions 3 --replication-factor 1 \
        --bootstrap-server localhost:9092

    docker exec kb7-kafka kafka-topics --create --if-not-exists \
        --topic patient-events --partitions 3 --replication-factor 1 \
        --bootstrap-server localhost:9092

    docker exec kb7-kafka kafka-topics --create --if-not-exists \
        --topic service-events --partitions 3 --replication-factor 1 \
        --bootstrap-server localhost:9092

    log_success "Databases initialized"
}

# Install Python dependencies
install_dependencies() {
    log_info "Installing Python dependencies..."

    cd "$RUNTIME_DIR"

    # Create virtual environment if it doesn't exist
    if [ ! -d "venv" ]; then
        python3 -m venv venv
    fi

    # Activate virtual environment and install dependencies
    source venv/bin/activate
    pip install --upgrade pip
    pip install -r requirements.txt

    log_success "Python dependencies installed"
}

# Initialize runtime components
initialize_runtime() {
    log_info "Initializing runtime components..."

    cd "$RUNTIME_DIR"
    source venv/bin/activate

    # Run the complete integration initialization
    python main_integration.py --initialize

    log_success "Runtime components initialized"
}

# Start runtime services
start_runtime_services() {
    log_info "Starting runtime services..."

    cd "$RUNTIME_DIR"

    # Start runtime services
    docker-compose -f docker-compose.runtime.yml up -d \
        query-router \
        adapter-microservice \
        cdc-cache-warmer \
        event-bus-orchestrator \
        medication-runtime

    log_success "Runtime services started"
}

# Verify installation
verify_installation() {
    log_info "Verifying installation..."

    # Wait a bit for services to start
    sleep 30

    # Check service health
    cd "$RUNTIME_DIR"
    source venv/bin/activate

    if python main_integration.py --health; then
        log_success "Installation verification passed"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

# Display access information
display_access_info() {
    log_success "KB7 Runtime Layer initialization complete!"
    echo ""
    echo "Access Information:"
    echo "=================="
    echo "Neo4j Browser:        http://localhost:7474 (neo4j/$NEO4J_PASSWORD)"
    echo "GraphDB Workbench:    http://localhost:7200"
    echo "ClickHouse Play:      http://localhost:8123/play"
    echo "Grafana:              http://localhost:3000 (admin/$GRAFANA_PASSWORD)"
    echo "Prometheus:           http://localhost:9090"
    echo "Query Router:         http://localhost:8080"
    echo ""
    echo "Commands:"
    echo "========="
    echo "Health Check:         python main_integration.py --health"
    echo "Run Tests:            python main_integration.py --test"
    echo "Stop Services:        docker-compose -f docker-compose.runtime.yml down"
    echo ""
}

# Main execution
main() {
    log_info "Starting KB7 Neo4j Dual-Stream & Service Runtime Layer initialization..."

    check_prerequisites
    setup_environment
    start_infrastructure
    wait_for_services
    initialize_databases
    install_dependencies
    initialize_runtime
    start_runtime_services
    verify_installation
    display_access_info
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--help]"
        echo ""
        echo "This script initializes the KB7 Neo4j Dual-Stream & Service Runtime Layer"
        echo "including all infrastructure services, databases, and runtime components."
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  NEO4J_PASSWORD     Password for Neo4j (default: kb7password)"
        echo "  CH_PASSWORD        Password for ClickHouse (default: kb7password)"
        echo "  GRAFANA_PASSWORD   Password for Grafana (default: admin)"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac