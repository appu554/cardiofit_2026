#!/bin/bash

# Deploy CardioFit Hybrid Kafka Architecture with Docker
# Complete deployment sequence for production-ready system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.hybrid-kafka.yml"
PROJECT_NAME="cardiofit"

echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}    CardioFit Hybrid Kafka Architecture Deployment${NC}"
echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Function to check Docker and Docker Compose
check_prerequisites() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Checking Prerequisites${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Check Docker
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}❌ Docker is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Docker installed: $(docker --version)${NC}"

    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo -e "${RED}❌ Docker Compose is not installed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Docker Compose installed${NC}"

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        echo -e "${RED}❌ Docker is not running${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Docker is running${NC}"
}

# Function to clean up previous deployment
cleanup_previous() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Cleaning Previous Deployment${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    if [ "$1" == "--clean" ]; then
        echo -e "${YELLOW}⚠️  Stopping and removing existing containers...${NC}"
        docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME down -v
        echo -e "${GREEN}✅ Cleanup complete${NC}"
    else
        echo -e "${YELLOW}ℹ️  Keeping existing volumes (use --clean to remove)${NC}"
        docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME down
    fi
}

# Function to start infrastructure
start_infrastructure() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Phase 1: Starting Kafka Infrastructure${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    echo -e "${YELLOW}🚀 Starting Zookeeper and Kafka...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d zookeeper kafka

    # Wait for Kafka to be healthy
    echo -e "${YELLOW}⏳ Waiting for Kafka to be ready...${NC}"
    while ! docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T kafka \
        kafka-topics --bootstrap-server kafka:9092 --list &> /dev/null; do
        echo -n "."
        sleep 5
    done
    echo ""
    echo -e "${GREEN}✅ Kafka is ready${NC}"

    echo -e "${YELLOW}🚀 Starting Schema Registry...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d schema-registry

    echo -e "${YELLOW}🚀 Starting Kafka Connect...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d kafka-connect

    echo -e "${YELLOW}🚀 Starting Kafka UI...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d kafka-ui
}

# Function to create topics and deploy connectors
initialize_kafka() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Phase 2: Creating Topics & Deploying Connectors${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    echo -e "${YELLOW}🎯 Running initialization container...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up init-kafka

    # Verify topics
    echo ""
    echo -e "${YELLOW}📋 Verifying hybrid topics...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T kafka \
        kafka-topics --bootstrap-server kafka:9092 --list | grep "prod.ehr" || true
}

# Function to start data stores
start_datastores() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Phase 3: Starting Downstream Data Stores${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    echo -e "${YELLOW}🚀 Starting Redis...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d redis

    echo -e "${YELLOW}🚀 Starting ClickHouse...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d clickhouse

    echo -e "${YELLOW}🚀 Starting Neo4j...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d neo4j

    # Wait for data stores to be ready
    echo -e "${YELLOW}⏳ Waiting for data stores to be ready...${NC}"
    sleep 20

    # Verify data stores
    echo -e "${YELLOW}🔍 Verifying data stores...${NC}"

    # Check Redis
    if docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T redis redis-cli ping | grep -q PONG; then
        echo -e "${GREEN}✅ Redis is ready${NC}"
    else
        echo -e "${RED}❌ Redis is not responding${NC}"
    fi

    # Check ClickHouse
    if docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T clickhouse \
        clickhouse-client --query "SELECT 1" &> /dev/null; then
        echo -e "${GREEN}✅ ClickHouse is ready${NC}"
    else
        echo -e "${RED}❌ ClickHouse is not responding${NC}"
    fi

    # Check Neo4j (may take longer to start)
    echo -e "${YELLOW}⏳ Waiting for Neo4j (may take up to 60s)...${NC}"
    sleep 30
    if docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T neo4j \
        cypher-shell -u neo4j -p 'CardioFit2024!' "RETURN 1" &> /dev/null; then
        echo -e "${GREEN}✅ Neo4j is ready${NC}"
    else
        echo -e "${YELLOW}⚠️  Neo4j may still be initializing${NC}"
    fi
}

# Function to deploy Flink job
deploy_flink() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Phase 4: Deploying Flink Processing Pipeline${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Check if JAR exists
    if [ ! -f "../flink-processing/target/cardiofit-flink-processing-*.jar" ]; then
        echo -e "${YELLOW}⚠️  Flink JAR not found. Building...${NC}"

        # Build Flink JAR if Maven is available
        if command -v mvn &> /dev/null; then
            cd ../flink-processing
            mvn clean package -DskipTests
            cd -
        else
            echo -e "${RED}❌ Maven not installed. Please build the Flink JAR manually:${NC}"
            echo "   cd ../flink-processing && mvn clean package"
            echo -e "${YELLOW}Continuing without Flink deployment...${NC}"
            return
        fi
    fi

    echo -e "${YELLOW}🚀 Starting Flink JobManager...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d flink-jobmanager

    echo -e "${YELLOW}⏳ Waiting for JobManager to be ready...${NC}"
    while ! curl -f http://localhost:8082/overview &> /dev/null; do
        echo -n "."
        sleep 5
    done
    echo ""
    echo -e "${GREEN}✅ JobManager is ready${NC}"

    echo -e "${YELLOW}🚀 Starting Flink TaskManagers...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME up -d --scale flink-taskmanager=2 flink-taskmanager

    # Submit Flink job
    if [ -f "../flink-processing/target/cardiofit-flink-processing-*.jar" ]; then
        echo -e "${YELLOW}📦 Submitting Flink job...${NC}"
        docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T flink-jobmanager \
            flink run -d /opt/flink/usrlib/cardiofit-flink-processing-*.jar || \
            echo -e "${YELLOW}⚠️  Job submission failed - submit manually via Flink UI${NC}"
    fi
}

# Function to verify deployment
verify_deployment() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}Phase 5: Verification & Testing${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    echo -e "${YELLOW}📊 System Status:${NC}"

    # Check all services
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME ps

    echo ""
    echo -e "${YELLOW}🔍 Testing end-to-end flow...${NC}"

    # Create test event
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T kafka \
        bash -c 'echo "{\"eventId\":\"test-001\",\"patientId\":\"patient-123\",\"eventType\":\"vital_signs\",\"timestamp\":\"2024-01-01T10:00:00Z\",\"isCritical\":true}" | \
        kafka-console-producer --bootstrap-server kafka:9092 --topic patient-events'

    echo -e "${GREEN}✅ Test event sent${NC}"

    # Check if event appears in enriched topic
    echo -e "${YELLOW}⏳ Waiting for event processing...${NC}"
    sleep 5

    echo -e "${YELLOW}📨 Checking hybrid topics for processed events...${NC}"
    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec -T kafka \
        kafka-console-consumer --bootstrap-server kafka:9092 \
        --topic prod.ehr.events.enriched \
        --from-beginning --max-messages 1 --timeout-ms 5000 || \
        echo -e "${YELLOW}⚠️  No events in enriched topic yet${NC}"
}

# Function to show access URLs
show_access_urls() {
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}    ✅ Deployment Complete!${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${CYAN}📡 Access URLs:${NC}"
    echo "  • Kafka UI:         http://localhost:8080"
    echo "  • Flink UI:         http://localhost:8082"
    echo "  • Schema Registry:  http://localhost:8081"
    echo "  • Kafka Connect:    http://localhost:8083"
    echo "  • Neo4j Browser:    http://localhost:7474"
    echo "  • ClickHouse:       http://localhost:8123"
    echo ""
    echo -e "${CYAN}🔌 Connection Endpoints:${NC}"
    echo "  • Kafka Bootstrap:  localhost:29092"
    echo "  • Redis:           localhost:6379"
    echo "  • Neo4j Bolt:      localhost:7687"
    echo "  • ClickHouse:      localhost:9000"
    echo ""
    echo -e "${CYAN}📝 Useful Commands:${NC}"
    echo "  • View logs:       docker-compose -f $COMPOSE_FILE logs -f [service]"
    echo "  • List topics:     docker-compose -f $COMPOSE_FILE exec kafka kafka-topics --bootstrap-server kafka:9092 --list"
    echo "  • Stop services:   docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME down"
    echo "  • Clean all:       docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME down -v"
    echo ""
    echo -e "${YELLOW}⚠️  Note: Some services may take a few minutes to fully initialize${NC}"
}

# Main execution
main() {
    check_prerequisites

    if [ "$1" == "stop" ]; then
        echo -e "${YELLOW}🛑 Stopping all services...${NC}"
        docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME down
        echo -e "${GREEN}✅ All services stopped${NC}"
        exit 0
    fi

    cleanup_previous "$1"
    start_infrastructure
    initialize_kafka
    start_datastores
    deploy_flink
    verify_deployment
    show_access_urls
}

# Run main function
main "$@"