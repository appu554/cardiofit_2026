#!/bin/bash

# CardioFit Shared Data Stores Stop Script
# Gracefully stops all data stores and preserves data

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.datastores.yml"

echo -e "${BLUE}🛑 Stopping CardioFit Shared Data Stores...${NC}"
echo "================================================"

# Parse command line arguments
REMOVE_VOLUMES=false
REMOVE_IMAGES=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --remove-volumes)
            REMOVE_VOLUMES=true
            shift
            ;;
        --remove-images)
            REMOVE_IMAGES=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --remove-volumes    Remove all data volumes (WARNING: Data will be lost!)"
            echo "  --remove-images     Remove Docker images after stopping"
            echo "  --help              Show this help message"
            echo ""
            echo "Default behavior: Stop containers but preserve data volumes"
            exit 0
            ;;
        *)
            echo -e "${RED}❌ Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Warn about data loss if removing volumes
if [ "$REMOVE_VOLUMES" = true ]; then
    echo -e "${RED}⚠️  WARNING: You have requested to remove data volumes!${NC}"
    echo -e "${RED}⚠️  This will permanently delete all data in:${NC}"
    echo "   • Neo4j databases and transaction logs"
    echo "   • ClickHouse tables and data"
    echo "   • Elasticsearch indices and documents"
    echo "   • GraphDB repositories and data"
    echo "   • Redis data and snapshots"
    echo "   • Prometheus metrics history"
    echo "   • Grafana dashboards and configurations"
    echo ""
    read -p "Are you sure you want to continue? (type 'yes' to confirm): " -r
    if [[ ! $REPLY == "yes" ]]; then
        echo -e "${YELLOW}💾 Operation cancelled. Data preserved.${NC}"
        exit 0
    fi
fi

# Stop services gracefully in reverse order of dependencies
echo -e "${BLUE}🔄 Stopping UI and monitoring services...${NC}"
docker-compose -f "$COMPOSE_FILE" stop kibana redis-insight grafana prometheus 2>/dev/null || true

echo -e "${BLUE}🔄 Stopping database services...${NC}"
docker-compose -f "$COMPOSE_FILE" stop graphdb elasticsearch neo4j clickhouse 2>/dev/null || true

echo -e "${BLUE}🔄 Stopping Redis services...${NC}"
docker-compose -f "$COMPOSE_FILE" stop redis-replica redis-master 2>/dev/null || true

# Remove containers
echo -e "${BLUE}🗑️  Removing containers...${NC}"
docker-compose -f "$COMPOSE_FILE" down

# Remove volumes if requested
if [ "$REMOVE_VOLUMES" = true ]; then
    echo -e "${RED}🗑️  Removing data volumes...${NC}"
    docker-compose -f "$COMPOSE_FILE" down -v

    # Also remove named volumes explicitly
    echo -e "${YELLOW}🔍 Removing named volumes...${NC}"
    docker volume rm cardiofit_neo4j_data 2>/dev/null || true
    docker volume rm cardiofit_neo4j_logs 2>/dev/null || true
    docker volume rm cardiofit_neo4j_import 2>/dev/null || true
    docker volume rm cardiofit_neo4j_plugins 2>/dev/null || true
    docker volume rm cardiofit_clickhouse_data 2>/dev/null || true
    docker volume rm cardiofit_clickhouse_logs 2>/dev/null || true
    docker volume rm cardiofit_graphdb_data 2>/dev/null || true
    docker volume rm cardiofit_graphdb_work 2>/dev/null || true
    docker volume rm cardiofit_graphdb_logs 2>/dev/null || true
    docker volume rm cardiofit_elasticsearch_data 2>/dev/null || true
    docker volume rm cardiofit_elasticsearch_logs 2>/dev/null || true
    docker volume rm cardiofit_redis_data 2>/dev/null || true
    docker volume rm cardiofit_redis_replica_data 2>/dev/null || true
    docker volume rm cardiofit_redis_insight_data 2>/dev/null || true
    docker volume rm cardiofit_prometheus_data 2>/dev/null || true
    docker volume rm cardiofit_grafana_data 2>/dev/null || true

    echo -e "${RED}💥 All data volumes have been removed!${NC}"
else
    echo -e "${GREEN}💾 Data volumes preserved${NC}"
fi

# Remove images if requested
if [ "$REMOVE_IMAGES" = true ]; then
    echo -e "${BLUE}🗑️  Removing Docker images...${NC}"
    docker-compose -f "$COMPOSE_FILE" down --rmi all
    echo -e "${GREEN}🗑️  Docker images removed${NC}"
fi

# Clean up dangling resources
echo -e "${BLUE}🧹 Cleaning up dangling resources...${NC}"
docker system prune -f > /dev/null 2>&1 || true

# Display final status
echo -e "${GREEN}✅ CardioFit Data Stores stopped successfully!${NC}"
echo "================================================"

if [ "$REMOVE_VOLUMES" = true ]; then
    echo -e "${RED}⚠️  All data has been permanently removed${NC}"
    echo -e "${YELLOW}💡 To start fresh, run: ./start-datastores.sh${NC}"
else
    echo -e "${GREEN}💾 All data has been preserved in Docker volumes${NC}"
    echo -e "${YELLOW}💡 To restart with existing data, run: ./start-datastores.sh${NC}"
fi

echo ""
echo -e "${BLUE}🔍 Volume Status:${NC}"
if [ "$REMOVE_VOLUMES" = true ]; then
    echo "All volumes removed"
else
    echo "Checking preserved volumes..."
    docker volume ls | grep cardiofit || echo "No CardioFit volumes found"
fi

echo ""
echo -e "${BLUE}📊 Container Status:${NC}"
docker ps -a --filter "name=cardiofit-" --format "table {{.Names}}\t{{.Status}}" || echo "No CardioFit containers found"