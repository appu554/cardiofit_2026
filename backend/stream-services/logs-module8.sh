#!/bin/bash

# ========================================
# Module 8 Log Viewer Script
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

# Available services
PROJECTOR_SERVICES=(
    "postgresql-projector"
    "mongodb-projector"
    "elasticsearch-projector"
    "clickhouse-projector"
    "influxdb-projector"
    "ups-projector"
    "fhir-store-projector"
    "neo4j-graph-projector"
)

INFRASTRUCTURE_SERVICES=(
    "mongodb"
    "elasticsearch"
    "clickhouse"
    "redis"
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

print_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

# ========================================
# Usage Information
# ========================================

show_usage() {
    cat << EOF
Module 8 Log Viewer

USAGE:
    ./logs-module8.sh [OPTIONS] [SERVICE]

OPTIONS:
    -f, --follow         Follow log output (tail -f mode)
    -n, --lines NUM      Number of lines to show (default: 100)
    -s, --search TERM    Search for specific term in logs
    -e, --errors         Show only errors and exceptions
    -l, --list           List all available services
    -a, --all            Show logs from all services
    -h, --help           Show this help message

SERVICES:
    Projectors:
        postgresql-projector
        mongodb-projector
        elasticsearch-projector
        clickhouse-projector
        influxdb-projector
        ups-projector
        fhir-store-projector
        neo4j-graph-projector

    Infrastructure:
        mongodb
        elasticsearch
        clickhouse
        redis

EXAMPLES:
    # Follow logs from PostgreSQL projector
    ./logs-module8.sh -f postgresql-projector

    # Show last 50 lines from MongoDB projector
    ./logs-module8.sh -n 50 mongodb-projector

    # Search for errors in all services
    ./logs-module8.sh -a -s "error"

    # Show only errors from FHIR Store projector
    ./logs-module8.sh -e fhir-store-projector

    # Follow all projector logs
    ./logs-module8.sh -f -a

EOF
}

# ========================================
# List Available Services
# ========================================

list_services() {
    print_header "Available Services"

    echo "📊 Projector Services:"
    for service in "${PROJECTOR_SERVICES[@]}"; do
        echo "  - $service"
    done

    echo ""
    echo "🗄️  Infrastructure Services:"
    for service in "${INFRASTRUCTURE_SERVICES[@]}"; do
        echo "  - $service"
    done
}

# ========================================
# Log Viewing Functions
# ========================================

view_logs() {
    local service=$1
    local follow=$2
    local lines=$3
    local search=$4
    local errors_only=$5

    # Build docker-compose logs command
    local cmd="docker-compose -f $COMPOSE_FILE logs"

    # Add follow flag
    if [ "$follow" = true ]; then
        cmd="$cmd --follow"
    fi

    # Add lines flag
    if [ -n "$lines" ]; then
        cmd="$cmd --tail=$lines"
    fi

    # Add service name or all services
    if [ -n "$service" ]; then
        cmd="$cmd $service"
    fi

    print_info "Viewing logs for: ${service:-all services}"

    # Execute command
    if [ -n "$search" ]; then
        print_info "Searching for: $search"
        eval "$cmd" 2>&1 | grep -i "$search"
    elif [ "$errors_only" = true ]; then
        print_info "Showing only errors and exceptions"
        eval "$cmd" 2>&1 | grep -iE "error|exception|failed|critical|fatal"
    else
        eval "$cmd"
    fi
}

view_all_projectors() {
    local follow=$1
    local lines=$2

    print_header "Viewing All Projector Logs"

    local cmd="docker-compose -f $COMPOSE_FILE logs"

    if [ "$follow" = true ]; then
        cmd="$cmd --follow"
    fi

    if [ -n "$lines" ]; then
        cmd="$cmd --tail=$lines"
    fi

    # Add all projector services
    for service in "${PROJECTOR_SERVICES[@]}"; do
        cmd="$cmd $service"
    done

    eval "$cmd"
}

# ========================================
# Advanced Log Analysis
# ========================================

analyze_logs() {
    local service=$1

    print_header "Log Analysis: $service"

    local log_output=$(docker-compose -f "$COMPOSE_FILE" logs --tail=1000 "$service" 2>&1)

    # Count log levels
    echo "📊 Log Level Distribution:"
    echo "  INFO:     $(echo "$log_output" | grep -c "INFO")"
    echo "  WARNING:  $(echo "$log_output" | grep -c "WARNING")"
    echo "  ERROR:    $(echo "$log_output" | grep -c "ERROR")"
    echo "  CRITICAL: $(echo "$log_output" | grep -c "CRITICAL")"
    echo ""

    # Show recent errors
    echo "🔴 Recent Errors (last 5):"
    echo "$log_output" | grep -i "error\|exception" | tail -5
    echo ""

    # Show processing statistics
    echo "📈 Processing Statistics:"
    echo "$log_output" | grep -i "processed\|consumed\|written" | tail -10
}

# ========================================
# Export Logs
# ========================================

export_logs() {
    local service=$1
    local output_dir="${SCRIPT_DIR}/module8-logs-export"

    mkdir -p "$output_dir"

    local timestamp=$(date +%Y%m%d-%H%M%S)
    local output_file="${output_dir}/${service}-${timestamp}.log"

    print_info "Exporting logs to: $output_file"

    if [ -n "$service" ]; then
        docker-compose -f "$COMPOSE_FILE" logs "$service" > "$output_file"
    else
        docker-compose -f "$COMPOSE_FILE" logs > "${output_dir}/all-services-${timestamp}.log"
    fi

    print_success "Logs exported successfully"
}

# ========================================
# Parse Command Line Arguments
# ========================================

FOLLOW=false
LINES=""
SEARCH=""
ERRORS_ONLY=false
LIST_SERVICES=false
ALL_SERVICES=false
SERVICE=""
ANALYZE=false
EXPORT=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--follow)
            FOLLOW=true
            shift
            ;;
        -n|--lines)
            LINES="$2"
            shift 2
            ;;
        -s|--search)
            SEARCH="$2"
            shift 2
            ;;
        -e|--errors)
            ERRORS_ONLY=true
            shift
            ;;
        -l|--list)
            LIST_SERVICES=true
            shift
            ;;
        -a|--all)
            ALL_SERVICES=true
            shift
            ;;
        --analyze)
            ANALYZE=true
            shift
            ;;
        --export)
            EXPORT=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            SERVICE="$1"
            shift
            ;;
    esac
done

# ========================================
# Main Execution
# ========================================

main() {
    # List services if requested
    if [ "$LIST_SERVICES" = true ]; then
        list_services
        exit 0
    fi

    # Analyze logs if requested
    if [ "$ANALYZE" = true ]; then
        if [ -z "$SERVICE" ]; then
            print_error "Please specify a service to analyze"
            exit 1
        fi
        analyze_logs "$SERVICE"
        exit 0
    fi

    # Export logs if requested
    if [ "$EXPORT" = true ]; then
        export_logs "$SERVICE"
        exit 0
    fi

    # View all services if requested
    if [ "$ALL_SERVICES" = true ]; then
        view_all_projectors "$FOLLOW" "$LINES"
        exit 0
    fi

    # View specific service logs
    if [ -n "$SERVICE" ]; then
        view_logs "$SERVICE" "$FOLLOW" "$LINES" "$SEARCH" "$ERRORS_ONLY"
    else
        print_error "Please specify a service or use -a for all services"
        echo ""
        show_usage
        exit 1
    fi
}

# Run main function
main
