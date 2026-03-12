#!/bin/bash

# CardioFit Kafka Infrastructure Shutdown Script

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "==========================================="
echo "CardioFit Kafka Infrastructure Shutdown"
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

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Ask for confirmation
read -p "Are you sure you want to stop the Kafka infrastructure? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_warning "Shutdown cancelled"
    exit 0
fi

# Stop containers
print_status "Stopping Kafka containers..."
$DOCKER_COMPOSE stop

# Ask if volumes should be removed
read -p "Do you want to remove data volumes? This will delete all Kafka data! (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_warning "Removing containers and volumes..."
    $DOCKER_COMPOSE down -v
    print_status "All data has been removed"
else
    print_status "Containers stopped, data preserved"
fi

# Save shutdown log
mkdir -p scripts/logs
echo "$(date): Kafka infrastructure stopped" >> scripts/logs/shutdown.log

echo ""
echo "==========================================="
echo "Kafka Infrastructure Stopped"
echo "==========================================="
echo ""
echo "To restart: ./start-kafka.sh"
echo ""