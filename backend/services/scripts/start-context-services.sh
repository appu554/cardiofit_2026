#!/bin/bash
# Context Services Startup Script
# Orchestrates the complete Context Services stack with proper dependency management

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.context-services.yml"
SERVICES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_TIMEOUT=60
HEALTH_CHECK_TIMEOUT=300

# Function to print colored output
print_status() {
    echo -e "${GREEN}[$(date '+%H:%M:%S')]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[$(date '+%H:%M:%S')] WARNING:${NC} $1"
}

print_error() {
    echo -e "${RED}[$(date '+%H:%M:%S')] ERROR:${NC} $1"
}

print_info() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')] INFO:${NC} $1"
}

# Function to check if Docker is running
check_docker() {
    print_info "Checking Docker availability..."
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi
    print_status "Docker is running"
}

# Function to check if docker-compose is available
check_docker_compose() {
    if command -v docker-compose >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        DOCKER_COMPOSE="docker compose"
    else
        print_error "docker-compose or 'docker compose' not found. Please install Docker Compose."
        exit 1
    fi
    print_status "Using: $DOCKER_COMPOSE"
}

# Function to generate SSL certificates
generate_ssl_certs() {
    print_info "Checking SSL certificates..."
    
    if [[ ! -f "$SERVICES_DIR/nginx/ssl/context-services.crt" ]]; then
        print_info "Generating SSL certificates..."
        mkdir -p "$SERVICES_DIR/nginx/ssl"
        
        # Generate self-signed certificate
        openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
            -keyout "$SERVICES_DIR/nginx/ssl/context-services.key" \
            -out "$SERVICES_DIR/nginx/ssl/context-services.crt" \
            -subj "/C=US/ST=CA/L=San Francisco/O=Clinical Synthesis Hub/CN=localhost" \
            >/dev/null 2>&1
        
        print_status "SSL certificates generated"
    else
        print_status "SSL certificates already exist"
    fi
}

# Function to create necessary directories
create_directories() {
    print_info "Creating necessary directories..."
    
    directories=(
        "$SERVICES_DIR/logs"
        "$SERVICES_DIR/data/redis"
        "$SERVICES_DIR/data/mongo"
        "$SERVICES_DIR/data/postgres"
        "$SERVICES_DIR/data/kafka"
        "$SERVICES_DIR/monitoring/data"
    )
    
    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
    done
    
    print_status "Directories created"
}

# Function to start infrastructure services first
start_infrastructure() {
    print_info "Starting infrastructure services..."
    
    cd "$SERVICES_DIR"
    
    # Start infrastructure in dependency order
    infrastructure_services=(
        "zookeeper"
        "kafka"
        "redis-cluster"
        "mongo-context"
        "postgres-hub"
        "consul"
    )
    
    for service in "${infrastructure_services[@]}"; do
        print_info "Starting $service..."
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d "$service"
        
        # Brief pause to allow service to start
        sleep 2
    done
    
    print_status "Infrastructure services started"
}

# Function to wait for infrastructure to be ready
wait_for_infrastructure() {
    print_info "Waiting for infrastructure services to be ready..."
    
    # Wait for services with custom health checks
    services_to_check=(
        "redis-cluster:6379"
        "mongo-context:27017"
        "postgres-hub:5432"
        "kafka-context:9092"
        "consul-context:8500"
    )
    
    for service_port in "${services_to_check[@]}"; do
        service_name=${service_port%:*}
        port=${service_port#*:}
        
        print_info "Waiting for $service_name on port $port..."
        
        timeout=60
        while ! nc -z localhost "$port" 2>/dev/null && [ $timeout -gt 0 ]; do
            sleep 1
            timeout=$((timeout - 1))
        done
        
        if [ $timeout -eq 0 ]; then
            print_warning "$service_name not responding on port $port, continuing anyway..."
        else
            print_status "$service_name is ready"
        fi
    done
}

# Function to start monitoring services
start_monitoring() {
    print_info "Starting monitoring services..."
    
    monitoring_services=(
        "prometheus"
        "grafana"
    )
    
    for service in "${monitoring_services[@]}"; do
        print_info "Starting $service..."
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d "$service"
    done
    
    print_status "Monitoring services started"
}

# Function to start context services
start_context_services() {
    print_info "Starting Context Services..."
    
    # Start services in dependency order
    context_services=(
        "clinical-data-hub-rust"  # Start Rust service first (dependency for Go service)
        "context-gateway-go"      # Start Go service
        "nginx"                   # Start load balancer last
    )
    
    for service in "${context_services[@]}"; do
        print_info "Starting $service..."
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d "$service"
        
        # Wait a bit longer for context services to start
        sleep 5
    done
    
    print_status "Context Services started"
}

# Function to check service health
check_service_health() {
    print_info "Checking service health..."
    
    health_endpoints=(
        "http://localhost:8117/health:Context Gateway"
        "http://localhost:8118/health:Clinical Data Hub"
        "http://localhost:8080/health:Load Balancer"
        "http://localhost:8090/health/detailed:Detailed Health"
    )
    
    all_healthy=true
    
    for endpoint_info in "${health_endpoints[@]}"; do
        endpoint=${endpoint_info%:*}
        service_name=${endpoint_info#*:}
        
        print_info "Checking $service_name..."
        
        # Try up to 10 times with 3-second intervals
        for i in {1..10}; do
            if curl -s -f "$endpoint" >/dev/null 2>&1; then
                print_status "$service_name is healthy"
                break
            fi
            
            if [ $i -eq 10 ]; then
                print_warning "$service_name health check failed"
                all_healthy=false
            else
                sleep 3
            fi
        done
    done
    
    if [ "$all_healthy" = true ]; then
        print_status "All services are healthy!"
    else
        print_warning "Some services may not be fully ready"
    fi
}

# Function to show service status
show_service_status() {
    print_info "Service Status:"
    echo
    
    cd "$SERVICES_DIR"
    $DOCKER_COMPOSE -f "$COMPOSE_FILE" ps
    
    echo
    print_info "Access URLs:"
    echo "  🌐 Load Balancer (HTTP):  http://localhost:8080"
    echo "  🔒 Load Balancer (HTTPS): https://localhost:8443"
    echo "  📊 Context Gateway:       http://localhost:8117"
    echo "  ⚡ Clinical Data Hub:     http://localhost:8118"
    echo "  📈 Prometheus:            http://localhost:9090"
    echo "  📊 Grafana:               http://localhost:3001"
    echo "  🏗️  Consul:                http://localhost:8500"
    echo
    
    print_info "gRPC Services:"
    echo "  🔌 Context Gateway gRPC:  localhost:8017"
    echo "  ⚡ Clinical Hub gRPC:     localhost:8018"
    echo "  🌐 Load Balancer gRPC:    localhost:50051 (Context), localhost:50052 (Clinical)"
    echo
}

# Function to run integration tests
run_integration_tests() {
    if [ "$RUN_TESTS" = "true" ]; then
        print_info "Running integration tests..."
        
        if [ -f "$SERVICES_DIR/integration-tests/context-services-integration-test.py" ]; then
            cd "$SERVICES_DIR/integration-tests"
            python3 context-services-integration-test.py
        else
            print_warning "Integration test file not found, skipping tests"
        fi
    fi
}

# Function to handle cleanup on exit
cleanup() {
    if [ "$CLEANUP_ON_EXIT" = "true" ]; then
        print_info "Cleaning up..."
        cd "$SERVICES_DIR"
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" down
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  -t, --test              Run integration tests after startup"
    echo "  -c, --cleanup           Clean up on exit (Ctrl+C)"
    echo "  -q, --quick             Skip health checks for faster startup"
    echo "  -v, --verbose           Verbose output"
    echo "  --build                 Rebuild images before starting"
    echo "  --pull                  Pull latest images before starting"
    echo
    echo "Examples:"
    echo "  $0                      Start all services"
    echo "  $0 --test               Start services and run tests"
    echo "  $0 --build --test       Rebuild, start, and test"
    echo "  $0 --cleanup            Start with cleanup on exit"
    echo
}

# Parse command line arguments
SKIP_HEALTH_CHECKS=false
RUN_TESTS=false
CLEANUP_ON_EXIT=false
VERBOSE=false
BUILD_IMAGES=false
PULL_IMAGES=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -t|--test)
            RUN_TESTS=true
            shift
            ;;
        -c|--cleanup)
            CLEANUP_ON_EXIT=true
            shift
            ;;
        -q|--quick)
            SKIP_HEALTH_CHECKS=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            set -x
            shift
            ;;
        --build)
            BUILD_IMAGES=true
            shift
            ;;
        --pull)
            PULL_IMAGES=true
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Set up cleanup handler
if [ "$CLEANUP_ON_EXIT" = "true" ]; then
    trap cleanup EXIT INT TERM
fi

# Main execution
main() {
    print_status "Starting Context Services Stack"
    echo "======================================"
    
    # Pre-flight checks
    check_docker
    check_docker_compose
    
    # Setup
    create_directories
    generate_ssl_certs
    
    # Change to services directory
    cd "$SERVICES_DIR"
    
    # Pull or build images if requested
    if [ "$PULL_IMAGES" = "true" ]; then
        print_info "Pulling latest images..."
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" pull
    fi
    
    if [ "$BUILD_IMAGES" = "true" ]; then
        print_info "Building images..."
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" build
    fi
    
    # Start services in order
    start_infrastructure
    
    if [ "$SKIP_HEALTH_CHECKS" = "false" ]; then
        wait_for_infrastructure
    fi
    
    start_monitoring
    start_context_services
    
    if [ "$SKIP_HEALTH_CHECKS" = "false" ]; then
        sleep 10  # Give services time to fully start
        check_service_health
    fi
    
    # Show status
    show_service_status
    
    # Run tests if requested
    run_integration_tests
    
    print_status "Context Services stack is ready!"
    
    if [ "$CLEANUP_ON_EXIT" = "true" ]; then
        print_info "Press Ctrl+C to stop all services..."
        # Wait for interrupt
        while true; do
            sleep 1
        done
    else
        print_info "Services are running in the background"
        print_info "Use 'docker-compose -f $COMPOSE_FILE down' to stop"
    fi
}

# Run main function
main