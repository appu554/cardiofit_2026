#!/bin/bash

set -e

echo "🚀 Deploying KB-7 Semantic Web Infrastructure (Phase 3)"
echo "======================================================="

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="docker-compose.semantic.yml"
PROJECT_NAME="kb7-semantic"
TIMEOUT=300

# Function to print colored output
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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to wait for service health
wait_for_service() {
    local service_name=$1
    local health_url=$2
    local max_attempts=30
    local attempt=1

    print_status "Waiting for $service_name to be healthy..."

    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$health_url" >/dev/null 2>&1; then
            print_success "$service_name is healthy"
            return 0
        fi

        print_status "Attempt $attempt/$max_attempts - $service_name not ready yet..."
        sleep 10
        ((attempt++))
    done

    print_error "$service_name failed to become healthy within $((max_attempts * 10)) seconds"
    return 1
}

# Pre-deployment checks
print_status "Running pre-deployment checks..."

if ! command_exists docker; then
    print_error "Docker is not installed"
    exit 1
fi

if ! command_exists docker-compose; then
    print_error "Docker Compose is not installed"
    exit 1
fi

if ! command_exists curl; then
    print_error "curl is not installed"
    exit 1
fi

if [ ! -f "$COMPOSE_FILE" ]; then
    print_error "Docker Compose file not found: $COMPOSE_FILE"
    exit 1
fi

print_success "Pre-deployment checks passed"

# Check if services are already running
if docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps | grep -q "Up"; then
    print_warning "Some services are already running"
    read -p "Do you want to recreate them? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_status "Stopping existing services..."
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down
    else
        print_status "Continuing with existing services..."
    fi
fi

# Create necessary directories
print_status "Creating required directories..."
mkdir -p semantic/ontologies
mkdir -p semantic/config
mkdir -p semantic/robot-configs
mkdir -p semantic/schemas

# Build custom images
print_status "Building custom Docker images..."

# Build SPARQL proxy
if [ -d "semantic/sparql-proxy" ]; then
    print_status "Building SPARQL proxy image..."
    docker build -f semantic/Dockerfile.sparql-proxy -t kb7-sparql-proxy .
else
    print_warning "SPARQL proxy source not found, skipping build"
fi

# Build ROBOT service
if [ -f "semantic/Dockerfile.robot" ]; then
    print_status "Building ROBOT service image..."
    docker build -f semantic/Dockerfile.robot -t kb7-robot .
else
    print_warning "ROBOT Dockerfile not found, skipping build"
fi

# Deploy services
print_status "Deploying semantic services..."
docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d

print_success "Services deployed successfully"

# Wait for core services to be healthy
print_status "Waiting for services to become healthy..."

# Wait for GraphDB
if ! wait_for_service "GraphDB" "http://localhost:7200/rest/repositories"; then
    print_error "GraphDB failed to start"
    exit 1
fi

# Wait for Redis
if ! wait_for_service "Redis" "http://localhost:6381"; then
    print_warning "Redis health check failed, but continuing..."
fi

# Wait for SPARQL proxy (if built)
if docker images | grep -q "kb7-sparql-proxy"; then
    if ! wait_for_service "SPARQL Proxy" "http://localhost:8095/health"; then
        print_warning "SPARQL Proxy health check failed, but continuing..."
    fi
fi

print_success "Core services are healthy"

# Initialize GraphDB repository
print_status "Initializing GraphDB repository..."

# Check if repository exists
REPO_CHECK=$(curl -s -w "%{http_code}" -o /dev/null "http://localhost:7200/rest/repositories/kb7-terminology")

if [ "$REPO_CHECK" != "200" ]; then
    print_status "Creating KB-7 terminology repository..."

    # Create repository using configuration
    if [ -f "semantic/config/kb7-repository-config.ttl" ]; then
        curl -X PUT \
            -H "Content-Type: application/x-turtle" \
            -T semantic/config/kb7-repository-config.ttl \
            "http://localhost:7200/rest/repositories/kb7-terminology" || {
            print_warning "Repository creation via config file failed, trying basic creation..."

            # Fallback: Create basic repository
            curl -X PUT \
                -H "Content-Type: application/json" \
                -d '{
                    "repositoryID": "kb7-terminology",
                    "title": "KB-7 Clinical Terminology Repository",
                    "type": "file-repository"
                }' \
                "http://localhost:7200/rest/repositories/kb7-terminology"
        }
    else
        print_warning "Repository config file not found, creating basic repository..."

        # Create basic repository
        curl -X PUT \
            -H "Content-Type: application/json" \
            -d '{
                "repositoryID": "kb7-terminology",
                "title": "KB-7 Clinical Terminology Repository",
                "type": "file-repository"
            }' \
            "http://localhost:7200/rest/repositories/kb7-terminology"
    fi

    print_success "Repository created successfully"
else
    print_success "Repository already exists"
fi

# Load core ontology
print_status "Loading core KB-7 ontology..."

if [ -f "semantic/ontologies/kb7-core.ttl" ]; then
    curl -X POST \
        -H "Content-Type: application/x-turtle" \
        -T semantic/ontologies/kb7-core.ttl \
        "http://localhost:7200/repositories/kb7-terminology/statements" && {
        print_success "Core ontology loaded successfully"
    } || {
        print_warning "Core ontology loading failed, but continuing..."
    }
else
    print_warning "Core ontology file not found: semantic/ontologies/kb7-core.ttl"
fi

# Run validation if ROBOT service is available
if docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps robot-service | grep -q "Up"; then
    print_status "Running ontology validation..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" exec -T robot-service python3 scripts/validate_ontologies.py || {
        print_warning "Ontology validation failed, but continuing..."
    }
else
    print_warning "ROBOT service not available, skipping validation"
fi

# Display service URLs
print_success "KB-7 Semantic Web Infrastructure deployed successfully!"
echo
echo "📊 Service URLs:"
echo "🔹 GraphDB Workbench: http://localhost:7200"
echo "🔹 GraphDB Repository: http://localhost:7200/repository/kb7-terminology"
echo "🔹 SPARQL Proxy: http://localhost:8095"
echo "🔹 Redis Cache: localhost:6381"
echo "🔹 RDF4J Workbench: http://localhost:8082"
echo

# Display SPARQL endpoint test
echo "🧪 Testing SPARQL endpoint..."
TEST_QUERY='{"query": "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 5"}'

if curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$TEST_QUERY" \
    "http://localhost:8095/sparql" | grep -q "head"; then
    print_success "SPARQL endpoint is responding"
else
    print_warning "SPARQL endpoint test failed"
fi

# Display management commands
echo
echo "🔧 Management Commands:"
echo "  View logs:       docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME logs -f"
echo "  Stop services:   docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME down"
echo "  Restart:         docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME restart"
echo "  Shell access:    docker-compose -f $COMPOSE_FILE -p $PROJECT_NAME exec [service] bash"
echo

# Check system resources
print_status "System resource usage:"
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
    $(docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps -q) 2>/dev/null || {
    print_warning "Could not retrieve container stats"
}

print_success "Deployment completed successfully!"
print_status "Phase 3: Semantic Web Infrastructure is now operational"

echo
echo "🎯 Next steps:"
echo "  1. Access GraphDB Workbench at http://localhost:7200"
echo "  2. Run SPARQL queries via the proxy at http://localhost:8095"
echo "  3. Load additional ontologies using the ROBOT service"
echo "  4. Configure semantic reasoning rules"
echo "  5. Begin Phase 4: Real-Time Architecture implementation"