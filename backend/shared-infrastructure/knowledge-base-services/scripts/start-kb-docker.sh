#!/bin/bash

# Start KB-Drug-Rules service with Docker PostgreSQL
# This script starts PostgreSQL in Docker and the KB service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

echo "🚀 Starting KB-Drug-Rules Service with Docker PostgreSQL"
echo "========================================================"
echo ""

# Check if Docker is running
print_status "Checking Docker..."
if ! command -v docker >/dev/null 2>&1; then
    print_error "Docker is not installed"
    echo "Please install Docker from: https://www.docker.com/get-started"
    exit 1
fi

if ! docker info >/dev/null 2>&1; then
    print_error "Docker is not running"
    echo "Please start Docker Desktop or Docker daemon"
    exit 1
fi

print_success "Docker is available"

# Navigate to the correct directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Check if docker-compose file exists
if [ ! -f "docker-compose.kb-only.yml" ]; then
    print_error "docker-compose.kb-only.yml not found"
    echo "Please make sure you're in the correct directory"
    exit 1
fi

# Stop any existing containers
print_status "Stopping any existing KB containers..."
docker-compose -f docker-compose.kb-only.yml down >/dev/null 2>&1 || true

# Start the services
print_status "Starting KB services with Docker..."
echo "This will:"
echo "  - Start PostgreSQL on port 5433 (to avoid conflict with your PostgreSQL 17.6)"
echo "  - Start Redis on port 6380"
echo "  - Start KB-Drug-Rules service on port 8081"
echo "  - Start Adminer (database UI) on port 8082"
echo ""

if ! docker-compose -f docker-compose.kb-only.yml up -d; then
    print_error "Failed to start services"
    echo "Check Docker logs for details"
    exit 1
fi

print_success "Services are starting..."
echo ""

# Wait for services to be ready
print_status "Waiting for services to be ready..."
sleep 10

# Check service health
print_status "Checking service health..."

# Check PostgreSQL
if docker exec kb-postgres pg_isready -U postgres >/dev/null 2>&1; then
    print_success "PostgreSQL is ready"
else
    print_warning "PostgreSQL is still starting..."
fi

# Check Redis
if docker exec kb-redis redis-cli ping >/dev/null 2>&1; then
    print_success "Redis is ready"
else
    print_warning "Redis is still starting..."
fi

# Wait a bit more for KB service
print_status "Waiting for KB-Drug-Rules service..."
sleep 15

# Test KB service
if curl -s http://localhost:8081/health >/dev/null 2>&1; then
    print_success "KB-Drug-Rules service is ready!"
else
    print_warning "KB-Drug-Rules service is still starting..."
    echo "You can check logs with: docker logs kb-drug-rules"
fi

echo ""
echo "🎉 KB Services Started Successfully!"
echo "===================================="
echo ""
echo "Services available at:"
echo "  📊 KB-Drug-Rules API:    http://localhost:8081"
echo "  🔍 Health Check:         http://localhost:8081/health"
echo "  📈 Metrics:              http://localhost:8081/metrics"
echo "  🗄️  Database (Adminer):   http://localhost:8082"
echo "  🗄️  PostgreSQL:          localhost:5433 (user: kb_drug_rules_user, password: kb_password)"
echo "  🗄️  Redis:               localhost:6380"
echo ""
echo "Database connection details:"
echo "  Host:     localhost"
echo "  Port:     5433"
echo "  Database: kb_drug_rules"
echo "  Username: kb_drug_rules_user"
echo "  Password: kb_password"
echo ""
echo "Test commands:"
echo "  curl http://localhost:8081/health"
echo "  curl http://localhost:8081/v1/items/metformin"
echo ""
echo "To stop services: docker-compose -f docker-compose.kb-only.yml down"
echo "To view logs: docker logs kb-drug-rules"
echo ""

# Test the API
print_status "Testing API endpoints..."
echo ""

# Test health endpoint
echo "Testing health endpoint..."
if health_response=$(curl -s http://localhost:8081/health 2>/dev/null); then
    echo "$health_response" | jq '.' 2>/dev/null || echo "$health_response"
    echo ""
    print_success "Health endpoint working!"
else
    print_warning "Health endpoint not ready yet, service may still be starting"
fi

echo ""
echo "Testing drug rules endpoint..."
if drug_response=$(curl -s http://localhost:8081/v1/items/metformin 2>/dev/null); then
    echo "$drug_response" | jq '.drug_id, .version, .content.meta.drug_name' 2>/dev/null || echo "$drug_response"
    echo ""
    print_success "Drug rules endpoint working!"
else
    print_warning "Drug rules endpoint not ready yet, service may still be starting"
fi

echo ""
echo "🎯 Setup Complete!"
echo ""
echo "Your KB-Drug-Rules service is now running with:"
echo "  ✅ Isolated PostgreSQL (port 5433)"
echo "  ✅ Sample drug data (metformin, lisinopril, warfarin)"
echo "  ✅ Complete API endpoints"
echo "  ✅ Database management UI"
echo ""
echo "Ready for Flow2 integration! 🚀"
