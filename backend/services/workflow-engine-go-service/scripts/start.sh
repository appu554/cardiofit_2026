#!/bin/bash

# Workflow Engine Service - Quick Start Script

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting Workflow Engine Service${NC}"
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ docker-compose is not installed. Please install it and try again.${NC}"
    exit 1
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️  No .env file found. Creating from template...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✅ Created .env file. You may want to review and customize it.${NC}"
fi

# Create required directories
echo -e "${BLUE}📁 Creating required directories...${NC}"
mkdir -p logs
mkdir -p configs/grafana/provisioning/dashboards
mkdir -p configs/grafana/provisioning/datasources

# Create Prometheus configuration if it doesn't exist
if [ ! -f configs/prometheus.yml ]; then
    echo -e "${YELLOW}⚠️  Creating Prometheus configuration...${NC}"
    mkdir -p configs
    cat > configs/prometheus.yml << EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
  
  - job_name: 'workflow-engine'
    static_configs:
      - targets: ['workflow-engine:8017']
    scrape_interval: 5s
    metrics_path: /metrics
EOF
    echo -e "${GREEN}✅ Created Prometheus configuration${NC}"
fi

# Create Grafana datasource configuration
if [ ! -f configs/grafana/provisioning/datasources/prometheus.yml ]; then
    cat > configs/grafana/provisioning/datasources/prometheus.yml << EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
EOF
    echo -e "${GREEN}✅ Created Grafana datasource configuration${NC}"
fi

echo ""
echo -e "${BLUE}🐳 Starting Docker services...${NC}"

# Start the infrastructure services first
echo -e "${YELLOW}🔧 Starting infrastructure services (database, monitoring)...${NC}"
docker-compose up -d postgres redis prometheus grafana jaeger adminer

# Wait for database to be ready
echo -e "${YELLOW}⏳ Waiting for database to be ready...${NC}"
sleep 15

# Check if database is ready
while ! docker-compose exec -T postgres pg_isready -U workflow_user -d workflow_engine > /dev/null 2>&1; do
    echo -e "${YELLOW}⏳ Still waiting for database...${NC}"
    sleep 5
done

echo -e "${GREEN}✅ Database is ready${NC}"

# Build and start the main application
echo -e "${YELLOW}🔨 Building and starting Workflow Engine Service...${NC}"
docker-compose up -d --build workflow-engine

# Wait for the service to be ready
echo -e "${YELLOW}⏳ Waiting for Workflow Engine Service to be ready...${NC}"
sleep 10

# Health check
max_attempts=12
attempt=1
while [ $attempt -le $max_attempts ]; do
    if curl -f -s http://localhost:8017/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Workflow Engine Service is ready!${NC}"
        break
    fi
    
    if [ $attempt -eq $max_attempts ]; then
        echo -e "${RED}❌ Workflow Engine Service failed to start. Check logs with: docker-compose logs workflow-engine${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}⏳ Attempt $attempt/$max_attempts - Service not ready yet...${NC}"
    sleep 5
    ((attempt++))
done

echo ""
echo -e "${GREEN}🎉 All services are up and running!${NC}"
echo ""
echo -e "${BLUE}📋 Service Access URLs:${NC}"
echo -e "  • ${GREEN}Workflow Engine API:${NC} http://localhost:8017"
echo -e "  • ${GREEN}Health Check:${NC}        http://localhost:8017/health"
echo -e "  • ${GREEN}GraphQL Playground:${NC}  http://localhost:8017/graphql"
echo -e "  • ${GREEN}Metrics:${NC}             http://localhost:8017/metrics"
echo ""
echo -e "${BLUE}🔍 Monitoring & Management:${NC}"
echo -e "  • ${GREEN}Grafana:${NC}             http://localhost:3000 (admin:admin123)"
echo -e "  • ${GREEN}Prometheus:${NC}          http://localhost:9090"
echo -e "  • ${GREEN}Jaeger Tracing:${NC}      http://localhost:16686"
echo -e "  • ${GREEN}Database Admin:${NC}      http://localhost:8080"
echo ""
echo -e "${BLUE}📊 Quick Commands:${NC}"
echo -e "  • View logs:              ${YELLOW}docker-compose logs -f workflow-engine${NC}"
echo -e "  • View all logs:          ${YELLOW}docker-compose logs -f${NC}"
echo -e "  • Stop services:          ${YELLOW}docker-compose stop${NC}"
echo -e "  • Stop & remove:          ${YELLOW}docker-compose down${NC}"
echo -e "  • Restart service:        ${YELLOW}docker-compose restart workflow-engine${NC}"
echo ""
echo -e "${BLUE}🧪 Testing:${NC}"
echo -e "  • Health check:           ${YELLOW}curl http://localhost:8017/health${NC}"
echo -e "  • Load test:              ${YELLOW}make load-test${NC}"
echo ""
echo -e "${GREEN}✨ Setup complete! The Workflow Engine Service is ready for use.${NC}"