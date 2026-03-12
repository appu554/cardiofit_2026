#!/bin/bash

# Enhanced Flink EHR Intelligence Engine Startup Script
# Tests all data store connections before starting Flink processing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SHARED_INFRA_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}🏥 Starting Flink EHR Intelligence Engine with Data Store Integration${NC}"
echo "======================================================================"

# Function to check if shared data stores are running
check_shared_datastores() {
    echo -e "${BLUE}🔍 Checking shared data store infrastructure...${NC}"

    # Check if the shared infrastructure is running
    if ! docker network ls | grep -q "cardiofit-network"; then
        echo -e "${RED}❌ CardioFit shared network not found!${NC}"
        echo -e "${YELLOW}💡 Please start the shared data stores first:${NC}"
        echo "   cd $SHARED_INFRA_DIR"
        echo "   ./start-datastores.sh"
        return 1
    fi

    # Check individual services
    local services=("cardiofit-neo4j" "cardiofit-clickhouse" "cardiofit-elasticsearch" "cardiofit-redis-master" "cardiofit-redis-replica")
    local missing_services=()

    for service in "${services[@]}"; do
        if ! docker ps --format "table {{.Names}}" | grep -q "$service"; then
            missing_services+=("$service")
        fi
    done

    if [ ${#missing_services[@]} -ne 0 ]; then
        echo -e "${RED}❌ Missing data store services:${NC}"
        printf '   %s\n' "${missing_services[@]}"
        echo -e "${YELLOW}💡 Please start the shared data stores:${NC}"
        echo "   cd $SHARED_INFRA_DIR"
        echo "   ./start-datastores.sh"
        return 1
    fi

    echo -e "${GREEN}✅ All shared data stores are running${NC}"
    return 0
}

# Function to test data store connections
test_connections() {
    echo -e "${BLUE}🔧 Testing data store connections...${NC}"

    # Load environment variables
    if [ -f "$SCRIPT_DIR/flink-datastores.env" ]; then
        export $(grep -v '^#' "$SCRIPT_DIR/flink-datastores.env" | xargs)
        echo -e "${GREEN}✅ Loaded Flink data store configuration${NC}"
    else
        echo -e "${YELLOW}⚠️ flink-datastores.env not found, using defaults${NC}"
    fi

    # Test Neo4j connection
    echo -e "${YELLOW}Testing Neo4j...${NC}"
    if docker run --rm --network cardiofit-network neo4j:5.15-community cypher-shell -a "bolt://cardiofit-neo4j:7687" -u neo4j -p "CardioFit2024!" "RETURN 'Connected' as status" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Neo4j connection successful${NC}"
    else
        echo -e "${RED}❌ Neo4j connection failed${NC}"
        return 1
    fi

    # Test ClickHouse connection
    echo -e "${YELLOW}Testing ClickHouse...${NC}"
    if docker run --rm --network cardiofit-network curlimages/curl:latest curl -s "http://cardiofit-clickhouse:8123/ping" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ ClickHouse connection successful${NC}"
    else
        echo -e "${RED}❌ ClickHouse connection failed${NC}"
        return 1
    fi

    # Test Elasticsearch connection
    echo -e "${YELLOW}Testing Elasticsearch...${NC}"
    if docker run --rm --network cardiofit-network curlimages/curl:latest curl -s -u "elastic:ElasticCardioFit2024!" "http://cardiofit-elasticsearch:9200/_cluster/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Elasticsearch connection successful${NC}"
    else
        echo -e "${RED}❌ Elasticsearch connection failed${NC}"
        return 1
    fi

    # Test Redis Master connection
    echo -e "${YELLOW}Testing Redis Master...${NC}"
    if docker run --rm --network cardiofit-network redis:7.2-alpine redis-cli -h cardiofit-redis-master -p 6379 -a "RedisCardioFit2024!" ping > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis Master connection successful${NC}"
    else
        echo -e "${RED}❌ Redis Master connection failed${NC}"
        return 1
    fi

    # Test Redis Replica connection
    echo -e "${YELLOW}Testing Redis Replica...${NC}"
    if docker run --rm --network cardiofit-network redis:7.2-alpine redis-cli -h cardiofit-redis-replica -p 6379 -a "RedisCardioFit2024!" ping > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis Replica connection successful${NC}"
    else
        echo -e "${RED}❌ Redis Replica connection failed${NC}"
        return 1
    fi

    echo -e "${GREEN}🎉 All data store connections verified!${NC}"
    return 0
}

# Function to build Flink application
build_application() {
    echo -e "${BLUE}🔨 Building Flink EHR Intelligence Engine...${NC}"

    if [ ! -f "$SCRIPT_DIR/pom.xml" ]; then
        echo -e "${RED}❌ pom.xml not found in $SCRIPT_DIR${NC}"
        return 1
    fi

    cd "$SCRIPT_DIR"

    # Check if Maven is available
    if ! command -v mvn &> /dev/null; then
        echo -e "${RED}❌ Maven not found. Please install Maven first.${NC}"
        return 1
    fi

    # Build the application
    echo -e "${YELLOW}⏳ Running Maven build...${NC}"
    if mvn clean package -DskipTests > build.log 2>&1; then
        echo -e "${GREEN}✅ Build successful${NC}"
        rm -f build.log
    else
        echo -e "${RED}❌ Build failed. Check build.log for details${NC}"
        tail -20 build.log
        return 1
    fi
}

# Function to start Flink cluster
start_flink_cluster() {
    echo -e "${BLUE}🚀 Starting Flink cluster...${NC}"

    cd "$SCRIPT_DIR"

    # Stop any existing Flink containers
    echo -e "${YELLOW}🔄 Stopping any existing Flink containers...${NC}"
    docker-compose down > /dev/null 2>&1 || true

    # Start Flink cluster
    echo -e "${YELLOW}⏳ Starting Flink cluster with data store connections...${NC}"
    docker-compose up -d

    # Wait for Flink JobManager to be ready
    echo -e "${YELLOW}⏳ Waiting for Flink JobManager to be ready...${NC}"
    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:8081/overview > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Flink JobManager is ready${NC}"
            break
        fi

        echo -n "."
        sleep 5
        attempt=$((attempt + 1))
    done

    if [ $attempt -gt $max_attempts ]; then
        echo -e "${RED}❌ Flink JobManager failed to start${NC}"
        return 1
    fi

    # Wait for TaskManagers to register
    echo -e "${YELLOW}⏳ Waiting for TaskManagers to register...${NC}"
    sleep 10

    local taskmanagers=$(curl -s http://localhost:8081/taskmanagers | jq '.taskmanagers | length' 2>/dev/null || echo "0")
    if [ "$taskmanagers" -ge 3 ]; then
        echo -e "${GREEN}✅ $taskmanagers TaskManagers registered${NC}"
    else
        echo -e "${YELLOW}⚠️ Only $taskmanagers TaskManagers registered (expected 3)${NC}"
    fi
}

# Function to submit Flink jobs
submit_flink_jobs() {
    echo -e "${BLUE}📊 Available Flink job modules:${NC}"
    echo "1. Module 1: Ingestion & Gateway"
    echo "2. Module 2: Context Assembly"
    echo "3. Module 3: Semantic Mesh"
    echo "4. Module 4: Pattern Detection"
    echo "5. Module 5: ML Inference"
    echo "6. Module 6: Egress Routing"
    echo "7. All Modules (Complete Pipeline)"
    echo ""

    read -p "Which module would you like to start? (1-7 or 'skip'): " -r

    case $REPLY in
        1)
            echo -e "${BLUE}🚀 Starting Module 1: Ingestion & Gateway${NC}"
            # Submit Module 1 job here
            ;;
        2)
            echo -e "${BLUE}🚀 Starting Module 2: Context Assembly${NC}"
            # Submit Module 2 job here
            ;;
        3)
            echo -e "${BLUE}🚀 Starting Module 3: Semantic Mesh${NC}"
            # Submit Module 3 job here
            ;;
        4)
            echo -e "${BLUE}🚀 Starting Module 4: Pattern Detection${NC}"
            # Submit Module 4 job here
            ;;
        5)
            echo -e "${BLUE}🚀 Starting Module 5: ML Inference${NC}"
            # Submit Module 5 job here
            ;;
        6)
            echo -e "${BLUE}🚀 Starting Module 6: Egress Routing${NC}"
            # Submit Module 6 job here
            ;;
        7)
            echo -e "${BLUE}🚀 Starting Complete EHR Intelligence Pipeline${NC}"
            # Submit all modules in order
            ;;
        skip|"")
            echo -e "${YELLOW}⏭️ Skipping job submission${NC}"
            ;;
        *)
            echo -e "${YELLOW}⏭️ Invalid selection, skipping job submission${NC}"
            ;;
    esac
}

# Function to display access information
display_access_info() {
    echo -e "${GREEN}🎉 Flink EHR Intelligence Engine is ready!${NC}"
    echo "======================================================================"
    echo -e "${BLUE}🌐 Access URLs:${NC}"
    echo "• Flink Web UI:         http://localhost:8081"
    echo "• Flink Metrics:        http://localhost:9090 (Prometheus)"
    echo "• Flink Dashboards:     http://localhost:3001 (Grafana)"
    echo ""
    echo -e "${BLUE}🗄️ Data Store Access:${NC}"
    echo "• Neo4j Browser:        http://localhost:7474"
    echo "• ClickHouse:           http://localhost:8123"
    echo "• Elasticsearch:        http://localhost:9200"
    echo "• Redis Master:         localhost:6379"
    echo "• Redis Replica:        localhost:6380"
    echo ""
    echo -e "${BLUE}📊 Monitoring:${NC}"
    echo "• Prometheus:           http://localhost:9090"
    echo "• Grafana:              http://localhost:3000"
    echo "• Kibana:               http://localhost:5601"
    echo ""
    echo -e "${BLUE}📝 Logs:${NC}"
    echo "• View Flink logs:      docker-compose logs -f"
    echo "• View specific service: docker-compose logs -f jobmanager"
    echo ""
    echo -e "${YELLOW}💡 Next Steps:${NC}"
    echo "1. Monitor job progress in Flink Web UI"
    echo "2. Check data flow in monitoring dashboards"
    echo "3. Verify data is flowing to all sinks"
    echo "4. Test clinical event processing"
}

# Main execution
main() {
    # Parse command line arguments
    SKIP_BUILD=false
    SKIP_TESTS=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-tests)
                SKIP_TESTS=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skip-build    Skip Maven build step"
                echo "  --skip-tests    Skip connection tests"
                echo "  --help          Show this help message"
                exit 0
                ;;
            *)
                echo -e "${RED}❌ Unknown option: $1${NC}"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Execute startup sequence
    if ! check_shared_datastores; then
        exit 1
    fi

    if [ "$SKIP_TESTS" = false ]; then
        if ! test_connections; then
            exit 1
        fi
    fi

    if [ "$SKIP_BUILD" = false ]; then
        if ! build_application; then
            exit 1
        fi
    fi

    if ! start_flink_cluster; then
        exit 1
    fi

    submit_flink_jobs

    display_access_info
}

# Run main function
main "$@"