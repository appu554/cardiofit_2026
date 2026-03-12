#!/bin/bash

# KB-7 Semantic Container Management Script

set -e

echo "🐳 KB-7 Semantic Infrastructure Container"
echo "========================================"

# Configuration
CONTAINER_NAME="kb7-semantic"
IMAGE_NAME="kb7-semantic:latest"
GRAPHDB_PORT="7200"
SPARQL_PORT="8095"
REDIS_PORT="6379"

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
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

# Function to check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        echo "💡 Try: open -a Docker"
        exit 1
    fi
    print_success "Docker is running"
}

# Function to stop existing container
stop_container() {
    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        print_info "Stopping existing KB-7 container..."
        docker stop "$CONTAINER_NAME" >/dev/null 2>&1
        print_success "Container stopped"
    fi

    if docker ps -aq -f name="$CONTAINER_NAME" | grep -q .; then
        print_info "Removing existing KB-7 container..."
        docker rm "$CONTAINER_NAME" >/dev/null 2>&1
        print_success "Container removed"
    fi
}

# Function to build the image
build_image() {
    print_info "Building KB-7 semantic image..."

    # Check if Dockerfile exists
    if [ ! -f "Dockerfile.kb7-semantic" ]; then
        print_error "Dockerfile.kb7-semantic not found"
        exit 1
    fi

    # Build the image
    docker build -f Dockerfile.kb7-semantic -t "$IMAGE_NAME" . || {
        print_error "Failed to build image"
        exit 1
    }

    print_success "Image built successfully: $IMAGE_NAME"
}

# Function to run the container
run_container() {
    print_info "Starting KB-7 semantic container..."

    docker run -d \
        --name "$CONTAINER_NAME" \
        -p "$GRAPHDB_PORT:7200" \
        -p "$SPARQL_PORT:8095" \
        -p "$REDIS_PORT:6379" \
        -v "$(pwd)/semantic/ontologies:/app/ontologies" \
        -v "$(pwd)/semantic/config:/app/config" \
        --health-cmd="curl -f http://localhost:7200/rest/repositories || exit 1" \
        --health-interval=30s \
        --health-timeout=10s \
        --health-retries=3 \
        --health-start-period=60s \
        "$IMAGE_NAME" || {
        print_error "Failed to start container"
        exit 1
    }

    print_success "Container started: $CONTAINER_NAME"
}

# Function to wait for services
wait_for_services() {
    print_info "Waiting for services to be ready..."

    local max_attempts=20
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "http://localhost:$GRAPHDB_PORT/rest/repositories" >/dev/null 2>&1; then
            print_success "GraphDB is ready"
            break
        fi

        print_info "Attempt $attempt/$max_attempts - Services starting..."
        sleep 15
        ((attempt++))
    done

    if [ $attempt -gt $max_attempts ]; then
        print_error "Services failed to start within 5 minutes"
        print_info "Check logs: docker logs $CONTAINER_NAME"
        exit 1
    fi
}

# Function to test services
test_services() {
    print_info "Testing KB-7 services..."

    # Test GraphDB
    if curl -f -s "http://localhost:$GRAPHDB_PORT/rest/repositories" >/dev/null; then
        print_success "✅ GraphDB is responding"
    else
        print_error "❌ GraphDB is not responding"
    fi

    # Test SPARQL Proxy (if built)
    if curl -f -s "http://localhost:$SPARQL_PORT/health" >/dev/null 2>&1; then
        print_success "✅ SPARQL Proxy is responding"
    else
        print_warning "⚠️ SPARQL Proxy not responding (may not be built)"
    fi

    # Test Redis
    if redis-cli -p "$REDIS_PORT" ping >/dev/null 2>&1; then
        print_success "✅ Redis is responding"
    else
        print_warning "⚠️ Redis not responding"
    fi

    # Test SPARQL endpoint
    TEST_QUERY='SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 1'
    if curl -s -X POST \
        -H "Content-Type: application/x-www-form-urlencoded" \
        --data-urlencode "query=$TEST_QUERY" \
        "http://localhost:$GRAPHDB_PORT/repositories/kb7-terminology" | grep -q "head"; then
        print_success "✅ SPARQL endpoint is working"
    else
        print_warning "⚠️ SPARQL endpoint test failed (repository may not be created yet)"
    fi
}

# Function to show status
show_status() {
    echo ""
    echo "📊 KB-7 Semantic Infrastructure Status"
    echo "====================================="

    if docker ps -q -f name="$CONTAINER_NAME" | grep -q .; then
        print_success "Container Status: RUNNING"

        echo ""
        echo "🌐 Service URLs:"
        echo "  • GraphDB Workbench: http://localhost:$GRAPHDB_PORT"
        echo "  • SPARQL Proxy: http://localhost:$SPARQL_PORT"
        echo "  • KB-7 Repository: http://localhost:$GRAPHDB_PORT/repository/kb7-terminology"

        echo ""
        echo "🔧 Management Commands:"
        echo "  • View logs: docker logs $CONTAINER_NAME"
        echo "  • Stop: docker stop $CONTAINER_NAME"
        echo "  • Shell: docker exec -it $CONTAINER_NAME bash"

        echo ""
        test_services
    else
        print_error "Container Status: NOT RUNNING"
    fi
}

# Main execution
case "${1:-run}" in
    "build")
        check_docker
        build_image
        ;;
    "run")
        check_docker
        stop_container
        build_image
        run_container
        wait_for_services
        show_status
        ;;
    "stop")
        stop_container
        ;;
    "status")
        show_status
        ;;
    "logs")
        docker logs -f "$CONTAINER_NAME"
        ;;
    "shell")
        docker exec -it "$CONTAINER_NAME" bash
        ;;
    *)
        echo "Usage: $0 [build|run|stop|status|logs|shell]"
        echo ""
        echo "Commands:"
        echo "  build  - Build the KB-7 image only"
        echo "  run    - Build and run KB-7 container (default)"
        echo "  stop   - Stop and remove KB-7 container"
        echo "  status - Show KB-7 container status"
        echo "  logs   - Show KB-7 container logs"
        echo "  shell  - Open shell in KB-7 container"
        exit 1
        ;;
esac