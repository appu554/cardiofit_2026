#!/bin/bash

# CardioFit Shared Data Stores Startup Script
# Starts all global data stores in the correct order with health checks

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

echo -e "${BLUE}🏥 Starting CardioFit Shared Data Stores...${NC}"
echo "================================================"

# Function to check if a service is healthy
check_service_health() {
    local service_name=$1
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}⏳ Waiting for $service_name to be healthy...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if docker-compose -f "$COMPOSE_FILE" ps "$service_name" | grep -q "healthy"; then
            echo -e "${GREEN}✅ $service_name is healthy${NC}"
            return 0
        fi

        echo -n "."
        sleep 5
        attempt=$((attempt + 1))
    done

    echo -e "${RED}❌ $service_name failed to become healthy${NC}"
    return 1
}

# Function to wait for service port
wait_for_port() {
    local host=$1
    local port=$2
    local service_name=$3
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}⏳ Waiting for $service_name port $port...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if nc -z "$host" "$port" 2>/dev/null; then
            echo -e "${GREEN}✅ $service_name port $port is ready${NC}"
            return 0
        fi

        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    echo -e "${RED}❌ $service_name port $port is not responding${NC}"
    return 1
}

# Check if Docker and Docker Compose are available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed or not in PATH${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed or not in PATH${NC}"
    exit 1
fi

# Create necessary directories
echo -e "${BLUE}📁 Creating configuration directories...${NC}"
mkdir -p "$SCRIPT_DIR/redis/config"
mkdir -p "$SCRIPT_DIR/clickhouse/config"
mkdir -p "$SCRIPT_DIR/clickhouse/users"
mkdir -p "$SCRIPT_DIR/clickhouse/init"
mkdir -p "$SCRIPT_DIR/neo4j/conf"
mkdir -p "$SCRIPT_DIR/neo4j/init"
mkdir -p "$SCRIPT_DIR/elasticsearch/config"
mkdir -p "$SCRIPT_DIR/elasticsearch/synonyms"
mkdir -p "$SCRIPT_DIR/graphdb/config"
mkdir -p "$SCRIPT_DIR/graphdb/import"
mkdir -p "$SCRIPT_DIR/kibana/config"
mkdir -p "$SCRIPT_DIR/kibana/dashboards"
mkdir -p "$SCRIPT_DIR/monitoring/alerts"
mkdir -p "$SCRIPT_DIR/monitoring/grafana/provisioning/datasources"
mkdir -p "$SCRIPT_DIR/monitoring/grafana/provisioning/dashboards"
mkdir -p "$SCRIPT_DIR/monitoring/grafana/dashboards"

# Pull latest images
echo -e "${BLUE}🐳 Pulling latest Docker images...${NC}"
docker-compose -f "$COMPOSE_FILE" pull

# Start infrastructure services first (Redis, then others)
echo -e "${BLUE}🚀 Starting Redis Master and Replica...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d redis-master redis-replica
sleep 10

# Wait for Redis to be ready
wait_for_port "localhost" "6379" "Redis Master" || exit 1
wait_for_port "localhost" "6380" "Redis Replica" || exit 1

echo -e "${BLUE}🚀 Starting ClickHouse...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d clickhouse
wait_for_port "localhost" "8123" "ClickHouse" || exit 1

echo -e "${BLUE}🚀 Starting Neo4j...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d neo4j
wait_for_port "localhost" "7474" "Neo4j HTTP" || exit 1
wait_for_port "localhost" "7687" "Neo4j Bolt" || exit 1

echo -e "${BLUE}🚀 Starting Elasticsearch...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d elasticsearch
wait_for_port "localhost" "9200" "Elasticsearch" || exit 1

echo -e "${BLUE}🚀 Starting Ontotext GraphDB...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d graphdb
wait_for_port "localhost" "7200" "GraphDB" || exit 1

# Start monitoring services
echo -e "${BLUE}🚀 Starting Monitoring Services...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d prometheus grafana
wait_for_port "localhost" "9090" "Prometheus" || exit 1
wait_for_port "localhost" "3000" "Grafana" || exit 1

# Start UI services
echo -e "${BLUE}🚀 Starting UI Services...${NC}"
docker-compose -f "$COMPOSE_FILE" up -d kibana redis-insight

# Wait for all services to be fully healthy
echo -e "${BLUE}🔍 Performing health checks...${NC}"
sleep 30

# Display service status
echo -e "${BLUE}📊 Service Status:${NC}"
echo "================================================"
docker-compose -f "$COMPOSE_FILE" ps

# Display access URLs
echo -e "${GREEN}🎉 CardioFit Data Stores are ready!${NC}"
echo "================================================"
echo -e "${BLUE}Access URLs:${NC}"
echo "• Neo4j Browser:        http://localhost:7474"
echo "  - Username: neo4j"
echo "  - Password: CardioFit2024!"
echo ""
echo "• ClickHouse:           http://localhost:8123"
echo "  - Username: cardiofit_user"
echo "  - Password: ClickHouse2024!"
echo ""
echo "• Ontotext GraphDB:     http://localhost:7200"
echo "  - Admin Password: admin2024"
echo ""
echo "• Elasticsearch:        http://localhost:9200"
echo "  - Username: elastic"
echo "  - Password: ElasticCardioFit2024!"
echo ""
echo "• Kibana:               http://localhost:5601"
echo "  - Use Elasticsearch credentials"
echo ""
echo "• Redis Insight:        http://localhost:8001"
echo "  - Connect to: redis-master:6379"
echo "  - Password: RedisCardioFit2024!"
echo ""
echo "• Prometheus:           http://localhost:9090"
echo "• Grafana:              http://localhost:3000"
echo "  - Username: admin"
echo "  - Password: GrafanaCardioFit2024!"
echo ""
echo -e "${BLUE}Database Connections:${NC}"
echo "• Redis Master:         localhost:6379"
echo "• Redis Replica:        localhost:6380"
echo "• Neo4j:               bolt://localhost:7687"
echo "• ClickHouse:          localhost:8123 (HTTP), localhost:9000 (Native)"
echo "• GraphDB:             http://localhost:7200"
echo "• Elasticsearch:       localhost:9200"
echo ""
echo -e "${YELLOW}📝 Note: All passwords are stored in the Docker Compose file.${NC}"
echo -e "${YELLOW}🔒 For production, use Docker secrets or environment variables.${NC}"
echo ""
echo -e "${GREEN}✅ All services are ready for CardioFit applications!${NC}"