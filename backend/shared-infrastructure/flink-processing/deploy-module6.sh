#!/bin/bash

# =============================================================================
# Module 6 Deployment Script
# Deploys all Module 6 components in the correct order
# =============================================================================

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KAFKA_BROKER="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
FLINK_JOB_MANAGER="${FLINK_JOB_MANAGER:-localhost:8081}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5433}"

echo -e "${BLUE}=========================================${NC}"
echo -e "${BLUE}Module 6 Deployment - CardioFit${NC}"
echo -e "${BLUE}=========================================${NC}"
echo ""

# =============================================================================
# Step 1: Verify Prerequisites
# =============================================================================

echo -e "${YELLOW}[1/8] Verifying prerequisites...${NC}"

# Check if Modules 1-5 are running
echo "Checking Flink jobs..."
RUNNING_JOBS=$(curl -s http://${FLINK_JOB_MANAGER}/jobs | jq -r '.jobs[] | select(.status == "RUNNING") | .name' || echo "")

if [[ -z "$RUNNING_JOBS" ]]; then
    echo -e "${RED}ERROR: No Flink jobs are running. Please start Modules 1-5 first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Found running Flink jobs:${NC}"
echo "$RUNNING_JOBS"

# Check Kafka connectivity
echo "Checking Kafka connectivity..."
if ! timeout 5 kafka-broker-api-versions --bootstrap-server ${KAFKA_BROKER} >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Cannot connect to Kafka at ${KAFKA_BROKER}${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Kafka is accessible${NC}"

echo ""

# =============================================================================
# Step 2: Create Kafka Topics
# =============================================================================

echo -e "${YELLOW}[2/8] Creating Kafka topics for Module 6...${NC}"

if [ -f "./create-module6-topics.sh" ]; then
    chmod +x ./create-module6-topics.sh
    ./create-module6-topics.sh
    echo -e "${GREEN}✓ Kafka topics created${NC}"
else
    echo -e "${RED}ERROR: create-module6-topics.sh not found${NC}"
    exit 1
fi

echo ""

# =============================================================================
# Step 3: Initialize PostgreSQL Database
# =============================================================================

echo -e "${YELLOW}[3/8] Initializing PostgreSQL analytics database...${NC}"

if [ -f "./sql/init-analytics-db.sql" ]; then
    echo "Creating analytics database schema..."

    # Check if PostgreSQL is accessible
    if ! pg_isready -h ${POSTGRES_HOST} -p ${POSTGRES_PORT} >/dev/null 2>&1; then
        echo -e "${RED}ERROR: PostgreSQL is not accessible at ${POSTGRES_HOST}:${POSTGRES_PORT}${NC}"
        exit 1
    fi

    # Run initialization script
    PGPASSWORD="${POSTGRES_PASSWORD:-cardiofit_analytics_pass}" psql \
        -h ${POSTGRES_HOST} \
        -p ${POSTGRES_PORT} \
        -U cardiofit \
        -d cardiofit_analytics \
        -f ./sql/init-analytics-db.sql \
        > /dev/null 2>&1

    echo -e "${GREEN}✓ Database schema initialized${NC}"
else
    echo -e "${YELLOW}⚠ SQL initialization script not found, skipping...${NC}"
fi

echo ""

# =============================================================================
# Step 4: Build Flink Analytics Engine
# =============================================================================

echo -e "${YELLOW}[4/8] Building Flink Analytics Engine...${NC}"

if [ ! -f "pom.xml" ]; then
    echo -e "${RED}ERROR: pom.xml not found. Are you in the correct directory?${NC}"
    exit 1
fi

echo "Running Maven build..."
mvn clean package -DskipTests -q

if [ ! -f "target/flink-ehr-intelligence-1.0.0.jar" ]; then
    echo -e "${RED}ERROR: JAR file not found after build${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Flink job built successfully${NC}"
echo ""

# =============================================================================
# Step 5: Deploy Flink Analytics Engine
# =============================================================================

echo -e "${YELLOW}[5/8] Deploying Flink Analytics Engine to cluster...${NC}"

# Check if Flink is accessible
if ! curl -s http://${FLINK_JOB_MANAGER}/overview >/dev/null 2>&1; then
    echo -e "${RED}ERROR: Flink JobManager is not accessible at ${FLINK_JOB_MANAGER}${NC}"
    exit 1
fi

# Submit job
echo "Submitting Module 6 Analytics Engine..."
SUBMIT_RESPONSE=$(curl -s -X POST \
    -F "jarfile=@target/flink-ehr-intelligence-1.0.0.jar" \
    http://${FLINK_JOB_MANAGER}/jars/upload)

JAR_ID=$(echo $SUBMIT_RESPONSE | jq -r '.filename' | sed 's/.*\///')

if [ -z "$JAR_ID" ]; then
    echo -e "${RED}ERROR: Failed to upload JAR to Flink${NC}"
    exit 1
fi

echo "Starting job..."
RUN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"entryClass":"com.cardiofit.flink.analytics.Module6_AnalyticsEngine","parallelism":4}' \
    http://${FLINK_JOB_MANAGER}/jars/${JAR_ID}/run)

JOB_ID=$(echo $RUN_RESPONSE | jq -r '.jobid')

if [ -z "$JOB_ID" ] || [ "$JOB_ID" == "null" ]; then
    echo -e "${RED}ERROR: Failed to start Flink job${NC}"
    echo "Response: $RUN_RESPONSE"
    exit 1
fi

echo -e "${GREEN}✓ Flink Analytics Engine deployed (Job ID: ${JOB_ID})${NC}"
echo ""

# Wait for job to start
echo "Waiting for job to start..."
sleep 5

# Verify job is running
JOB_STATUS=$(curl -s http://${FLINK_JOB_MANAGER}/jobs/${JOB_ID} | jq -r '.state')
if [ "$JOB_STATUS" == "RUNNING" ]; then
    echo -e "${GREEN}✓ Job is running${NC}"
else
    echo -e "${YELLOW}⚠ Job status: ${JOB_STATUS}${NC}"
fi

echo ""

# =============================================================================
# Step 6: Start Module 6 Services with Docker Compose
# =============================================================================

echo -e "${YELLOW}[6/8] Starting Module 6 services with Docker Compose...${NC}"

if [ ! -f "docker-compose-module6.yml" ]; then
    echo -e "${RED}ERROR: docker-compose-module6.yml not found${NC}"
    exit 1
fi

echo "Starting infrastructure services..."
docker-compose -f docker-compose-module6.yml up -d redis-analytics postgres-analytics influxdb

echo "Waiting for infrastructure to be healthy..."
sleep 10

echo "Starting application services..."
docker-compose -f docker-compose-module6.yml up -d dashboard-api notification-service websocket-server export-reporting-service

echo "Waiting for services to start..."
sleep 15

echo "Starting dashboard UI..."
docker-compose -f docker-compose-module6.yml up -d dashboard-ui

echo -e "${GREEN}✓ All Module 6 services started${NC}"
echo ""

# =============================================================================
# Step 7: Verify Data Flow
# =============================================================================

echo -e "${YELLOW}[7/8] Verifying data flow...${NC}"

echo "Checking Kafka topics for data..."
TOPIC="analytics-patient-census"

# Check if topic has messages
TOPIC_MESSAGES=$(timeout 5 kafka-console-consumer \
    --bootstrap-server ${KAFKA_BROKER} \
    --topic ${TOPIC} \
    --max-messages 1 \
    --timeout-ms 3000 2>/dev/null || echo "")

if [ -n "$TOPIC_MESSAGES" ]; then
    echo -e "${GREEN}✓ Data flowing through Kafka topics${NC}"
else
    echo -e "${YELLOW}⚠ No data in ${TOPIC} yet (this is normal if modules are just starting)${NC}"
fi

echo "Checking Redis cache..."
REDIS_KEYS=$(docker exec cardiofit-redis-analytics redis-cli KEYS 'census:*' 2>/dev/null || echo "")

if [ -n "$REDIS_KEYS" ]; then
    echo -e "${GREEN}✓ Data cached in Redis${NC}"
else
    echo -e "${YELLOW}⚠ No data in Redis cache yet${NC}"
fi

echo ""

# =============================================================================
# Step 8: Health Checks
# =============================================================================

echo -e "${YELLOW}[8/8] Running health checks...${NC}"

# Dashboard API
echo -n "Dashboard API (port 4001): "
if curl -s http://localhost:4001/health >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

# WebSocket Server
echo -n "WebSocket Server (port 8082): "
if curl -s http://localhost:8082/health >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

# Notification Service
echo -n "Notification Service (port 8090): "
if curl -s http://localhost:8090/actuator/health >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

# Export & Reporting Service
echo -n "Export & Reporting Service (port 8050): "
if curl -s http://localhost:8050/actuator/health >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

# Dashboard UI
echo -n "Dashboard UI (port 3000): "
if curl -s http://localhost:3000 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Accessible${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo ""

# =============================================================================
# Deployment Complete
# =============================================================================

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}Module 6 Deployment Complete!${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo -e "${BLUE}Access Points:${NC}"
echo "  • Dashboard UI:        http://localhost:3000"
echo "  • GraphQL API:         http://localhost:4001/graphql"
echo "  • WebSocket:           ws://localhost:8082/dashboard/realtime"
echo "  • Notification API:    http://localhost:8090"
echo "  • Export & Reporting:  http://localhost:8050"
echo "  • Flink Web UI:        http://${FLINK_JOB_MANAGER}"
echo "  • kafka-ui:            http://localhost:8080"
echo ""
echo -e "${BLUE}Monitoring:${NC}"
echo "  • Health checks:       http://localhost:4001/health"
echo "  • WebSocket metrics:   http://localhost:8082/metrics"
echo "  • Flink job metrics:   http://${FLINK_JOB_MANAGER}/jobs/${JOB_ID}"
echo ""
echo -e "${BLUE}Logs:${NC}"
echo "  • View all services:   docker-compose -f docker-compose-module6.yml logs -f"
echo "  • Dashboard API:       docker-compose -f docker-compose-module6.yml logs -f dashboard-api"
echo "  • WebSocket:           docker-compose -f docker-compose-module6.yml logs -f websocket-server"
echo ""
echo -e "${BLUE}Quick Commands:${NC}"
echo "  • Status:              docker-compose -f docker-compose-module6.yml ps"
echo "  • Stop all:            docker-compose -f docker-compose-module6.yml down"
echo "  • Restart service:     docker-compose -f docker-compose-module6.yml restart <service>"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. Open dashboard at http://localhost:3000"
echo "  2. Monitor Kafka topics for data flow"
echo "  3. Check Flink Web UI for job metrics"
echo "  4. Review logs for any errors"
echo ""
