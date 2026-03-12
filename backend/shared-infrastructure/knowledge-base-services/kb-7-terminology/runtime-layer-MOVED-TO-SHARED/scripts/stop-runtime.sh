#!/bin/bash
# Stop script for KB7 Neo4j Dual-Stream & Service Runtime Layer
# This script gracefully stops all runtime services

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

# Stop Python runtime services gracefully
stop_python_services() {
    log_info "Stopping Python runtime services..."

    cd "$RUNTIME_DIR"

    # Check if virtual environment exists
    if [ -d "venv" ]; then
        source venv/bin/activate

        # Gracefully stop the main integration service if running
        if python main_integration.py --stop &> /dev/null; then
            log_success "Python runtime services stopped gracefully"
        else
            log_warning "Python runtime services were not running or stopped with warnings"
        fi
    else
        log_warning "Virtual environment not found, skipping Python service shutdown"
    fi
}

# Stop Docker services
stop_docker_services() {
    log_info "Stopping Docker services..."

    cd "$RUNTIME_DIR"

    # Stop all services defined in docker-compose
    if docker-compose -f docker-compose.runtime.yml down; then
        log_success "Docker services stopped"
    else
        log_error "Error stopping Docker services"
        return 1
    fi
}

# Force stop if needed
force_stop() {
    log_warning "Force stopping all KB7 runtime containers..."

    # Get all KB7 related containers
    local containers=$(docker ps -a --filter "name=kb7-*" --format "{{.Names}}")

    if [ -n "$containers" ]; then
        echo "$containers" | while read container; do
            log_info "Force stopping $container..."
            docker stop "$container" 2>/dev/null || true
            docker rm "$container" 2>/dev/null || true
        done
        log_success "Force stop completed"
    else
        log_info "No KB7 containers found to stop"
    fi
}

# Clean up volumes if requested
cleanup_volumes() {
    log_warning "Cleaning up Docker volumes..."

    cd "$RUNTIME_DIR"

    # Stop and remove containers and volumes
    docker-compose -f docker-compose.runtime.yml down -v

    log_success "Volumes cleaned up"
}

# Clean up networks
cleanup_networks() {
    log_info "Cleaning up Docker networks..."

    # Remove KB7 runtime network if it exists
    if docker network ls --filter "name=kb7-runtime-network" --format "{{.Name}}" | grep -q "kb7-runtime-network"; then
        docker network rm kb7-runtime-network 2>/dev/null || true
        log_success "KB7 runtime network removed"
    else
        log_info "KB7 runtime network not found"
    fi
}

# Show status after stopping
show_status() {
    log_info "Checking remaining KB7 services..."

    local running_containers=$(docker ps --filter "name=kb7-*" --format "{{.Names}}")

    if [ -n "$running_containers" ]; then
        log_warning "Some KB7 containers are still running:"
        echo "$running_containers"
    else
        log_success "All KB7 containers have been stopped"
    fi

    # Check for any listening ports from our services
    local kb7_ports="7474 7687 7200 8123 9000 6379 6380 9092 2181 8080 8081 8082 8083 3000 9090"
    local active_ports=""

    for port in $kb7_ports; do
        if lsof -i ":$port" &>/dev/null; then
            active_ports="$active_ports $port"
        fi
    done

    if [ -n "$active_ports" ]; then
        log_warning "Some KB7 ports are still in use:$active_ports"
    else
        log_success "All KB7 ports are available"
    fi
}

# Main stop function
graceful_stop() {
    log_info "Initiating graceful shutdown of KB7 Runtime Layer..."

    stop_python_services
    stop_docker_services
    cleanup_networks
    show_status

    log_success "KB7 Runtime Layer shutdown complete"
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--help|--force|--clean-volumes|--status]"
        echo ""
        echo "This script stops the KB7 Neo4j Dual-Stream & Service Runtime Layer"
        echo ""
        echo "Options:"
        echo "  --help, -h        Show this help message"
        echo "  --force           Force stop all KB7 containers"
        echo "  --clean-volumes   Stop services and remove all volumes (data loss!)"
        echo "  --status          Show current status of KB7 services"
        echo ""
        echo "Default behavior is graceful shutdown of all services."
        exit 0
        ;;
    --force)
        log_warning "Force stopping all KB7 services..."
        force_stop
        cleanup_networks
        show_status
        ;;
    --clean-volumes)
        log_warning "⚠️  WARNING: This will remove all data volumes!"
        read -p "Are you sure you want to continue? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            stop_python_services
            cleanup_volumes
            cleanup_networks
            show_status
            log_warning "All volumes have been removed. Data has been lost!"
        else
            log_info "Operation cancelled"
        fi
        ;;
    --status)
        show_status
        ;;
    *)
        graceful_stop
        ;;
esac