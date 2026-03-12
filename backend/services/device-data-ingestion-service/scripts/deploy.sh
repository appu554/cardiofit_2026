#!/bin/bash
# Deployment script for Device Data Ingestion Service with Outbox Pattern
# Supports Docker Compose and Kubernetes deployments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DEPLOYMENT_TYPE="${DEPLOYMENT_TYPE:-docker-compose}"
ENVIRONMENT="${ENVIRONMENT:-development}"
VERSION="${VERSION:-latest}"

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
    log_info "Checking prerequisites for ${DEPLOYMENT_TYPE} deployment..."
    
    case "${DEPLOYMENT_TYPE}" in
        "docker-compose")
            if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
                log_error "Docker Compose is not installed"
                exit 1
            fi
            ;;
        "kubernetes")
            if ! command -v kubectl &> /dev/null; then
                log_error "kubectl is not installed"
                exit 1
            fi
            if ! kubectl cluster-info &> /dev/null; then
                log_error "Cannot connect to Kubernetes cluster"
                exit 1
            fi
            ;;
        *)
            log_error "Unknown deployment type: ${DEPLOYMENT_TYPE}"
            exit 1
            ;;
    esac
    
    log_success "Prerequisites check passed"
}

# Function to setup environment
setup_environment() {
    log_info "Setting up environment for ${ENVIRONMENT}..."
    
    # Create environment file if it doesn't exist
    local env_file="${PROJECT_DIR}/.env.${ENVIRONMENT}"
    if [[ ! -f "${env_file}" ]]; then
        log_info "Creating environment file: ${env_file}"
        cat > "${env_file}" << EOF
# Environment: ${ENVIRONMENT}
# Generated on: $(date)

# Service Configuration
VERSION=${VERSION}
ENVIRONMENT=${ENVIRONMENT}
DEBUG=false

# Database Configuration
DATABASE_URL=postgresql://postgres:Cardiofit@123@db.auugxeqzgrnknklgwqrh.supabase.co:5432/postgres

# Kafka Configuration
KAFKA_BOOTSTRAP_SERVERS=pkc-619z3.us-east1.gcp.confluent.cloud:9092
KAFKA_API_KEY=LGJ3AQ2L6VRPW4S2
KAFKA_API_SECRET=your-kafka-secret
KAFKA_TOPIC_DEVICE_DATA=raw-device-data.v1

# Outbox Configuration
OUTBOX_BATCH_SIZE=50
OUTBOX_POLL_INTERVAL=5
MAX_CONCURRENT_VENDORS=10
OUTBOX_RETRY_BACKOFF_SECONDS=60

# Monitoring Configuration
GCP_PROJECT_ID=cardiofit-905a8
ENABLE_CLOUD_METRICS=true

# Security
GRAFANA_PASSWORD=admin123
EOF
        log_warning "Please update the environment file with your actual configuration"
    fi
    
    # Source the environment file
    if [[ -f "${env_file}" ]]; then
        set -a
        source "${env_file}"
        set +a
        log_success "Environment loaded from ${env_file}"
    fi
}

# Function to run database migration
run_migration() {
    log_info "Running database migration..."
    
    case "${DEPLOYMENT_TYPE}" in
        "docker-compose")
            docker-compose -f "${PROJECT_DIR}/docker-compose.yml" run --rm ingestion-service python run_migration.py
            ;;
        "kubernetes")
            kubectl run migration-job --image="clinical-synthesis-hub/device-ingestion:${VERSION}" --restart=Never --command -- python run_migration.py
            kubectl wait --for=condition=complete job/migration-job --timeout=300s
            kubectl delete job migration-job
            ;;
    esac
    
    log_success "Database migration completed"
}

# Function to deploy with Docker Compose
deploy_docker_compose() {
    log_info "Deploying with Docker Compose..."
    
    cd "${PROJECT_DIR}"
    
    # Pull latest images
    docker-compose pull
    
    # Start services
    docker-compose up -d
    
    # Wait for services to be healthy
    log_info "Waiting for services to be healthy..."
    sleep 30
    
    # Check service health
    if docker-compose ps | grep -q "Up (healthy)"; then
        log_success "Services are healthy"
    else
        log_warning "Some services may not be healthy, check logs"
        docker-compose ps
    fi
    
    log_success "Docker Compose deployment completed"
}

# Function to deploy to Kubernetes
deploy_kubernetes() {
    log_info "Deploying to Kubernetes..."
    
    local k8s_dir="${PROJECT_DIR}/k8s"
    
    if [[ ! -d "${k8s_dir}" ]]; then
        log_error "Kubernetes manifests directory not found: ${k8s_dir}"
        exit 1
    fi
    
    # Apply namespace
    kubectl apply -f "${k8s_dir}/namespace.yaml"
    
    # Apply configmaps and secrets
    kubectl apply -f "${k8s_dir}/configmap.yaml"
    kubectl apply -f "${k8s_dir}/secret.yaml"
    
    # Apply services
    kubectl apply -f "${k8s_dir}/service.yaml"
    
    # Apply deployments
    kubectl apply -f "${k8s_dir}/deployment.yaml"
    
    # Wait for deployments to be ready
    kubectl wait --for=condition=available deployment/device-ingestion-service --timeout=300s
    kubectl wait --for=condition=available deployment/outbox-publisher-service --timeout=300s
    
    log_success "Kubernetes deployment completed"
}

# Function to verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    case "${DEPLOYMENT_TYPE}" in
        "docker-compose")
            # Check service health endpoints
            local ingestion_url="http://localhost:8015"
            
            if curl -f "${ingestion_url}/api/v1/health" > /dev/null 2>&1; then
                log_success "Ingestion service is responding"
            else
                log_error "Ingestion service health check failed"
                return 1
            fi
            
            if curl -f "${ingestion_url}/api/v1/outbox/health" > /dev/null 2>&1; then
                log_success "Outbox system is healthy"
            else
                log_error "Outbox health check failed"
                return 1
            fi
            ;;
        "kubernetes")
            # Check pod status
            kubectl get pods -l app=device-ingestion-service
            kubectl get pods -l app=outbox-publisher-service
            
            # Check service endpoints
            local service_ip=$(kubectl get service device-ingestion-service -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
            if [[ -n "${service_ip}" ]]; then
                if curl -f "http://${service_ip}/api/v1/health" > /dev/null 2>&1; then
                    log_success "Kubernetes service is responding"
                else
                    log_warning "Service health check failed, but pods may still be starting"
                fi
            fi
            ;;
    esac
    
    log_success "Deployment verification completed"
}

# Function to show deployment status
show_status() {
    log_info "Deployment status:"
    
    case "${DEPLOYMENT_TYPE}" in
        "docker-compose")
            docker-compose ps
            echo ""
            log_info "Service URLs:"
            echo "  - Ingestion Service: http://localhost:8015"
            echo "  - Grafana Dashboard: http://localhost:3000"
            echo "  - Prometheus: http://localhost:9090"
            ;;
        "kubernetes")
            kubectl get all -l component=device-ingestion
            echo ""
            log_info "To access services:"
            echo "  kubectl port-forward service/device-ingestion-service 8015:8015"
            ;;
    esac
}

# Function to cleanup deployment
cleanup() {
    log_info "Cleaning up deployment..."
    
    case "${DEPLOYMENT_TYPE}" in
        "docker-compose")
            docker-compose down -v
            docker system prune -f
            ;;
        "kubernetes")
            kubectl delete -f "${PROJECT_DIR}/k8s/"
            ;;
    esac
    
    log_success "Cleanup completed"
}

# Main execution
main() {
    log_info "Starting deployment process"
    log_info "Deployment Type: ${DEPLOYMENT_TYPE}"
    log_info "Environment: ${ENVIRONMENT}"
    log_info "Version: ${VERSION}"
    
    check_prerequisites
    setup_environment
    
    case "${DEPLOYMENT_TYPE}" in
        "docker-compose")
            deploy_docker_compose
            ;;
        "kubernetes")
            run_migration
            deploy_kubernetes
            ;;
    esac
    
    verify_deployment
    show_status
    
    log_success "Deployment completed successfully!"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "migrate")
        setup_environment
        run_migration
        ;;
    "verify")
        verify_deployment
        ;;
    "status")
        show_status
        ;;
    "cleanup")
        cleanup
        ;;
    "help")
        echo "Usage: $0 [deploy|migrate|verify|status|cleanup|help]"
        echo "  deploy  - Deploy the service (default)"
        echo "  migrate - Run database migration only"
        echo "  verify  - Verify deployment health"
        echo "  status  - Show deployment status"
        echo "  cleanup - Clean up deployment"
        echo "  help    - Show this help"
        echo ""
        echo "Environment variables:"
        echo "  DEPLOYMENT_TYPE - docker-compose or kubernetes (default: docker-compose)"
        echo "  ENVIRONMENT - development, staging, production (default: development)"
        echo "  VERSION - Image version to deploy (default: latest)"
        ;;
    *)
        log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac
