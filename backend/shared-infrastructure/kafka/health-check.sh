#!/bin/bash

# CardioFit Kafka Health Check Script
# Validates the health of all Kafka infrastructure components

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "==========================================="
echo "CardioFit Kafka Infrastructure Health Check"
echo "==========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Initialize health status
HEALTH_STATUS="HEALTHY"
ISSUES=()

echo ""
echo "1. CHECKING CONTAINER STATUS"
echo "----------------------------"

# Check if containers are running
check_container() {
    local container=$1
    local service=$2

    if docker ps --format '{{.Names}}' | grep -q "^${container}$"; then
        # Get container status
        STATUS=$(docker inspect -f '{{.State.Status}}' $container 2>/dev/null || echo "unknown")

        if [ "$STATUS" == "running" ]; then
            # Check if container is healthy (if health check exists)
            HEALTH=$(docker inspect -f '{{.State.Health.Status}}' $container 2>/dev/null || echo "no-healthcheck")

            if [ "$HEALTH" == "healthy" ] || [ "$HEALTH" == "no-healthcheck" ]; then
                print_status "$service is running"
            else
                print_warning "$service is $HEALTH"
                if [ "$HEALTH" == "unhealthy" ]; then
                    HEALTH_STATUS="DEGRADED"
                    ISSUES+=("$service is unhealthy")
                fi
            fi
        else
            print_error "$service is $STATUS"
            HEALTH_STATUS="CRITICAL"
            ISSUES+=("$service is not running")
        fi
    else
        print_error "$service container not found"
        HEALTH_STATUS="CRITICAL"
        ISSUES+=("$service container not found")
    fi
}

# Check all services
check_container "cardiofit-zookeeper" "Zookeeper"
check_container "cardiofit-kafka1" "Kafka Broker 1"
check_container "cardiofit-kafka2" "Kafka Broker 2"
check_container "cardiofit-kafka3" "Kafka Broker 3"
check_container "cardiofit-schema-registry" "Schema Registry"
check_container "cardiofit-kafka-connect" "Kafka Connect"
check_container "cardiofit-kafka-ui" "Kafka UI"
check_container "cardiofit-ksqldb-server" "KSQL DB"
check_container "cardiofit-kafdrop" "Kafdrop"

echo ""
echo "2. CHECKING KAFKA CLUSTER"
echo "-------------------------"

# Check Kafka cluster status
if docker exec cardiofit-kafka1 kafka-broker-api-versions --bootstrap-server kafka1:29092 &>/dev/null; then
    print_status "Kafka cluster is responsive"

    # Get broker count
    BROKER_COUNT=$(docker exec cardiofit-kafka1 kafka-metadata-shell --snapshot /var/kafka-logs/__cluster_metadata-0/00000000000000000000.log --print-brokers 2>/dev/null | grep -c "BrokerId" || echo "0")

    if [ "$BROKER_COUNT" -eq "3" ]; then
        print_status "All 3 brokers are registered"
    else
        print_warning "Only $BROKER_COUNT brokers registered (expected 3)"
        HEALTH_STATUS="DEGRADED"
        ISSUES+=("Broker count mismatch")
    fi
else
    print_error "Kafka cluster is not responding"
    HEALTH_STATUS="CRITICAL"
    ISSUES+=("Kafka cluster not responding")
fi

echo ""
echo "3. CHECKING TOPICS"
echo "------------------"

# Check topic count
if docker exec cardiofit-kafka1 kafka-topics --bootstrap-server kafka1:29092 --list &>/dev/null; then
    TOPIC_COUNT=$(docker exec cardiofit-kafka1 kafka-topics --bootstrap-server kafka1:29092 --list | grep -E '\.(v1|changes)$' | wc -l)

    if [ "$TOPIC_COUNT" -eq "68" ]; then
        print_status "All 68 topics exist"
    else
        print_warning "Found $TOPIC_COUNT topics (expected 68)"
        HEALTH_STATUS="DEGRADED"
        ISSUES+=("Topic count mismatch")
    fi

    # Check for under-replicated partitions
    UNDER_REPLICATED=$(docker exec cardiofit-kafka1 kafka-topics --bootstrap-server kafka1:29092 --describe --under-replicated-partitions 2>/dev/null | wc -l)

    if [ "$UNDER_REPLICATED" -eq "0" ]; then
        print_status "No under-replicated partitions"
    else
        print_warning "$UNDER_REPLICATED under-replicated partitions found"
        HEALTH_STATUS="DEGRADED"
        ISSUES+=("Under-replicated partitions")
    fi
else
    print_error "Cannot list topics"
    HEALTH_STATUS="CRITICAL"
    ISSUES+=("Cannot access topics")
fi

echo ""
echo "4. CHECKING CONSUMER GROUPS"
echo "---------------------------"

# List consumer groups
if docker exec cardiofit-kafka1 kafka-consumer-groups --bootstrap-server kafka1:29092 --list &>/dev/null; then
    GROUP_COUNT=$(docker exec cardiofit-kafka1 kafka-consumer-groups --bootstrap-server kafka1:29092 --list | wc -l)
    print_info "Found $GROUP_COUNT consumer groups"

    # Check for lag in consumer groups (if any exist)
    if [ "$GROUP_COUNT" -gt "0" ]; then
        HIGH_LAG_GROUPS=$(docker exec cardiofit-kafka1 kafka-consumer-groups --bootstrap-server kafka1:29092 --all-groups --describe 2>/dev/null | awk '$5 > 1000 {print $1}' | sort -u | wc -l || echo "0")

        if [ "$HIGH_LAG_GROUPS" -eq "0" ]; then
            print_status "No consumer groups with high lag"
        else
            print_warning "$HIGH_LAG_GROUPS consumer groups with lag > 1000"
            HEALTH_STATUS="DEGRADED"
            ISSUES+=("Consumer lag detected")
        fi
    fi
else
    print_warning "Cannot check consumer groups"
fi

echo ""
echo "5. CHECKING SERVICE ENDPOINTS"
echo "-----------------------------"

# Check HTTP endpoints
check_endpoint() {
    local url=$1
    local service=$2

    if curl -s -o /dev/null -w "%{http_code}" "$url" | grep -q "200\|302"; then
        print_status "$service endpoint is accessible"
    else
        print_warning "$service endpoint is not accessible"
        ISSUES+=("$service endpoint not accessible")
    fi
}

check_endpoint "http://localhost:8080/actuator/health" "Kafka UI"
check_endpoint "http://localhost:9000/" "Kafdrop"
check_endpoint "http://localhost:8081/subjects" "Schema Registry"
check_endpoint "http://localhost:8083/" "Kafka Connect"
check_endpoint "http://localhost:8088/info" "KSQL DB"

echo ""
echo "6. RESOURCE USAGE"
echo "-----------------"

# Check Docker resource usage
print_info "Container resource usage:"
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep cardiofit || true

echo ""
echo "==========================================="
echo "HEALTH CHECK SUMMARY"
echo "==========================================="

if [ "$HEALTH_STATUS" == "HEALTHY" ]; then
    print_status "Overall Status: HEALTHY ✅"
    echo "All components are running correctly"
elif [ "$HEALTH_STATUS" == "DEGRADED" ]; then
    print_warning "Overall Status: DEGRADED ⚠️"
    echo "System is operational but with issues:"
    for issue in "${ISSUES[@]}"; do
        echo "  • $issue"
    done
else
    print_error "Overall Status: CRITICAL ❌"
    echo "System has critical issues:"
    for issue in "${ISSUES[@]}"; do
        echo "  • $issue"
    done
fi

echo ""
echo "==========================================="

# Save health check log
mkdir -p scripts/logs
echo "$(date): Health check completed - Status: $HEALTH_STATUS" >> scripts/logs/health-check.log

# Exit with appropriate code
if [ "$HEALTH_STATUS" == "HEALTHY" ]; then
    exit 0
elif [ "$HEALTH_STATUS" == "DEGRADED" ]; then
    exit 1
else
    exit 2
fi