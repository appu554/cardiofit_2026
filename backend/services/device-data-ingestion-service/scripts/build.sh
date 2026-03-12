#!/bin/bash
# Build script for Device Data Ingestion Service with Outbox Pattern
# Creates optimized Docker images for production deployment

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
IMAGE_REGISTRY="${IMAGE_REGISTRY:-clinical-synthesis-hub}"
VERSION="${VERSION:-$(git rev-parse --short HEAD)}"
BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
VCS_REF="$(git rev-parse --short HEAD)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check Git
    if ! command -v git &> /dev/null; then
        log_error "Git is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Function to build ingestion service image
build_ingestion_service() {
    log_info "Building Device Data Ingestion Service image..."
    
    local image_name="${IMAGE_REGISTRY}/device-ingestion:${VERSION}"
    local latest_tag="${IMAGE_REGISTRY}/device-ingestion:latest"
    
    docker build \
        --file "${PROJECT_DIR}/Dockerfile" \
        --tag "${image_name}" \
        --tag "${latest_tag}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg VCS_REF="${VCS_REF}" \
        --target runtime \
        "${PROJECT_DIR}"
    
    log_success "Ingestion service image built: ${image_name}"
}

# Function to build publisher service image
build_publisher_service() {
    log_info "Building Outbox Publisher Service image..."
    
    local image_name="${IMAGE_REGISTRY}/outbox-publisher:${VERSION}"
    local latest_tag="${IMAGE_REGISTRY}/outbox-publisher:latest"
    
    docker build \
        --file "${PROJECT_DIR}/Dockerfile.publisher" \
        --tag "${image_name}" \
        --tag "${latest_tag}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg VCS_REF="${VCS_REF}" \
        --target runtime \
        "${PROJECT_DIR}"
    
    log_success "Publisher service image built: ${image_name}"
}

# Function to run security scan
security_scan() {
    log_info "Running security scan..."
    
    # Check if trivy is available
    if command -v trivy &> /dev/null; then
        log_info "Scanning ingestion service image..."
        trivy image "${IMAGE_REGISTRY}/device-ingestion:${VERSION}" || log_warning "Security scan found issues"
        
        log_info "Scanning publisher service image..."
        trivy image "${IMAGE_REGISTRY}/outbox-publisher:${VERSION}" || log_warning "Security scan found issues"
    else
        log_warning "Trivy not found, skipping security scan"
    fi
}

# Function to test images
test_images() {
    log_info "Testing built images..."
    
    # Test ingestion service image
    log_info "Testing ingestion service image..."
    docker run --rm "${IMAGE_REGISTRY}/device-ingestion:${VERSION}" python -c "
import sys
sys.path.append('/app')
from app.db.database import db_manager
from app.services.outbox_service import VendorAwareOutboxService
from app.core.monitoring import metrics_collector
print('✅ All imports successful')
"
    
    # Test publisher service image
    log_info "Testing publisher service image..."
    docker run --rm "${IMAGE_REGISTRY}/outbox-publisher:${VERSION}" python -c "
import sys
sys.path.append('/app')
from app.services.outbox_publisher import outbox_publisher
from app.services.outbox_service import VendorAwareOutboxService
print('✅ All imports successful')
"
    
    log_success "Image tests passed"
}

# Function to show image information
show_image_info() {
    log_info "Image information:"
    
    echo "Ingestion Service:"
    docker images "${IMAGE_REGISTRY}/device-ingestion:${VERSION}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
    
    echo "Publisher Service:"
    docker images "${IMAGE_REGISTRY}/outbox-publisher:${VERSION}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
}

# Function to push images
push_images() {
    if [[ "${PUSH_IMAGES:-false}" == "true" ]]; then
        log_info "Pushing images to registry..."
        
        docker push "${IMAGE_REGISTRY}/device-ingestion:${VERSION}"
        docker push "${IMAGE_REGISTRY}/device-ingestion:latest"
        docker push "${IMAGE_REGISTRY}/outbox-publisher:${VERSION}"
        docker push "${IMAGE_REGISTRY}/outbox-publisher:latest"
        
        log_success "Images pushed to registry"
    else
        log_info "Skipping image push (set PUSH_IMAGES=true to enable)"
    fi
}

# Function to cleanup
cleanup() {
    log_info "Cleaning up build artifacts..."
    
    # Remove dangling images
    docker image prune -f
    
    log_success "Cleanup completed"
}

# Main execution
main() {
    log_info "Starting build process for Device Data Ingestion Service"
    log_info "Version: ${VERSION}"
    log_info "Build Date: ${BUILD_DATE}"
    log_info "VCS Ref: ${VCS_REF}"
    log_info "Registry: ${IMAGE_REGISTRY}"
    
    cd "${PROJECT_DIR}"
    
    check_prerequisites
    build_ingestion_service
    build_publisher_service
    
    if [[ "${SKIP_TESTS:-false}" != "true" ]]; then
        test_images
    fi
    
    if [[ "${SECURITY_SCAN:-false}" == "true" ]]; then
        security_scan
    fi
    
    show_image_info
    push_images
    
    if [[ "${CLEANUP:-true}" == "true" ]]; then
        cleanup
    fi
    
    log_success "Build process completed successfully!"
    log_info "Images ready for deployment:"
    log_info "  - ${IMAGE_REGISTRY}/device-ingestion:${VERSION}"
    log_info "  - ${IMAGE_REGISTRY}/outbox-publisher:${VERSION}"
}

# Handle script arguments
case "${1:-build}" in
    "build")
        main
        ;;
    "test")
        test_images
        ;;
    "push")
        PUSH_IMAGES=true main
        ;;
    "scan")
        SECURITY_SCAN=true main
        ;;
    "help")
        echo "Usage: $0 [build|test|push|scan|help]"
        echo "  build - Build Docker images (default)"
        echo "  test  - Test built images"
        echo "  push  - Build and push images"
        echo "  scan  - Build with security scan"
        echo "  help  - Show this help"
        echo ""
        echo "Environment variables:"
        echo "  IMAGE_REGISTRY - Docker registry (default: clinical-synthesis-hub)"
        echo "  VERSION - Image version (default: git short hash)"
        echo "  PUSH_IMAGES - Push images after build (default: false)"
        echo "  SKIP_TESTS - Skip image testing (default: false)"
        echo "  SECURITY_SCAN - Run security scan (default: false)"
        echo "  CLEANUP - Cleanup after build (default: true)"
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
