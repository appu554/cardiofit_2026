#!/bin/bash

# Module 8 Infrastructure Management Script
# Manages MongoDB, Elasticsearch, ClickHouse, and Redis containers

set -e

COMPOSE_FILE="docker-compose.module8-infrastructure.yml"
PROJECT_NAME="module8"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Main commands
start_infrastructure() {
    print_header "Starting Module 8 Infrastructure"

    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d

    print_success "Infrastructure started successfully"
    print_info "Waiting for services to be healthy..."
    sleep 10

    check_health
}

stop_infrastructure() {
    print_header "Stopping Module 8 Infrastructure"

    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down

    print_success "Infrastructure stopped successfully"
}

restart_infrastructure() {
    print_header "Restarting Module 8 Infrastructure"

    stop_infrastructure
    sleep 3
    start_infrastructure
}

check_health() {
    print_header "Checking Service Health"

    # MongoDB
    if docker exec module8-mongodb mongosh --eval "db.adminCommand('ping')" > /dev/null 2>&1; then
        print_success "MongoDB: Healthy (port 27017)"
    else
        print_error "MongoDB: Unhealthy"
    fi

    # Elasticsearch
    if curl -s http://localhost:9200/_cluster/health > /dev/null 2>&1; then
        ES_STATUS=$(curl -s http://localhost:9200/_cluster/health | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        print_success "Elasticsearch: Healthy - Status: $ES_STATUS (ports 9200, 9300)"
    else
        print_error "Elasticsearch: Unhealthy"
    fi

    # ClickHouse
    if curl -s http://localhost:8123/ping > /dev/null 2>&1; then
        print_success "ClickHouse: Healthy (ports 8123 HTTP, 9000 native)"
    else
        print_error "ClickHouse: Unhealthy"
    fi

    # Redis
    if docker exec module8-redis redis-cli ping > /dev/null 2>&1; then
        print_success "Redis: Healthy (port 6379)"
    else
        print_error "Redis: Unhealthy"
    fi
}

show_status() {
    print_header "Module 8 Infrastructure Status"

    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps

    echo ""
    check_health
}

show_logs() {
    SERVICE=$1

    if [ -z "$SERVICE" ]; then
        print_header "All Service Logs"
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f
    else
        print_header "Logs for $SERVICE"
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f "$SERVICE"
    fi
}

show_urls() {
    print_header "Service URLs and Connection Information"

    echo -e "${GREEN}MongoDB:${NC}"
    echo "  Connection URL: mongodb://localhost:27017"
    echo "  Database: module8_clinical"
    echo "  Shell: docker exec -it module8-mongodb mongosh"
    echo ""

    echo -e "${GREEN}Elasticsearch:${NC}"
    echo "  HTTP URL: http://localhost:9200"
    echo "  Cluster Health: http://localhost:9200/_cluster/health"
    echo "  Indices: http://localhost:9200/_cat/indices?v"
    echo "  Native Transport: localhost:9300"
    echo ""

    echo -e "${GREEN}ClickHouse:${NC}"
    echo "  HTTP Interface: http://localhost:8123"
    echo "  Native Client: localhost:9000"
    echo "  Database: module8_analytics"
    echo "  User: module8_user"
    echo "  Password: module8_password"
    echo "  Client: docker exec -it module8-clickhouse clickhouse-client"
    echo ""

    echo -e "${GREEN}Redis:${NC}"
    echo "  Connection: localhost:6379"
    echo "  CLI: docker exec -it module8-redis redis-cli"
    echo ""
}

initialize_databases() {
    print_header "Initializing Databases"

    # Wait for services to be ready
    print_info "Waiting for services to be fully ready..."
    sleep 15

    # MongoDB initialization
    print_info "Initializing MongoDB..."
    docker exec module8-mongodb mongosh --eval '
        use module8_clinical;
        db.createCollection("patients");
        db.createCollection("observations");
        db.createCollection("encounters");
        db.createCollection("medications");
        db.createCollection("alerts");
        db.createCollection("clinical_events");
    ' > /dev/null 2>&1
    print_success "MongoDB initialized with collections"

    # Elasticsearch initialization
    print_info "Initializing Elasticsearch..."
    # Create index templates
    curl -s -X PUT "http://localhost:9200/_index_template/clinical_events" \
      -H "Content-Type: application/json" \
      -d '{
        "index_patterns": ["clinical_events_*"],
        "template": {
          "settings": {
            "number_of_shards": 1,
            "number_of_replicas": 0
          }
        }
      }' > /dev/null 2>&1
    print_success "Elasticsearch initialized with templates"

    # ClickHouse initialization
    print_info "Initializing ClickHouse..."
    docker exec module8-clickhouse clickhouse-client --query "
        CREATE DATABASE IF NOT EXISTS module8_analytics;

        CREATE TABLE IF NOT EXISTS module8_analytics.patient_events (
            event_id String,
            patient_id String,
            event_type String,
            event_time DateTime,
            event_data String
        ) ENGINE = MergeTree()
        ORDER BY (patient_id, event_time);

        CREATE TABLE IF NOT EXISTS module8_analytics.vital_signs (
            measurement_id String,
            patient_id String,
            vital_type String,
            value Float64,
            unit String,
            measured_at DateTime
        ) ENGINE = MergeTree()
        ORDER BY (patient_id, measured_at);
    " > /dev/null 2>&1
    print_success "ClickHouse initialized with tables"

    print_success "All databases initialized successfully"
}

clean_volumes() {
    print_header "Cleaning Volumes"
    print_warning "This will DELETE ALL DATA in Module 8 infrastructure"
    read -p "Are you sure? (yes/no): " -r
    echo

    if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v
        print_success "Volumes cleaned successfully"
    else
        print_info "Operation cancelled"
    fi
}

show_help() {
    cat << EOF
Module 8 Infrastructure Management Script

Usage: $0 [command]

Commands:
    start       Start all infrastructure services
    stop        Stop all infrastructure services
    restart     Restart all infrastructure services
    status      Show service status and health
    health      Check health of all services
    logs        Show logs (all services)
    logs <svc>  Show logs for specific service
                Services: mongodb, elasticsearch, clickhouse, redis
    urls        Show connection URLs and information
    init        Initialize databases with schema
    clean       Clean all volumes (DELETES DATA)
    help        Show this help message

Examples:
    $0 start                    # Start all services
    $0 status                   # Check status
    $0 logs elasticsearch       # View Elasticsearch logs
    $0 init                     # Initialize databases
    $0 clean                    # Clean all data

Service Ports:
    MongoDB:        27017
    Elasticsearch:  9200 (HTTP), 9300 (Transport)
    ClickHouse:     8123 (HTTP), 9000 (Native)
    Redis:          6379

EOF
}

# Main script logic
case "${1:-help}" in
    start)
        start_infrastructure
        ;;
    stop)
        stop_infrastructure
        ;;
    restart)
        restart_infrastructure
        ;;
    status)
        show_status
        ;;
    health)
        check_health
        ;;
    logs)
        show_logs "$2"
        ;;
    urls)
        show_urls
        ;;
    init)
        initialize_databases
        ;;
    clean)
        clean_volumes
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
