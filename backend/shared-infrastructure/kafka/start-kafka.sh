#!/bin/bash

# CardioFit Kafka Infrastructure Startup Script
# Starts the complete Kafka cluster with all 68 topics

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "==========================================="
echo "CardioFit Kafka Infrastructure Startup"
echo "==========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker first."
    exit 1
fi

print_status "Docker is running"

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    print_warning "docker-compose not found, trying docker compose..."
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Load environment variables
if [ -f .env ]; then
    print_status "Loading environment variables from .env"
    export $(cat .env | grep -v '^#' | xargs)
else
    print_warning ".env file not found, using defaults"
fi

# Create necessary directories
print_status "Creating necessary directories..."
mkdir -p data logs connectors scripts/logs

# Make scripts executable
chmod +x scripts/*.sh 2>/dev/null || true

# Stop any existing containers
print_status "Stopping any existing Kafka containers..."
$DOCKER_COMPOSE down -v 2>/dev/null || true

# Remove old volumes (optional, comment out to preserve data)
print_warning "Cleaning up old volumes..."
docker volume prune -f 2>/dev/null || true

# Pre-pull images to avoid timeout issues
print_status "Pre-pulling Docker images (this may take several minutes)..."
$DOCKER_COMPOSE pull

# Start infrastructure
print_status "Starting Kafka infrastructure..."
DOCKER_CLIENT_TIMEOUT=600 COMPOSE_HTTP_TIMEOUT=600 $DOCKER_COMPOSE up -d

# Wait for services to be healthy
print_status "Waiting for services to be healthy..."

# Function to check service health
check_service_health() {
    local service=$1
    local max_attempts=30
    local attempt=0

    while [ $attempt -lt $max_attempts ]; do
        if docker-compose ps | grep $service | grep -q "healthy\|running"; then
            return 0
        fi
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    return 1
}

# Check Zookeeper
echo -n "Waiting for Zookeeper"
if check_service_health "zookeeper"; then
    echo ""
    print_status "Zookeeper is ready"
else
    echo ""
    print_error "Zookeeper failed to start"
    exit 1
fi

# Check Kafka brokers
for broker in kafka1 kafka2 kafka3; do
    echo -n "Waiting for $broker"
    if check_service_health "$broker"; then
        echo ""
        print_status "$broker is ready"
    else
        echo ""
        print_error "$broker failed to start"
        exit 1
    fi
done

# Check Schema Registry
echo -n "Waiting for Schema Registry"
if check_service_health "schema-registry"; then
    echo ""
    print_status "Schema Registry is ready"
else
    echo ""
    print_warning "Schema Registry is still starting (non-critical)"
fi

# Wait a bit more for cluster formation
print_status "Waiting for Kafka cluster to stabilize..."
sleep 10

# Create topics
print_status "Creating Kafka topics..."
docker exec cardiofit-kafka1 bash /usr/bin/create-topics.sh

# Verify topic creation
print_status "Verifying topic creation..."
TOPIC_COUNT=$(docker exec cardiofit-kafka1 kafka-topics --bootstrap-server kafka1:29092 --list | grep -E '\.(v1|changes)$' | wc -l)
print_status "Created $TOPIC_COUNT topics"

# Display service URLs
echo ""
echo "==========================================="
echo "Kafka Infrastructure is Ready!"
echo "==========================================="
echo ""
echo "Service URLs:"
echo "  📊 Kafka UI: http://localhost:8080"
echo "  📈 Kafdrop: http://localhost:9000"
echo "  🔧 Schema Registry: http://localhost:8081"
echo "  💾 KSQL DB: http://localhost:8088"
echo "  🔌 Kafka Connect: http://localhost:8083"
echo ""
echo "Kafka Brokers:"
echo "  • Broker 1: localhost:9092"
echo "  • Broker 2: localhost:9093"
echo "  • Broker 3: localhost:9094"
echo ""
echo "Management Commands:"
echo "  • View logs: docker-compose logs -f [service]"
echo "  • Stop cluster: ./stop-kafka.sh"
echo "  • Health check: ./health-check.sh"
echo "  • Topic management: ./manage-topics.sh"
echo ""
echo "==========================================="

# Save startup log
echo "$(date): Kafka infrastructure started successfully" >> scripts/logs/startup.log