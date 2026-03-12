#!/bin/bash

# Safety Gateway Platform - Phase 2 Deployment Script
# Advanced Orchestration Enhancement Deployment

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOYMENT_ENV="${1:-development}"
PHASE="2"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Validation functions
validate_environment() {
    case "$DEPLOYMENT_ENV" in
        development|staging|production)
            log "Deploying to $DEPLOYMENT_ENV environment"
            ;;
        *)
            error "Invalid environment: $DEPLOYMENT_ENV. Must be one of: development, staging, production"
            ;;
    esac
}

validate_prerequisites() {
    log "Validating prerequisites for Phase 2 deployment..."
    
    # Check required tools
    local required_tools=("docker" "kubectl" "helm" "jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            error "Required tool '$tool' is not installed"
        fi
    done
    
    # Check Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        error "Docker daemon is not running"
    fi
    
    # Validate configuration files
    local config_files=(
        "${PROJECT_ROOT}/config.yaml"
        "${PROJECT_ROOT}/devops/k8s/overlays/phase2/safety-gateway-phase2-deployment.yaml"
        "${PROJECT_ROOT}/devops/k8s/overlays/phase2/safety-gateway-phase2-configmap.yaml"
    )
    
    for config_file in "${config_files[@]}"; do
        if [[ ! -f "$config_file" ]]; then
            error "Required configuration file not found: $config_file"
        fi
    done
    
    log "Prerequisites validation completed successfully"
}

validate_phase2_features() {
    log "Validating Phase 2 feature requirements..."
    
    # Check if Phase 2 features are properly configured
    local config_file="${PROJECT_ROOT}/config.yaml"
    
    # Validate advanced orchestration is enabled
    if ! grep -q "enabled: true" "$config_file" | head -1; then
        warn "Advanced orchestration may not be enabled in configuration"
    fi
    
    # Check resource requirements
    local min_memory_gb=2
    local min_cpu_cores=1
    
    info "Minimum resource requirements for Phase 2:"
    info "  Memory: ${min_memory_gb}GB per pod"
    info "  CPU: ${min_cpu_cores} cores per pod"
    info "  Storage: 5GB for logs and metrics"
    
    log "Phase 2 feature validation completed"
}

# Build functions
build_phase2_image() {
    log "Building Phase 2 Docker image..."
    
    cd "$PROJECT_ROOT"
    
    # Build arguments for Phase 2
    local build_args=(
        "--build-arg" "VERSION=v2.0.0"
        "--build-arg" "PHASE=2"
        "--build-arg" "BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        "--build-arg" "GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
        "--target" "production"
    )
    
    # Tag with Phase 2 version
    local image_tag="safety-gateway-platform:v2.0.0-phase2"
    local latest_tag="safety-gateway-platform:latest-phase2"
    
    docker build "${build_args[@]}" -t "$image_tag" -t "$latest_tag" .
    
    if [[ $? -eq 0 ]]; then
        log "Phase 2 Docker image built successfully: $image_tag"
    else
        error "Failed to build Phase 2 Docker image"
    fi
    
    # Push to registry if not development
    if [[ "$DEPLOYMENT_ENV" != "development" ]]; then
        push_image_to_registry "$image_tag"
    fi
}

push_image_to_registry() {
    local image_tag="$1"
    log "Pushing image to registry: $image_tag"
    
    # Configure registry based on environment
    case "$DEPLOYMENT_ENV" in
        staging)
            local registry="staging-registry.clinical-hub.com"
            ;;
        production)
            local registry="prod-registry.clinical-hub.com"
            ;;
        *)
            warn "Skipping registry push for development environment"
            return 0
            ;;
    esac
    
    # Tag for registry
    local registry_tag="${registry}/${image_tag}"
    docker tag "$image_tag" "$registry_tag"
    
    # Push to registry
    if docker push "$registry_tag"; then
        log "Image pushed successfully to registry: $registry_tag"
    else
        error "Failed to push image to registry"
    fi
}

# Database migration functions
run_phase2_migrations() {
    log "Running Phase 2 database migrations..."
    
    # Check if migration job already exists
    if kubectl get job safety-gateway-phase2-migration -n safety-gateway &> /dev/null; then
        warn "Migration job already exists, deleting..."
        kubectl delete job safety-gateway-phase2-migration -n safety-gateway --ignore-not-found=true
    fi
    
    # Create migration job
    cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: safety-gateway-phase2-migration
  namespace: safety-gateway
  labels:
    app: safety-gateway
    component: migration
    phase: "2"
spec:
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        app: safety-gateway
        component: migration
        phase: "2"
    spec:
      restartPolicy: Never
      containers:
      - name: migrate
        image: migrate/migrate:latest
        command: ["/migrate"]
        args:
          - "-path"
          - "/migrations/phase2"
          - "-database"
          - "\$(TIMESCALEDB_URL)"
          - "up"
        env:
        - name: TIMESCALEDB_URL
          valueFrom:
            secretKeyRef:
              name: safety-gateway-secrets
              key: timescaledb-url
        volumeMounts:
        - name: migrations
          mountPath: /migrations/phase2
          readOnly: true
      volumes:
      - name: migrations
        configMap:
          name: safety-gateway-phase2-migrations
EOF
    
    # Wait for migration to complete
    local timeout=300
    local elapsed=0
    local interval=10
    
    while [[ $elapsed -lt $timeout ]]; do
        local job_status=$(kubectl get job safety-gateway-phase2-migration -n safety-gateway -o jsonpath='{.status.conditions[0].type}' 2>/dev/null || echo "")
        
        if [[ "$job_status" == "Complete" ]]; then
            log "Database migration completed successfully"
            break
        elif [[ "$job_status" == "Failed" ]]; then
            error "Database migration failed"
        fi
        
        info "Waiting for migration to complete... ($elapsed/$timeout seconds)"
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    if [[ $elapsed -ge $timeout ]]; then
        error "Migration timed out after $timeout seconds"
    fi
}

# Kubernetes deployment functions
deploy_to_kubernetes() {
    log "Deploying Phase 2 to Kubernetes ($DEPLOYMENT_ENV)..."
    
    # Create namespace if it doesn't exist
    kubectl create namespace safety-gateway --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply ConfigMaps first
    kubectl apply -f "${PROJECT_ROOT}/devops/k8s/overlays/phase2/safety-gateway-phase2-configmap.yaml" -n safety-gateway
    
    # Apply secrets (they should already exist)
    if ! kubectl get secret safety-gateway-secrets -n safety-gateway &> /dev/null; then
        warn "Safety gateway secrets not found. Creating placeholder..."
        create_placeholder_secrets
    fi
    
    # Run migrations
    run_phase2_migrations
    
    # Apply deployment
    kubectl apply -f "${PROJECT_ROOT}/devops/k8s/overlays/phase2/safety-gateway-phase2-deployment.yaml" -n safety-gateway
    
    # Wait for deployment to be ready
    wait_for_deployment_ready
    
    # Verify deployment
    verify_phase2_deployment
}

create_placeholder_secrets() {
    log "Creating placeholder secrets for Phase 2..."
    
    kubectl create secret generic safety-gateway-secrets \
        --from-literal=timescaledb-url="postgresql://safety_user:placeholder@postgres:5432/safety_gateway" \
        --from-literal=redis-password="placeholder" \
        --from-literal=jwt-secret="placeholder-jwt-secret-change-in-production" \
        --from-literal=kafka-username="placeholder" \
        --from-literal=kafka-password="placeholder" \
        -n safety-gateway \
        --dry-run=client -o yaml | kubectl apply -f -
    
    warn "Placeholder secrets created. Update them with real values for production deployment!"
}

wait_for_deployment_ready() {
    log "Waiting for Phase 2 deployment to be ready..."
    
    # Wait for deployment rollout
    if kubectl rollout status deployment/safety-gateway-phase2 -n safety-gateway --timeout=600s; then
        log "Phase 2 deployment rolled out successfully"
    else
        error "Phase 2 deployment rollout failed or timed out"
    fi
    
    # Wait for pods to be ready
    local timeout=300
    local elapsed=0
    local interval=10
    
    while [[ $elapsed -lt $timeout ]]; do
        local ready_pods=$(kubectl get pods -n safety-gateway -l app=safety-gateway,version=v2.0.0 -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -o "True" | wc -l)
        local total_pods=$(kubectl get pods -n safety-gateway -l app=safety-gateway,version=v2.0.0 --no-headers | wc -l)
        
        if [[ $ready_pods -eq $total_pods ]] && [[ $total_pods -gt 0 ]]; then
            log "All Phase 2 pods are ready ($ready_pods/$total_pods)"
            break
        fi
        
        info "Waiting for pods to be ready... ($ready_pods/$total_pods ready, $elapsed/$timeout seconds)"
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    if [[ $elapsed -ge $timeout ]]; then
        error "Pods did not become ready within $timeout seconds"
    fi
}

verify_phase2_deployment() {
    log "Verifying Phase 2 deployment..."
    
    # Check pod status
    local pod_count=$(kubectl get pods -n safety-gateway -l app=safety-gateway,version=v2.0.0 --no-headers | wc -l)
    if [[ $pod_count -eq 0 ]]; then
        error "No Phase 2 pods found"
    fi
    
    log "Found $pod_count Phase 2 pods"
    
    # Check service endpoints
    local service_ip=$(kubectl get service safety-gateway-phase2-service -n safety-gateway -o jsonpath='{.spec.clusterIP}')
    if [[ -z "$service_ip" ]]; then
        error "Phase 2 service not found or has no ClusterIP"
    fi
    
    log "Phase 2 service available at: $service_ip"
    
    # Health check
    local health_check_timeout=60
    local health_check_elapsed=0
    local health_check_interval=5
    
    while [[ $health_check_elapsed -lt $health_check_timeout ]]; do
        if kubectl exec -n safety-gateway deployment/safety-gateway-phase2 -- curl -f http://localhost:8033/health/ready &> /dev/null; then
            log "Phase 2 health check passed"
            break
        fi
        
        info "Waiting for health check to pass... ($health_check_elapsed/$health_check_timeout seconds)"
        sleep $health_check_interval
        health_check_elapsed=$((health_check_elapsed + health_check_interval))
    done
    
    if [[ $health_check_elapsed -ge $health_check_timeout ]]; then
        warn "Health check did not pass within $health_check_timeout seconds"
    fi
    
    # Check Phase 2 specific endpoints
    verify_phase2_endpoints
}

verify_phase2_endpoints() {
    log "Verifying Phase 2 specific endpoints..."
    
    local endpoints=(
        "/api/v1/batch/validate"
        "/api/v1/orchestration/stats"
        "/api/v1/orchestration/metrics"
        "/api/v1/health/orchestration"
    )
    
    local pod_name=$(kubectl get pods -n safety-gateway -l app=safety-gateway,version=v2.0.0 -o jsonpath='{.items[0].metadata.name}')
    
    for endpoint in "${endpoints[@]}"; do
        if kubectl exec -n safety-gateway "$pod_name" -- curl -f "http://localhost:8031$endpoint" &> /dev/null; then
            log "✓ Phase 2 endpoint available: $endpoint"
        else
            warn "✗ Phase 2 endpoint not available: $endpoint"
        fi
    done
}

# Docker Compose deployment functions
deploy_with_docker_compose() {
    log "Deploying Phase 2 with Docker Compose ($DEPLOYMENT_ENV)..."
    
    cd "$PROJECT_ROOT"
    
    # Set environment variables for Docker Compose
    export COMPOSE_PROJECT_NAME="safety-gateway-phase2"
    export DEPLOYMENT_ENV="$DEPLOYMENT_ENV"
    
    # Environment-specific compose files
    local compose_files=("-f" "docker-compose.phase2.yml")
    
    case "$DEPLOYMENT_ENV" in
        development)
            compose_files+=("-f" "docker-compose.phase2.override.yml")
            ;;
        staging)
            compose_files+=("-f" "docker-compose.phase2.staging.yml")
            ;;
        production)
            compose_files+=("-f" "docker-compose.phase2.production.yml")
            ;;
    esac
    
    # Pull latest images (except for development)
    if [[ "$DEPLOYMENT_ENV" != "development" ]]; then
        docker-compose "${compose_files[@]}" pull
    fi
    
    # Start services
    docker-compose "${compose_files[@]}" up -d
    
    # Wait for services to be ready
    wait_for_docker_services
    
    # Verify deployment
    verify_docker_deployment
}

wait_for_docker_services() {
    log "Waiting for Docker services to be ready..."
    
    local services=("postgres" "redis" "kafka" "safety-gateway")
    local timeout=300
    local elapsed=0
    local interval=10
    
    while [[ $elapsed -lt $timeout ]]; do
        local ready_services=0
        
        for service in "${services[@]}"; do
            if docker-compose -f docker-compose.phase2.yml ps "$service" | grep -q "Up (healthy)"; then
                ready_services=$((ready_services + 1))
            fi
        done
        
        if [[ $ready_services -eq ${#services[@]} ]]; then
            log "All Docker services are ready"
            break
        fi
        
        info "Waiting for services to be ready... ($ready_services/${#services[@]} ready, $elapsed/$timeout seconds)"
        sleep $interval
        elapsed=$((elapsed + interval))
    done
    
    if [[ $elapsed -ge $timeout ]]; then
        warn "Some services may not be ready after $timeout seconds"
    fi
}

verify_docker_deployment() {
    log "Verifying Docker deployment..."
    
    # Check if safety-gateway container is running
    if ! docker-compose -f docker-compose.phase2.yml ps safety-gateway | grep -q "Up"; then
        error "Safety Gateway container is not running"
    fi
    
    # Health check
    local container_name="safety-gateway-platform"
    local health_url="http://localhost:8032/health"
    
    if curl -f "$health_url" &> /dev/null; then
        log "Phase 2 Docker deployment health check passed"
    else
        warn "Phase 2 Docker deployment health check failed"
    fi
    
    # Show service URLs
    info "Phase 2 Services:"
    info "  Safety Gateway API: http://localhost:8030"
    info "  Batch Processing API: http://localhost:8031"
    info "  Health Check: http://localhost:8032/health"
    info "  Orchestration Management: http://localhost:8034"
    info "  Metrics: http://localhost:9091/metrics"
    info "  Grafana Dashboard: http://localhost:3000"
    info "  Prometheus: http://localhost:9090"
}

# Testing functions
run_phase2_tests() {
    log "Running Phase 2 integration tests..."
    
    case "$DEPLOYMENT_TARGET" in
        kubernetes)
            run_kubernetes_tests
            ;;
        docker-compose)
            run_docker_tests
            ;;
        *)
            warn "Skipping tests for unknown deployment target: $DEPLOYMENT_TARGET"
            ;;
    esac
}

run_kubernetes_tests() {
    log "Running Kubernetes-based Phase 2 tests..."
    
    # Create test job
    kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: safety-gateway-phase2-tests
  namespace: safety-gateway
spec:
  ttlSecondsAfterFinished: 600
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: tests
        image: safety-gateway-platform:v2.0.0-phase2
        command: ["/bin/sh", "-c"]
        args:
          - |
            echo "Running Phase 2 integration tests..."
            go test -v ./tests/integration/phase2_orchestration_test.go -count=1 -race
        env:
        - name: API_BASE_URL
          value: "http://safety-gateway-phase2-service:8030"
        - name: BATCH_API_URL
          value: "http://safety-gateway-phase2-service:8031"
        - name: TEST_ENVIRONMENT
          value: "$DEPLOYMENT_ENV"
EOF
    
    # Wait for test completion
    kubectl wait --for=condition=complete job/safety-gateway-phase2-tests -n safety-gateway --timeout=600s
    
    # Get test results
    kubectl logs job/safety-gateway-phase2-tests -n safety-gateway
}

run_docker_tests() {
    log "Running Docker-based Phase 2 tests..."
    
    # Run tests using Docker Compose
    docker-compose -f docker-compose.phase2.yml --profile testing up --abort-on-container-exit api-tests
    
    # Check test results
    local test_exit_code=$(docker-compose -f docker-compose.phase2.yml ps -q api-tests | xargs docker inspect --format='{{.State.ExitCode}}')
    
    if [[ "$test_exit_code" == "0" ]]; then
        log "Phase 2 tests passed successfully"
    else
        error "Phase 2 tests failed with exit code: $test_exit_code"
    fi
}

# Monitoring setup
setup_monitoring() {
    log "Setting up Phase 2 monitoring..."
    
    case "$DEPLOYMENT_TARGET" in
        kubernetes)
            setup_kubernetes_monitoring
            ;;
        docker-compose)
            setup_docker_monitoring
            ;;
    esac
}

setup_kubernetes_monitoring() {
    log "Setting up Kubernetes monitoring for Phase 2..."
    
    # Apply ServiceMonitor for Prometheus
    kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: safety-gateway-phase2
  namespace: safety-gateway
  labels:
    app: safety-gateway
    phase: "2"
spec:
  selector:
    matchLabels:
      app: safety-gateway
      version: v2.0.0
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
  - port: metrics-node
    interval: 30s
    path: /metrics
EOF
    
    log "Phase 2 Kubernetes monitoring configured"
}

setup_docker_monitoring() {
    log "Setting up Docker monitoring for Phase 2..."
    
    # Monitoring is configured in docker-compose.phase2.yml
    # Just verify that monitoring services are running
    
    local monitoring_services=("prometheus" "grafana")
    for service in "${monitoring_services[@]}"; do
        if docker-compose -f docker-compose.phase2.yml ps "$service" | grep -q "Up"; then
            log "✓ Monitoring service running: $service"
        else
            warn "✗ Monitoring service not running: $service"
        fi
    done
}

# Cleanup functions
cleanup_failed_deployment() {
    warn "Cleaning up failed deployment..."
    
    case "$DEPLOYMENT_TARGET" in
        kubernetes)
            kubectl delete deployment safety-gateway-phase2 -n safety-gateway --ignore-not-found=true
            kubectl delete job safety-gateway-phase2-migration -n safety-gateway --ignore-not-found=true
            kubectl delete job safety-gateway-phase2-tests -n safety-gateway --ignore-not-found=true
            ;;
        docker-compose)
            docker-compose -f docker-compose.phase2.yml down -v
            ;;
    esac
    
    warn "Failed deployment cleanup completed"
}

# Main deployment function
main() {
    log "Starting Safety Gateway Platform Phase 2 deployment..."
    log "Environment: $DEPLOYMENT_ENV"
    log "Phase: $PHASE"
    
    # Determine deployment target
    DEPLOYMENT_TARGET="${2:-kubernetes}"
    log "Deployment target: $DEPLOYMENT_TARGET"
    
    # Trap for cleanup on failure
    trap cleanup_failed_deployment ERR
    
    # Validation phase
    validate_environment
    validate_prerequisites
    validate_phase2_features
    
    # Build phase
    build_phase2_image
    
    # Deployment phase
    case "$DEPLOYMENT_TARGET" in
        kubernetes|k8s)
            deploy_to_kubernetes
            ;;
        docker-compose|compose|docker)
            deploy_with_docker_compose
            ;;
        *)
            error "Invalid deployment target: $DEPLOYMENT_TARGET. Must be one of: kubernetes, docker-compose"
            ;;
    esac
    
    # Testing phase
    if [[ "${RUN_TESTS:-true}" == "true" ]]; then
        run_phase2_tests
    fi
    
    # Monitoring setup
    setup_monitoring
    
    # Success message
    log "Phase 2 deployment completed successfully!"
    log "Deployment environment: $DEPLOYMENT_ENV"
    log "Deployment target: $DEPLOYMENT_TARGET"
    
    # Display next steps
    info "Next steps:"
    info "1. Verify deployment health using the monitoring dashboards"
    info "2. Run additional load tests if needed"
    info "3. Configure alerting rules for production monitoring"
    info "4. Update DNS/ingress configuration for external access"
    
    case "$DEPLOYMENT_TARGET" in
        kubernetes)
            info "5. Access services via kubectl port-forward if needed"
            info "   kubectl port-forward -n safety-gateway service/safety-gateway-phase2-service 8030:8030"
            ;;
        docker-compose)
            info "5. Services are available at the following URLs:"
            info "   - Main API: http://localhost:8030"
            info "   - Batch API: http://localhost:8031"
            info "   - Monitoring: http://localhost:3000"
            ;;
    esac
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if help is requested
    if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
        cat <<EOF
Safety Gateway Platform - Phase 2 Deployment Script

Usage: $0 <environment> [deployment-target]

Arguments:
  environment        Deployment environment (development|staging|production)
  deployment-target  Deployment target (kubernetes|docker-compose) [default: kubernetes]

Environment Variables:
  RUN_TESTS         Run integration tests after deployment (default: true)

Examples:
  $0 development kubernetes
  $0 staging docker-compose
  $0 production
  
  RUN_TESTS=false $0 development kubernetes

EOF
        exit 0
    fi
    
    # Execute main function
    main "$@"
fi