#!/bin/bash
set -e

# ========================================
# Module 8 Storage Projectors Shutdown Script
# ========================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.module8-complete.yml"

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
# Collect Statistics Before Shutdown
# ========================================

collect_statistics() {
    print_header "Collecting Service Statistics"

    local stats_file="${SCRIPT_DIR}/module8-shutdown-stats-$(date +%Y%m%d-%H%M%S).log"

    echo "Module 8 Shutdown Statistics - $(date)" > "$stats_file"
    echo "======================================" >> "$stats_file"
    echo "" >> "$stats_file"

    # Collect container stats
    echo "Container Resource Usage:" >> "$stats_file"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" \
        $(docker-compose -f "$COMPOSE_FILE" ps -q 2>/dev/null) >> "$stats_file" 2>/dev/null || true
    echo "" >> "$stats_file"

    # Collect health status
    echo "Service Health Status:" >> "$stats_file"
    for port in 8050 8051 8052 8053 8054 8055 8056 8057; do
        if curl -sf "http://localhost:$port/health" > /dev/null 2>&1; then
            echo "Port $port: HEALTHY" >> "$stats_file"
        else
            echo "Port $port: UNHEALTHY" >> "$stats_file"
        fi
    done
    echo "" >> "$stats_file"

    # Collect log summary
    echo "Recent Log Errors:" >> "$stats_file"
    docker-compose -f "$COMPOSE_FILE" logs --tail=50 2>&1 | grep -i "error\|exception\|failed" >> "$stats_file" || echo "No errors found" >> "$stats_file"

    print_success "Statistics saved to: $stats_file"
}

# ========================================
# Stop Services
# ========================================

stop_projectors() {
    print_header "Stopping Storage Projectors"

    print_info "Stopping all 8 projector services..."
    docker-compose -f "$COMPOSE_FILE" stop \
        postgresql-projector \
        mongodb-projector \
        elasticsearch-projector \
        clickhouse-projector \
        influxdb-projector \
        ups-projector \
        fhir-store-projector \
        neo4j-graph-projector

    print_success "Projectors stopped"
}

stop_infrastructure() {
    print_header "Stopping Infrastructure Services"

    # Ask user if they want to stop infrastructure
    echo -n "Stop infrastructure services (MongoDB, Elasticsearch, ClickHouse, Redis)? [y/N]: "
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        print_info "Stopping infrastructure services..."
        docker-compose -f "$COMPOSE_FILE" stop mongodb elasticsearch clickhouse redis
        print_success "Infrastructure stopped"
    else
        print_info "Keeping infrastructure services running"
    fi
}

remove_containers() {
    print_header "Removing Containers"

    # Ask user if they want to remove containers
    echo -n "Remove stopped containers? [y/N]: "
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        print_info "Removing containers..."
        docker-compose -f "$COMPOSE_FILE" rm -f
        print_success "Containers removed"
    else
        print_info "Keeping containers"
    fi
}

cleanup_volumes() {
    print_header "Volume Cleanup"

    # Ask user if they want to remove volumes (data deletion)
    echo -n "⚠️  DANGER: Remove volumes (deletes all data)? [y/N]: "
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        print_warning "This will delete all data!"
        echo -n "Are you absolutely sure? Type 'DELETE' to confirm: "
        read -r confirm

        if [[ "$confirm" == "DELETE" ]]; then
            print_info "Removing volumes..."
            docker-compose -f "$COMPOSE_FILE" down -v
            print_success "Volumes removed"
        else
            print_info "Volume removal cancelled"
        fi
    else
        print_info "Keeping volumes"
    fi
}

disconnect_external_containers() {
    print_header "Disconnecting External Containers"

    # Ask user if they want to disconnect external containers
    echo -n "Disconnect external containers from module8-network? [y/N]: "
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        for container_id in a2f55d83b1fa 8502fd5d078d e8b3df4d8a02; do
            if docker ps --format '{{.ID}}' | grep -q "^$container_id"; then
                if docker network inspect module8-network 2>/dev/null | grep -q "$container_id"; then
                    print_info "Disconnecting container $container_id..."
                    docker network disconnect module8-network "$container_id" 2>/dev/null || true
                fi
            fi
        done
        print_success "External containers disconnected"
    else
        print_info "Keeping external containers connected"
    fi
}

# ========================================
# Show Final Status
# ========================================

show_final_status() {
    print_header "Final Status"

    echo "📊 Container Status:"
    docker-compose -f "$COMPOSE_FILE" ps || echo "No containers running"
    echo ""

    echo "🗄️  Volume Status:"
    docker volume ls | grep module8 || echo "No Module 8 volumes found"
    echo ""

    echo "🌐 Network Status:"
    docker network ls | grep module8 || echo "No Module 8 networks found"
    echo ""
}

# ========================================
# Main Execution
# ========================================

main() {
    print_header "🛑 Module 8 Storage Projectors Shutdown"

    # Collect stats before shutdown
    collect_statistics

    # Stop services
    stop_projectors
    stop_infrastructure

    # Cleanup options
    remove_containers
    cleanup_volumes
    disconnect_external_containers

    # Show final status
    show_final_status

    print_header "✅ Shutdown Complete"

    echo "To restart all services:"
    echo "  ./start-module8-projectors.sh"
    echo ""
}

# Run main function
main "$@"
